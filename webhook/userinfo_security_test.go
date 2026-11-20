// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package webhook_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/webhook"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"go.uber.org/zap/zaptest"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestUserInfoMutatorReplacesUntrustedIdentity(t *testing.T) {
	for _, kind := range []string{v1beta1.DisruptionKind, v1beta1.DisruptionCronKind} {
		for _, annotation := range []struct {
			name  string
			value string
		}{
			{name: "forged", value: `{"username":"forged-user","groups":["operators"]}`},
			{name: "extra fields", value: `{"username":"forged-user","groups":["operators"],"untrusted":true}`},
			{name: "malformed", value: `{"groups":`},
			{name: "empty", value: ""},
		} {
			t.Run(kind+"/"+annotation.name, func(t *testing.T) {
				object := &metav1.PartialObjectMetadata{
					TypeMeta: metav1.TypeMeta{APIVersion: v1beta1.GroupVersion.String(), Kind: kind},
					ObjectMeta: metav1.ObjectMeta{
						Name: "identity-test",
						Annotations: map[string]string{
							"UserInfo":              annotation.value,
							"example.com/preserved": "value",
						},
					},
				}
				raw, err := json.Marshal(object)
				if err != nil {
					t.Fatal(err)
				}
				identity := authv1.UserInfo{Username: "request-user", Groups: []string{"readers"}}
				mutator := webhook.UserInfoMutator{
					Log: zaptest.NewLogger(t).Sugar(), Decoder: admission.NewDecoder(runtime.NewScheme()),
				}
				response := mutator.Handle(context.Background(), admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:     metav1.GroupVersionKind{Kind: kind},
					UserInfo: identity, Object: runtime.RawExtension{Raw: raw},
				}})
				if !response.Allowed {
					t.Fatalf("expected identity overwrite, got %#v", response.Result)
				}
				patchBytes, err := json.Marshal(response.Patches)
				if err != nil {
					t.Fatal(err)
				}
				patch, err := jsonpatch.DecodePatch(patchBytes)
				if err != nil {
					t.Fatal(err)
				}
				mutated, err := patch.Apply(raw)
				if err != nil {
					t.Fatal(err)
				}
				var result metav1.PartialObjectMetadata
				if err := json.Unmarshal(mutated, &result); err != nil {
					t.Fatal(err)
				}
				want, err := json.Marshal(identity)
				if err != nil {
					t.Fatal(err)
				}
				if got := result.Annotations["UserInfo"]; got != string(want) {
					t.Fatalf("expected request identity %s, got %s", want, got)
				}
				if result.Annotations["example.com/preserved"] != "value" {
					t.Fatal("unrelated annotation was changed")
				}
			})
		}
	}
}

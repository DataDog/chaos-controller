// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
)

// Exercise AdmissionReview decoding, the real update validator, and admission
// responses without requiring a Kubernetes API server or deploying a controller.
func TestUserInfoAdmission(t *testing.T) {
	oldLogger, oldMetrics := logger, metricsSink
	t.Cleanup(func() { logger, metricsSink = oldLogger, oldMetrics })
	logger = zaptest.NewLogger(t).Sugar()
	metricsSink = noop.New(logger)
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	webhook := admission.WithValidator[*Disruption](scheme, &Disruption{})

	for _, tt := range []struct {
		name, oldInfo, newInfo string
		changeCount, allowed   bool
	}{
		{"groups merged", `{"groups":["a","b"]}`, `{"groups":["a b"]}`, false, false},
		{"groups split", `{"groups":["a b"]}`, `{"groups":["a","b"]}`, false, false},
		{"extra collision", `{"extra":{"scope":["read","write"]}}`, `{"extra":{"scope":["read write"]}}`, false, false},
		{"no-op", `{"groups":["a","b"]}`, `{"groups":["a","b"]}`, false, true},
		{"count update", `{"groups":["a","b"]}`, `{"groups":["a","b"]}`, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			oldObject := &Disruption{
				TypeMeta:   TypeMeta,
				ObjectMeta: metav1.ObjectMeta{Name: "identity-test", Namespace: "default", Annotations: map[string]string{annotationUserInfoKey: tt.oldInfo}},
				Spec: DisruptionSpec{
					Count:    &intstr.IntOrString{IntVal: 1},
					Selector: map[string]string{"app": "identity-test"},
					Network:  &NetworkDisruptionSpec{Drop: 100, Hosts: []NetworkDisruptionHostSpec{{Port: 80}}},
				},
			}
			newObject := oldObject.DeepCopy()
			newObject.Annotations[annotationUserInfoKey] = tt.newInfo
			if tt.changeCount {
				newObject.Spec.Count = &intstr.IntOrString{IntVal: 2}
			}
			marshal := func(value any) []byte {
				t.Helper()
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			review := admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
				Request: &admissionv1.AdmissionRequest{
					UID: "identity-test", Operation: admissionv1.Update,
					Kind:      metav1.GroupVersionKind{Group: GroupVersion.Group, Version: GroupVersion.Version, Kind: DisruptionKind},
					Resource:  metav1.GroupVersionResource{Group: GroupVersion.Group, Version: GroupVersion.Version, Resource: "disruptions"},
					Object:    runtime.RawExtension{Raw: marshal(newObject)},
					OldObject: runtime.RawExtension{Raw: marshal(oldObject)},
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(marshal(review)))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			webhook.ServeHTTP(response, request)
			// Kubernetes webhook transports return HTTP 200 even when admission is denied.
			if response.Code != http.StatusOK {
				t.Fatalf("HTTP status %d: %s", response.Code, response.Body)
			}
			var result admissionv1.AdmissionReview
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Response == nil {
				t.Fatal("missing admission response")
			}
			if result.Response.UID != review.Request.UID || result.Response.Allowed != tt.allowed {
				t.Fatalf("unexpected admission response: %+v", result.Response)
			}
			if !tt.allowed {
				status := result.Response.Result
				if status == nil || status.Code != http.StatusForbidden || status.Message != "the user info annotation is immutable" {
					t.Fatalf("unexpected denial status: %+v", status)
				}
			}
		})
	}
}

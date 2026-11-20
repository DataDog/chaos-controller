// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package webhook_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/webhook"
)

var _ = Describe("SpanContextMutator", func() {
	var (
		mutator *webhook.SpanContextMutator
	)

	BeforeEach(func() {
		mutator = &webhook.SpanContextMutator{
			Log:     zaptest.NewLogger(GinkgoT()).Sugar(),
			Decoder: admission.NewDecoder(runtime.NewScheme()),
		}
	})

	Describe("Handle", func() {
		Context("nil decoder", func() {
			It("returns 500", func() {
				mutator.Decoder = nil
				resp := mutator.Handle(context.TODO(), admission.Request{})
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Code).To(Equal(int32(http.StatusInternalServerError)))
			})
		})

		Context("malformed raw object", func() {
			It("returns 400", func() {
				req := admission.Request{
					AdmissionRequest: v1.AdmissionRequest{
						Object: runtime.RawExtension{Raw: []byte("not-valid-json{")},
					},
				}
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Code).To(Equal(int32(http.StatusBadRequest)))
			})
		})

		Context("valid disruption object", func() {
			It("returns allowed patch response", func() {
				dis := &v1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "test-disruption",
						Namespace:   "default",
						Annotations: map[string]string{"existing-key": "existing-value"},
					},
				}
				raw, err := json.Marshal(dis)
				Expect(err).NotTo(HaveOccurred())

				req := admission.Request{
					AdmissionRequest: v1.AdmissionRequest{
						Name:      "test-disruption",
						Namespace: "default",
						UserInfo:  authv1.UserInfo{Username: "test-user"},
						Object:    runtime.RawExtension{Raw: raw},
					},
				}
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})
	})
})

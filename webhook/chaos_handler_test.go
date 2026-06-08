// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package webhook_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/DataDog/chaos-controller/webhook"
)

func makePodRequest(pod *corev1.Pod, annotations map[string]string) admission.Request {
	pod.Annotations = annotations
	raw, _ := json.Marshal(pod)
	return admission.Request{
		AdmissionRequest: v1.AdmissionRequest{
			Object: runtime.RawExtension{Raw: raw},
		},
	}
}

var _ = Describe("ChaosHandlerMutator", func() {
	var (
		mutator      *webhook.ChaosHandlerMutator
		resourceList corev1.ResourceList
		scheme       *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		resourceList = corev1.ResourceList{}
		mutator = &webhook.ChaosHandlerMutator{
			Log:          zaptest.NewLogger(GinkgoT()).Sugar(),
			Image:        "chaos-handler:latest",
			Timeout:      30 * time.Second,
			MaxTimeout:   60 * time.Second,
			Decoder:      admission.NewDecoder(scheme),
			ResourceList: &resourceList,
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

		Context("valid pod, no annotations", func() {
			It("returns allowed patch response with chaos-handler init container", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
				req := makePodRequest(pod, nil)
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})

		Context("pod with GenerateName (no Name)", func() {
			It("returns allowed using GenerateName for logging", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"}}
				req := makePodRequest(pod, nil)
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})

		Context("valid timeout annotation within MaxTimeout", func() {
			It("uses custom timeout and returns allowed", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
				req := makePodRequest(pod, map[string]string{
					"chaos.datadoghq.com/disrupt-on-init-timeout": "20s",
				})
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})

		Context("timeout annotation exceeds MaxTimeout", func() {
			It("returns 400", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
				req := makePodRequest(pod, map[string]string{
					"chaos.datadoghq.com/disrupt-on-init-timeout": "120s",
				})
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Code).To(Equal(int32(http.StatusBadRequest)))
			})
		})

		Context("invalid timeout annotation format", func() {
			It("falls through to default timeout and returns allowed", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
				req := makePodRequest(pod, map[string]string{
					"chaos.datadoghq.com/disrupt-on-init-timeout": "not-a-duration",
				})
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})

		Context("succeed-on-timeout annotation present", func() {
			It("includes --succeed-on-timeout flag and returns allowed", func() {
				pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
				req := makePodRequest(pod, map[string]string{
					"chaos.datadoghq.com/disrupt-on-init-succeed-on-timeout": "true",
				})
				resp := mutator.Handle(context.TODO(), req)
				Expect(resp.Allowed).To(BeTrue())
			})
		})
	})
})

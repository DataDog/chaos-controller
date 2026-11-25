// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package webhook_test

import (
	"context"
	"net/http"
	"time"

	"github.com/DataDog/chaos-controller/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("Chaos handler nonpositive timeout admission", func() {
	DescribeTable("requires a positive explicit timeout", func(value string, allowed bool) {
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		resources := corev1.ResourceList{}
		mutator := webhook.ChaosHandlerMutator{
			Log: zap.NewNop().Sugar(), Image: "handler:test",
			Timeout: 30 * time.Second, MaxTimeout: time.Minute,
			Decoder: admission.NewDecoder(scheme), ResourceList: &resources,
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "target"}}
		request := makePodRequest(pod, map[string]string{
			"chaos.datadoghq.com/disrupt-on-init-timeout": value,
		})
		response := mutator.Handle(context.Background(), request)
		Expect(response.Allowed).To(Equal(allowed))
		if !allowed {
			Expect(response.Result).NotTo(BeNil())
			Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)))
		}
	},
		Entry("zero", "0", false),
		Entry("zero with units", "0s", false),
		Entry("negative nanosecond", "-1ns", false),
		Entry("negative minute", "-1m", false),
		Entry("positive override", "20s", true),
		Entry("maximum timeout", "1m", true),
	)
})

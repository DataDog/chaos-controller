// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disruption Webhook count update", func() {
	It("runs the count safety net when an update raises the count", func() {
		defaultNamespaceThreshold = 0.80
		defaultClusterThreshold = 0.66
		enableSafemode = true
		metricsSink = metricsnoop.New(logger)
		DeferCleanup(func() { enableSafemode = false; apiReader = nil; k8sClient = nil })

		pods := []client.Object{}
		for i := range 5 {
			pods = append(pods, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", i), Namespace: chaosNamespace, Labels: map[string]string{"app": "target"}},
				Spec:       v1.PodSpec{Containers: []v1.Container{{Image: "pause", Name: "ctn"}}},
			})
		}
		apiReader = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(pods...).Build()
		k8sClient = apiReader.(client.Client)

		mk := func(count int) *Disruption {
			return &Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "d", Namespace: chaosNamespace},
				Spec: DisruptionSpec{
					Level: chaostypes.DisruptionLevelPod, Count: &intstr.IntOrString{IntVal: int32(count)},
					Selector: labels.Set{"app": "target"}, Duration: "1h",
					CPUPressure: &CPUPressureSpec{},
				},
			}
		}
		caught, _, err := safetyNetCountNotTooLarge(context.Background(), mk(5))
		Expect(err).NotTo(HaveOccurred())
		Expect(caught).To(BeTrue(), "precondition: creating with count 5 of 5 is refused")

		old, updated := mk(1), mk(5)
		_, err = updated.ValidateUpdate(context.Background(), old, updated)
		Expect(err).To(HaveOccurred(), "update from count 1 to 5 of 5 pods was admitted without the count safety net")
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package controllers

import (
	"fmt"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-fi-controller/types"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// expectChaosPod retrieves the list of created chaos pods depending on the
// given mode (inject or clean) and returns an error if it doesn't
// equal the given count
func expectChaosPod(mode chaostypes.PodMode, count int) error {
	l := corev1.PodList{}
	ls := labels.NewSelector()
	targetPodRequirement, _ := labels.NewRequirement(chaostypes.TargetPodLabel, selection.In, []string{"foo", "bar"})
	podModRequirement, _ := labels.NewRequirement(chaostypes.PodModeLabel, selection.Equals, []string{string(mode)})
	ls = ls.Add(*targetPodRequirement, *podModRequirement)
	if err := k8sClient.List(context.Background(), &l, &client.ListOptions{
		Namespace:     "default",
		LabelSelector: ls,
	}); err != nil {
		return fmt.Errorf("can't list chaos pods: %w", err)
	}
	if len(l.Items) != count {
		return fmt.Errorf("unexpected injection pods count: %d", len(l.Items))
	}

	return nil
}

var _ = Describe("Disruption Controller", func() {
	var disruption *chaosv1beta1.Disruption
	var count int

	BeforeEach(func() {
		count = 0
		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Count:    &count,
				Selector: map[string]string{"foo": "bar"},
				NetworkFailure: &chaosv1beta1.NetworkFailureSpec{
					Hosts:       []string{"127.0.0.1"},
					Port:        80,
					Probability: 0,
					Protocol:    "tcp",
				},
				NetworkLatency: &chaosv1beta1.NetworkLatencySpec{
					Delay: 1000,
					Hosts: []string{"10.0.0.0/8"},
				},
			},
		}

		// patch
		monkey.Patch(getContainerID, func(pod *corev1.Pod) (string, error) {
			return "666", nil
		})
	})

	AfterEach(func() {
		_ = k8sClient.Delete(context.Background(), disruption)
		monkey.UnpatchAll()
	})

	Context("nominal case", func() {
		It("should create the injection and cleanup pods", func() {
			By("Creating disruption resource")
			Expect(k8sClient.Create(context.Background(), disruption)).To(BeNil())

			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(chaostypes.PodModeInject, 4) }, timeout).Should(Succeed())

			By("Deleting the disruption resource")
			Expect(k8sClient.Delete(context.Background(), disruption)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(Succeed())

			By("Ensuring that the cleanup pod has been created")
			Eventually(func() error { return expectChaosPod(chaostypes.PodModeClean, 4) }, timeout).Should(Succeed())

			By("Simulating the completion of the cleanup pod by removing the finalizer")
			Eventually(func() error {
				if err := k8sClient.Get(context.Background(), instanceKey, disruption); err != nil {
					return err
				}
				disruption.ObjectMeta.Finalizers = []string{}
				return k8sClient.Update(context.Background(), disruption)
			}, timeout).Should(Succeed())

			By("Waiting for network failure resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})
})

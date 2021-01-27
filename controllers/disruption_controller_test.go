// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// listChaosPods returns all the chaos pods for the given instance and mode
func listChaosPods(instance *chaosv1beta1.Disruption) (corev1.PodList, error) {
	l := corev1.PodList{}
	ls := labels.NewSelector()
	instancePods := corev1.PodList{}

	// create requirements
	targetPodRequirement, _ := labels.NewRequirement(chaostypes.TargetLabel, selection.In, []string{"foo", "bar", "car", "far"})

	// add requirements to label selector
	ls = ls.Add(*targetPodRequirement)

	// get matching pods
	if err := k8sClient.List(context.Background(), &l, &client.ListOptions{
		Namespace:     "default",
		LabelSelector: ls,
	}); err != nil {
		return corev1.PodList{}, fmt.Errorf("can't list chaos pods: %w", err)
	}

	// filter to get only pods owned by the given instance
	for _, pod := range l.Items {
		if metav1.IsControlledBy(&pod, instance) {
			instancePods.Items = append(instancePods.Items, pod)
		}
	}

	return instancePods, nil
}

// expectChaosPod retrieves the list of created chaos pods related to the given and to the
// given mode (inject or clean) and returns an error if it doesn't
// equal the given count
func expectChaosPod(instance *chaosv1beta1.Disruption, count int) error {
	l, err := listChaosPods(instance)
	if err != nil {
		return err
	}

	// ensure count is correct
	if len(l.Items) != count {
		return fmt.Errorf("unexpected chaos pods count: expected %d, found %d", count, len(l.Items))
	}

	// ensure generated pods have the needed fields
	for _, p := range l.Items {
		if p.GenerateName == "" {
			return fmt.Errorf("GenerateName field can't be empty")
		}
		if p.Namespace != instance.Namespace {
			return fmt.Errorf("pod namesapce must match instance namespace")
		}
		if len(p.Spec.Containers[0].Args) == 0 {
			return fmt.Errorf("pod container args must be set")
		}
		if p.Spec.Containers[0].Image == "" {
			return fmt.Errorf("pod container image must be set")
		}
		if len(p.ObjectMeta.Finalizers) == 0 {
			return fmt.Errorf("pod finalizer must be set")
		}
	}

	return nil
}

var _ = Describe("Disruption Controller", func() {
	var disruption *chaosv1beta1.Disruption

	BeforeEach(func() {
		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Selector: map[string]string{"foo": "bar"},
				NodeFailure: &chaosv1beta1.NodeFailureSpec{
					Shutdown: false,
				},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts:          []string{"127.0.0.1"},
					Port:           80,
					Protocol:       "tcp",
					Drop:           0,
					Corrupt:        0,
					Delay:          1000,
					BandwidthLimit: 10000,
				},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{},
				DiskPressure: &chaosv1beta1.DiskPressureSpec{
					Path: "/mnt/foo",
					Throttling: chaosv1beta1.DiskPressureThrottlingSpec{
						ReadBytesPerSec: func() *int { i := int(1); return &i }(),
					},
				},
			},
		}
	})

	AfterEach(func() {
		// delete disruption resource
		_ = k8sClient.Delete(context.Background(), disruption)
	})

	JustBeforeEach(func() {
		By("Creating disruption resource")
		Expect(k8sClient.Create(context.Background(), disruption)).To(BeNil())
	})

	Context("target all pods", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "100%"}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the spec hash has been computed")
			Eventually(func() error {
				if err := k8sClient.Get(context.Background(), instanceKey, disruption); err != nil {
					return err
				}

				if disruption.Status.SpecHash == nil {
					return fmt.Errorf("instance spec hash has not been computed yet")
				}

				return nil
			}, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have been created")
			Eventually(func() error { return expectChaosPod(disruption, 16) }, timeout).Should(Succeed())

			By("Deleting the disruption resource")
			Expect(k8sClient.Delete(context.Background(), disruption)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have been deleted")
			Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout).Should(Succeed())

			By("Waiting for disruption to be removed")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})

	Context("target one pod only", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 4) }, timeout).Should(Succeed())

			By("Deleting the disruption resource")
			Expect(k8sClient.Delete(context.Background(), disruption)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have been deleted")
			Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout).Should(Succeed())

			By("Waiting for disruption resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})

	Context("target 70% of pods (3 pods out of 4)", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "70%"}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 12) }, timeout).Should(Succeed())

			By("Deleting the disruption resource")
			Expect(k8sClient.Delete(context.Background(), disruption)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have been deleted")
			Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout).Should(Succeed())

			By("Waiting for disruption resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})
})

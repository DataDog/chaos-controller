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
	"strings"

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

	// create requirements
	targetPodRequirement, _ := labels.NewRequirement(chaostypes.TargetLabel, selection.In, []string{"foo", "bar", "minikube"})
	disruptionNameRequirement, _ := labels.NewRequirement(chaostypes.DisruptionNameLabel, selection.Equals, []string{instance.Name})
	disruptionNamespaceRequirement, _ := labels.NewRequirement(chaostypes.DisruptionNamespaceLabel, selection.Equals, []string{instance.Namespace})

	// add requirements to label selector
	ls = ls.Add(*targetPodRequirement, *disruptionNamespaceRequirement, *disruptionNameRequirement)

	// get matching pods
	if err := k8sClient.List(context.Background(), &l, &client.ListOptions{
		LabelSelector: ls,
	}); err != nil {
		return corev1.PodList{}, fmt.Errorf("can't list chaos pods: %w", err)
	}

	return l, nil
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
		if len(p.Spec.Containers[0].Args) == 0 {
			return fmt.Errorf("pod container args must be set")
		}
		if p.Spec.Containers[0].Image == "" {
			return fmt.Errorf("pod container image must be set")
		}
		if len(p.ObjectMeta.Finalizers) == 0 {
			return fmt.Errorf("pod finalizer must be set")
		}

		// ensure pod container is running (not completed or failed)
		for _, status := range p.Status.ContainerStatuses {
			if status.State.Running == nil {
				return fmt.Errorf("pod container is not running")
			}
		}
	}

	return nil
}

// expectChaosInjectors retrieves the list of created chaos pods and confirms
// that the targeted containers are present
func expectChaosInjectors(instance *chaosv1beta1.Disruption, count int) error {
	injectors := 0

	// get chaos pods
	l, err := listChaosPods(instance)

	if err != nil {
		return err
	}

	// sum up injectors
	for _, p := range l.Items {
		args := p.Spec.Containers[0].Args
		for i, arg := range args {
			if arg == "--target-container-ids" {
				injectors += len(strings.Split(args[i+1], ","))
			}
		}
	}

	if injectors != count {
		return fmt.Errorf("incorrect number of targeted containers in spec: expected %d, found %d", count, injectors)
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
				DryRun:     true,
				Count:      &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Selector:   map[string]string{"foo": "bar"},
				Containers: []string{"ctn1"},
				Duration:   "10m",
				NodeFailure: &chaosv1beta1.NodeFailureSpec{
					Shutdown: false,
				},
				ContainerFailure: &chaosv1beta1.ContainerFailureSpec{
					Forced: false,
				},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop:           0,
					Corrupt:        0,
					Delay:          1000,
					BandwidthLimit: 10000,
				},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{},
				DiskPressure: &chaosv1beta1.DiskPressureSpec{
					Path: "/mnt/foo",
					Throttling: chaosv1beta1.DiskPressureThrottlingSpec{
						ReadBytesPerSec: func() *int { i := int(10000); return &i }(),
					},
				},
				DNS: []chaosv1beta1.HostRecordPair{
					{
						Hostname: "ctn",
						Record: chaosv1beta1.DNSRecord{
							Type:  "A",
							Value: "10.0.0.1, 10.0.0.2 , 10.0.0.3",
						},
					},
				},
				GRPC: &chaosv1beta1.GRPCDisruptionSpec{
					Port: 2000,
					Endpoints: []chaosv1beta1.EndpointAlteration{
						{
							TargetEndpoint:   "/chaos_dogfood.ChaosDogfood/order",
							ErrorToReturn:    "",
							OverrideToReturn: "{}",
							QueryPercent:     50,
						},
					},
				},
			},
		}
	})

	AfterEach(func() {
		// delete disruption resource
		_ = k8sClient.Delete(context.Background(), disruption)
		Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout*2).Should(Succeed())
		Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
	})

	JustBeforeEach(func() {
		By("Creating disruption resource and waiting for injection to be done")
		Expect(k8sClient.Create(context.Background(), disruption)).To(BeNil())

		Eventually(func() error {
			// retrieve the previously created disruption
			d := chaosv1beta1.Disruption{}
			if err := k8sClient.Get(context.Background(), instanceKey, &d); err != nil {
				return err
			}

			// check disruption injection status
			if d.Status.InjectionStatus != chaostypes.DisruptionInjectionStatusInjected {
				return fmt.Errorf("disruptions is not injected, current status is %s", d.Status.InjectionStatus)
			}

			return nil
		}, timeout).Should(Succeed())
	})

	Context("a node level test should pass", func() {
		BeforeEach(func() {
			disruption = &chaosv1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: chaosv1beta1.DisruptionSpec{
					DryRun:   false,
					Count:    &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					Selector: map[string]string{"kubernetes.io/hostname": "minikube"},
					Level:    chaostypes.DisruptionLevelNode,
					Network: &chaosv1beta1.NetworkDisruptionSpec{
						Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
							{
								Host:     "127.0.0.1",
								Port:     80,
								Protocol: "tcp",
							},
						},
						Drop:    0,
						Corrupt: 0,
						Delay:   1,
					},
				},
			}
		})

		It("should target the node", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 1) }, timeout).Should(Succeed())
		})
	})

	Context("disruption expires naturally", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun:     true,
				Count:      &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Selector:   map[string]string{"foo": "bar"},
				Containers: []string{"ctn1"},
				Duration:   "30s",
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop:           0,
					Corrupt:        0,
					Delay:          1000,
					BandwidthLimit: 10000,
				},
			}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the chaos pods have been created")
			Eventually(func() error { return expectChaosPod(disruption, 2) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have correct number of targeted containers")
			Expect(expectChaosInjectors(disruption, 2)).To(BeNil())

			By("Waiting for the disruption to expire naturally")
			Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout*2).Should(Succeed())

			By("Waiting for disruption to be removed")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})

	Context("target one pod and one container only", func() {
		It("should target all the selected pods", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 7) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have correct number of targeted containers")
			Expect(expectChaosInjectors(disruption, 7)).To(BeNil())
		})
	})

	Context("target all pods and one container", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "100%"}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the chaos pods have been created")
			Eventually(func() error { return expectChaosPod(disruption, 14) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have correct number of targeted containers")
			Expect(expectChaosInjectors(disruption, 14)).To(BeNil())
		})
	})

	Context("target 30% of pods (1 pod out of 2) and one container", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "30%"}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 7) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have correct number of targeted containers")
			Expect(expectChaosInjectors(disruption, 7)).To(BeNil())
		})
	})

	Context("target all pods and all containers by default", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "100%"}
			disruption.Spec.Containers = []string{}
		})

		It("should target all the selected pods", func() {
			By("Ensuring that the chaos pods have been created")
			Eventually(func() error { return expectChaosPod(disruption, 14) }, timeout).Should(Succeed())

			By("Ensuring that the chaos pods have correct number of targeted containers")
			Expect(expectChaosInjectors(disruption, 21)).To(BeNil())
		})
	})

	// NOTE: disabled until fixed
	// the feature is broken now that we moved all chaos pods into the same namespace
	// because we had to remove the owner reference on those pods, meaning that
	// the reconcile loop does not automatically trigger anymore on chaos pods events like a delete
	// Context("manually delete a chaos pod", func() {
	// 	It("should properly handle the chaos pod finalizer", func() {
	// 		By("Ensuring that the chaos pods have been created")
	// 		Eventually(func() error { return expectChaosPod(disruption, 5) }, timeout).Should(Succeed())

	// 		By("Listing chaos pods to pick one to delete")
	// 		chaosPods, err := listChaosPods(disruption)
	// 		Expect(err).To(BeNil())
	// 		chaosPod := chaosPods.Items[0]
	// 		chaosPodKey := types.NamespacedName{Namespace: chaosPod.Namespace, Name: chaosPod.Name}

	// 		By("Deleting one of the chaos pod")
	// 		Expect(k8sClient.Delete(context.Background(), &chaosPod)).To(BeNil())

	// 		By("Waiting for the chaos pod finalizer to be removed")
	// 		Eventually(func() error { return k8sClient.Get(context.Background(), chaosPodKey, &chaosPod) }, timeout).Should(MatchError(fmt.Sprintf("Pod \"%s\" not found", chaosPod.Name)))
	// 	})
	// })
})

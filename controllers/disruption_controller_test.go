// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const shortDisruptionDuration = "3m" // somehow short, keep in mind the CI is slow AND reconcile loop is sequential, it should not be too short to avoid irrelevant failures

var _ = Describe("Disruption Controller", func() {
	var (
		targetPod, anotherTargetPod corev1.Pod
		disruption                  chaosv1beta1.Disruption
		skipSecondPod               bool
		expectedDisruptionStatus    chaostypes.DisruptionInjectionStatus
	)

	BeforeEach(func(ctx SpecContext) {
		disruption = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				DryRun: true,
				Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Containers: []string{"ctn1"},
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
							TargetEndpoint:   "/chaosdogfood.ChaosDogfood/order",
							ErrorToReturn:    "",
							OverrideToReturn: "{}",
							QueryPercent:     50,
						},
					},
				},
			},
		}

		skipSecondPod = false
		expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusInjected
	})

	JustBeforeEach(func(ctx SpecContext) {
		By("Creating disruption resource and waiting for injection to be done")
		disruption, targetPod, anotherTargetPod = InjectPodsAndDisruption(ctx, disruption, skipSecondPod)
		ExpectDisruptionStatus(ctx, disruption, expectedDisruptionStatus)
	})

	Context("annotation filters should limit selected targets", func() {
		BeforeEach(func() {
			disruption.Spec.Filter = &chaosv1beta1.DisruptionFilter{
				Annotations: targetPod.Annotations,
			}
		})

		It("should only select half of all pods", func(ctx SpecContext) {
			ExpectChaosPods(ctx, disruption, 4)
		})
	})

	Context("node level", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun: false,
				Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Selector: map[string]string{"kubernetes.io/hostname": clusterName},
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
			}
		})

		It("should target the node", func(ctx SpecContext) {
			By("Ensuring that the inject pod has been created")
			ExpectChaosPods(ctx, disruption, 1)
		})
	})

	Context("disruption expires naturally", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun: true,
				Count:  &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Containers:  []string{"ctn1"},
				Duration:    shortDisruptionDuration,
				CPUPressure: &chaosv1beta1.CPUPressureSpec{},
			}
		})

		It("should target all the selected pods", func(ctx SpecContext) {
			Concurrently{
				func(ctx SpecContext) {
					By("Ensuring that the chaos pods have been created")
					ExpectChaosPods(ctx, disruption, 2)
				},
				func(ctx SpecContext) {
					By("Ensuring that the chaos pods have correct number of targeted containers")
					ExpectChaosInjectors(ctx, disruption, 2)
				},
			}.DoAndWait(ctx)

			Concurrently{
				func(ctx SpecContext) {
					By("Waiting for the disruption to expire naturally")
					ExpectChaosPods(ctx, disruption, 0)
				},
				func(ctx SpecContext) {
					By("Waiting for the disruption to reach PreviouslyInjected")
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
				},
				func(ctx SpecContext) {
					By("Waiting for disruption to be removed")
					Eventually(k8sClient.Get).
						WithContext(ctx).WithArguments(
						types.NamespacedName{
							Namespace: disruption.Namespace,
							Name:      disruption.Name,
						}, &chaosv1beta1.Disruption{}).
						Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).
						Should(WithTransform(apierrors.IsNotFound, BeTrue()))
				},
			}.DoAndWait(ctx)
		})
	})

	Context("target one pod and one container only", func() {
		It("should target all the selected pods", func(ctx SpecContext) {
			By("Ensuring that the inject pod has been created")
			ExpectChaosPods(ctx, disruption, 4)

			By("Ensuring that the chaos pods have correct number of targeted containers")
			ExpectChaosInjectors(ctx, disruption, 4)
		})
	})

	Context("target all pods and one container", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "100%"}
		})

		It("should target all the selected pods", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 8)

			By("Ensuring that the chaos pods have correct number of targeted containers")
			ExpectChaosInjectors(ctx, disruption, 8)
		})
	})

	Context("target 30% of pods (1 pod out of 2) and one container", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "30%"}
		})

		It("should target all the selected pods", func(ctx SpecContext) {
			By("Ensuring that the inject pod has been created")
			ExpectChaosPods(ctx, disruption, 4)

			By("Ensuring that the chaos pods have correct number of targeted containers")
			ExpectChaosInjectors(ctx, disruption, 4)
		})
	})

	Context("target all pods and all containers by default", func() {
		BeforeEach(func() {
			disruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "100%"}
			disruption.Spec.Containers = []string{}
			disruption.Spec.Duration = "2m"
		})

		It("should target all the selected pods", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 8)

			By("Ensuring that the chaos pods have correct number of targeted containers")
			ExpectChaosInjectors(ctx, disruption, 12)
		})
	})

	Context("Dynamic targeting", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				StaticTargeting: false,
				DryRun:          true,
				Count:           &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Containers: []string{"ctn1"},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop: 100,
				},
			}
		})

		It("should scale up then down properly", func(ctx SpecContext) {
			ExpectChaosPodsAndStatuses := func(ctx SpecContext, count int) {
				GinkgoHelper()

				Concurrently{
					func(ctx SpecContext) {
						By("Ensuring that the chaos pods have been created")
						ExpectChaosPods(ctx, disruption, count)
					},
					func(ctx SpecContext) {
						By("Ensuring that the chaos pods have correct number of targeted containers")
						ExpectChaosInjectors(ctx, disruption, count)
					},
					func(ctx SpecContext) {
						By("Ensuring that the disruption status is displaying the right number of targets")
						ExpectDisruptionStatusCounts(ctx, disruption, count, 0, count, count)
					},
				}.DoAndWait(ctx)
			}

			ExpectChaosPodsAndStatuses(ctx, 2)

			By("Adding an extra target")
			extraPod := <-CreateRunningPod(ctx, *targetPod.DeepCopy())

			ExpectChaosPodsAndStatuses(ctx, 3)

			By("Deleting the extra target")
			DeleteRunningPod(ctx, extraPod)

			ExpectChaosPodsAndStatuses(ctx, 2)
		})
	})

	Context("On init", func() {
		BeforeEach(func(ctx SpecContext) {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun:   true,
				Duration: shortDisruptionDuration,
				Count:    &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Selector: map[string]string{"foo-foo": "bar-bar"},
				OnInit:   true,
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop: 100,
				},
			}

			// we don't need any pod, at least let's not create the first one...
			skipSecondPod = true
			expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusNotInjected
		})

		It("should keep on init target pods throughout reconcile loop", func(ctx SpecContext) {
			By("Ensuring that the on init target is ready and still targeted")
			initPodCreated := CreateRunningPod(
				ctx,
				corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "on-init-gen-",
						Namespace:    namespace,
						Labels: map[string]string{
							"foo-foo":                     "bar-bar",
							chaostypes.DisruptOnInitLabel: "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image: "k8s.gcr.io/pause:3.4.1",
								Name:  "ctn1",
							},
						},
					},
				},
			)

			// We create a pod and the disruption concurrently
			// We expect the pod to be running at some point (thanks to the disruption)
			Concurrently{
				func(ctx SpecContext) {
					// In order to reach the status running, the disruption NEEDS to have a valid effect
					// Otherwise the pod will stay stuck in init phase with the chaos handler container waiting
					Expect(<-initPodCreated).ToNot(BeZero())
				},
				func(sc SpecContext) {
					Eventually(func(ctx SpecContext) error {
						podList := corev1.PodList{}
						err := k8sClient.List(ctx, &podList, &client.ListOptions{
							LabelSelector: disruption.Spec.Selector.AsSelector(),
						})
						if err != nil {
							return fmt.Errorf("unable to target pods with selector: %w", err)
						}

						if len(podList.Items) == 0 {
							return fmt.Errorf("no target found")
						}

						for _, ctn := range podList.Items[0].Status.InitContainerStatuses {
							if ctn.State.Running != nil {
								return fmt.Errorf("chaos-handler container is still running")
							}
						}

						for _, ctn := range podList.Items[0].Status.ContainerStatuses {
							if !ctn.Ready {
								return fmt.Errorf("container %s is not ready", ctn.Name)
							}
						}

						return nil
					}).WithContext(ctx).ProbeEvery(disruptionPotentialChangesEvery).Within(calcDisruptionGoneTimeout(disruption)).Should(Succeed())
				},
			}.DoAndWait(ctx)
		})
	})

	Context("Target injections count", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				StaticTargeting: false,
				DryRun:          true,
				Count:           &intstr.IntOrString{Type: intstr.String, StrVal: "3"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Containers: []string{"ctn1"},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop: 100,
				},
			}

			// not all pods are available at first
			expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusPartiallyInjected
		})

		It("should scale up and down with the right number of targets count", func(ctx SpecContext) {
			ExpectDisruptionStatusAndCounts := func(ctx SpecContext, status chaostypes.DisruptionInjectionStatus, a, b, c, d int) {
				GinkgoHelper()

				Concurrently{
					func(ctx SpecContext) {
						ExpectDisruptionStatusCounts(ctx, disruption, a, b, c, d)
					},
					func(ctx SpecContext) {
						ExpectDisruptionStatus(ctx, disruption, status)
					},
				}.DoAndWait(ctx)
			}

			By("Missing targets are reported")
			ExpectDisruptionStatusAndCounts(ctx, chaostypes.DisruptionInjectionStatusPartiallyInjected, 3, 0, 2, 2)

			By("creating extra target one")
			extraOneCreated := CreateRunningPod(ctx, *targetPod.DeepCopy())
			By("creating extra target two")
			extraTwoCreated := CreateRunningPod(ctx, *targetPod.DeepCopy())

			By("waiting extra targets to be created and running")
			extraOne, extraTwo := <-extraOneCreated, <-extraTwoCreated

			By("Additional targets are reported")
			ExpectDisruptionStatusAndCounts(ctx, chaostypes.DisruptionInjectionStatusInjected, 3, 1, 3, 3)

			Concurrently{
				func(ctx SpecContext) {
					By("deleting extra target one")
					DeleteRunningPod(ctx, extraOne)
				},
				func(ctx SpecContext) {
					By("deleting extra target two")
					DeleteRunningPod(ctx, extraTwo)
				},
			}.DoAndWait(ctx)

			By("Back to missing target properly reported")
			ExpectDisruptionStatusAndCounts(ctx, chaostypes.DisruptionInjectionStatusPartiallyInjected, 3, 0, 2, 2)
		})
	})

	Context("manually delete a chaos pod", func() {
		It("should properly handle the chaos pod finalizer", func(ctx SpecContext) {
			By("Ensuring that the chaos pods have been created")
			ExpectChaosPods(ctx, disruption, 4)

			By("Listing chaos pods to pick one to delete")
			chaosPod := PickFirstChaodPod(ctx, disruption)
			chaosPodKey := types.NamespacedName{Namespace: chaosPod.Namespace, Name: chaosPod.Name}

			By("Deleting one of the chaos pod")
			DeleteRunningPod(ctx, chaosPod)

			By("Waiting for the chaos pod finalizer to be removed")
			ExpectChaosPodToDisappear(ctx, chaosPodKey, disruption)
		})
	})

	Context("don't reinject a static node disruption", func() {
		BeforeEach(func() {
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun:   true,
				Duration: shortDisruptionDuration,
				Count:    &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				StaticTargeting: true,
				Level:           chaostypes.DisruptionLevelPod,
				NodeFailure:     &chaosv1beta1.NodeFailureSpec{Shutdown: false},
			}
		})

		When("chaos pods doesn't exist for injected targets", func() {
			It("should not recreate those chaos pods", func(ctx SpecContext) {
				By("Initially targeting both pods")
				ExpectChaosPods(ctx, disruption, 2)

				By("Listing chaos pods to pick one to delete")
				chaosPod := PickFirstChaodPod(ctx, disruption)

				By("Deleting one of the chaos pod")
				DeleteRunningPod(ctx, chaosPod)

				By("Waiting to only have one chaos pod")
				ExpectChaosPods(ctx, disruption, 1)

				By("Waiting to see the second chaos pod is not re-created")
				Consistently(expectChaosPod).WithContext(ctx).WithArguments(disruption, 2).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).ShouldNot(Succeed())
			})
		})
	})

	Context("Cloud disruption is a host disruption disguised", func() {
		BeforeEach(func() {
			skipSecondPod = false
			disruption.Spec = chaosv1beta1.DisruptionSpec{
				DryRun: false,
				Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 2},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{
					DisableAll: true,
				},
				Level: chaostypes.DisruptionLevelPod,
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Cloud: &chaosv1beta1.NetworkDisruptionCloudSpec{
						AWSServiceList: &[]chaosv1beta1.NetworkDisruptionCloudServiceSpec{
							{
								ServiceName: "S3",
								Flow:        "egress",
								Protocol:    "tcp",
							},
						},
					},
					Drop:    0,
					Corrupt: 0,
					Delay:   1,
				},
			}
		})

		It("should create a cloud disruption but apply a host disruption with the list of cloud managed service ip ranges", func(ctx SpecContext) {
			By("Ensuring that the chaos pod have been created")
			ExpectChaosPods(ctx, disruption, 2)

			By("Ensuring that the chaos pods have the list of AWS hosts")
			Eventually(func(ctx SpecContext) error {
				// get chaos pods
				l, err := listChaosPods(ctx, disruption)
				if err != nil {
					return err
				}

				hosts := make([]int, len(l.Items))

				// sum up injectors
				for i, p := range l.Items {
					hosts[i] = 0
					args := p.Spec.Containers[0].Args
					for _, arg := range args {
						if arg == "--hosts" {
							hosts[i]++
						}
					}
				}

				for i, hostsForItem := range hosts {
					if hostsForItem == 0 {
						return fmt.Errorf("should have multiple hosts parameters.")
					}

					// verify that all chaos pods have the same list of hosts
					if i > 0 {
						if hosts[i] != hosts[i-1] {
							return fmt.Errorf("should have the same list of hosts for all chaos pods")
						}
					}
				}

				return nil
			}).WithContext(ctx).ProbeEvery(disruptionPotentialChangesEvery).Within(calcDisruptionGoneTimeout(disruption)).Should(Succeed())
		})
	})

	Context("Injection Statuses", func() {
		BeforeEach(func() {
			disruption.Spec.Count.IntVal = 2
		})

		Specify("paused statuses", func(ctx SpecContext) {
			AddReportEntry("removing one pod so we reach status partially injected", targetPod.Name)
			DeleteRunningPod(ctx, targetPod)
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPartiallyInjected)

			AddReportEntry("removing all pods so we reach status paused partially injected")
			DeleteRunningPod(ctx, anotherTargetPod)
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPausedPartiallyInjected)

			AddReportEntry("creating the two pods again we reach status injected")
			targetPodCreated := CreateRunningPod(ctx, targetPod)
			anotherTargetPodCreated := CreateRunningPod(ctx, anotherTargetPod)

			AddReportEntry("waiting for the two pods to be running")
			targetPod, anotherTargetPod = <-targetPodCreated, <-anotherTargetPodCreated

			AddReportEntry("waiting for disruption status to be injected again")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)

			AddReportEntry("deleting all targets quickly we reach status paused injected or paused partially injected")
			Concurrently{
				func(ctx SpecContext) {
					DeleteRunningPod(ctx, targetPod)
				},
				func(ctx SpecContext) {
					DeleteRunningPod(ctx, anotherTargetPod)
				},
			}.DoAndWait(ctx)
			// it's not possible to guarantee a reconcile loop will notice the two pods at once or one pod then another
			// and it's also not critical for us, hence we allow both statuses
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPausedInjected, chaostypes.DisruptionInjectionStatusPausedPartiallyInjected)

			AddReportEntry("early disruption deletion as all statuses have been seen")
			DeleteDisruption(ctx, disruption)
		})

		Context("disruption expired statuses", func() {
			BeforeEach(func() {
				// let's have a quick disruption by default when we test expiration
				disruption.Spec.Duration = shortDisruptionDuration
				disruption.Spec.CPUPressure = nil
				disruption.Spec.DNS = nil
				disruption.Spec.GRPC = nil
			})

			Context("PreviouslyInjected", func() {
				BeforeEach(func() {
					// single pod so it's possible to inject quickly
					disruption.Spec.Count.IntVal = 1

					// no need to have two pods here, hence skipped
					skipSecondPod = true
				})

				Specify("is the default ending status", func(ctx SpecContext) {
					AddReportEntry("we expect previously injected status at the end")
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
					ExpectDisruptionStatusUntilExpired(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
				})

				Specify("is reached after PausedInjected and stays until disruption expires", func(ctx SpecContext) {
					AddReportEntry("deleting the target pod until paused injected")
					DeleteRunningPod(ctx, targetPod)
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPausedInjected)

					AddReportEntry("no new target leads to previoulsy injected at the end")
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
					ExpectDisruptionStatusUntilExpired(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
				})
			})

			Context("PreviouslyNotInjected", func() {
				BeforeEach(func() {
					// we do not create the second pod AND we ask for random labels that does not exists so disruption stays NotInjected
					// it also confirm we update the status at least once (to NotInjected instead of keeping an empty value)
					skipSecondPod = true

					disruption.Spec.Selector = labels.Set{
						"anything-that-does-not-exists": "this-also-should-not-exists",
					}

					expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusNotInjected
				})

				Specify("stays until disruption expires", func(ctx SpecContext) {
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyNotInjected)
					ExpectDisruptionStatusUntilExpired(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyNotInjected)
				})
			})

			Context("PreviouslyPartiallyInjected", func() {
				BeforeEach(func() {
					// we ask the disruption to target 2 pods and never create the second one
					// hence status should stay until the disruption expire to PartiallyInjected
					disruption.Spec.Count.IntVal = 2
					skipSecondPod = true
					expectedDisruptionStatus = chaostypes.DisruptionInjectionStatusPartiallyInjected
				})

				Specify("stays until disruption expires", func(ctx SpecContext) {
					ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyPartiallyInjected)
					ExpectDisruptionStatusUntilExpired(ctx, disruption, chaostypes.DisruptionInjectionStatusPreviouslyPartiallyInjected)
				})
			})
		})
	})
})

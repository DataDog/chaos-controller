// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/ddmark"
	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"
	tracernoop "github.com/DataDog/chaos-controller/o11y/tracer/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testDisruptionName = "test-disruption"
)

var oldDisruption, newDisruption *Disruption

var _ = Describe("Disruption", func() {
	var ddmarkMock *ddmark.ClientMock

	BeforeEach(func() {
		ddmarkMock = ddmark.NewClientMock(GinkgoT())
		ddmarkClient = ddmarkMock
	})

	Context("ValidateUpdate", func() {
		BeforeEach(func() {
			oldDisruption = makeValidNetworkDisruption()
			newDisruption = oldDisruption.DeepCopy()
		})

		Describe("finalizer removal expectations", func() {
			BeforeEach(func() {
				k8sClient = makek8sClientWithDisruptionPod()
				controllerutil.AddFinalizer(oldDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				k8sClient = nil
			})

			When("disruption is deleting without associated disruption pods", func() {
				It("should succeed to remove finalizer (controller needs to be able to remove finalizer in such case)", func() {
					// override client to return NO pod
					k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

					oldDisruption.DeletionTimestamp = &metav1.Time{
						Time: time.Now(),
					}

					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
				})
			})

			When("disruption is running and has pods", func() {
				It("should fail to remove finalizer", func() {
					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
				})
			})

			When("disruption is deleting WITH associated disruption pods", func() {
				It("should fail to remove finalizer", func() {
					oldDisruption.DeletionTimestamp = &metav1.Time{
						Time: time.Now(),
					}

					err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("unable to remove disruption finalizer"))
				})
			})

			When("disruption did not had finalizer", func() {
				It("should be OK to stays without finalizer", func() {
					oldDisruption.Finalizers = nil

					err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})
		})

		Describe("hash changes expectations", func() {
			When("nothing is updated", func() {
				It("should succeed", func() {
					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
				})
			})

			When("userInfo annotation is updated", func() {
				Context("with an old disruption without user info", func() {
					It("should allow the update", func() {
						// Arrange
						delete(oldDisruption.Annotations, annotationUserInfoKey)

						// Action
						err := newDisruption.ValidateUpdate(oldDisruption)

						// Assert
						By("not return an error")
						Expect(err).ShouldNot(HaveOccurred())
					})
				})

				Context("with an old disruption with an empty user info", func() {
					When("the user info of the new disruption is updated ", func() {
						It("should not allow the update", func() {
							// Arrange
							Expect(newDisruption.SetUserInfo(authv1.UserInfo{})).Should(Succeed())

							// Action
							err := newDisruption.ValidateUpdate(oldDisruption)

							// Assert
							By("return an error")
							Expect(err).Should(HaveOccurred())
							Expect(err).To(MatchError("the user info annotation is immutable"))
						})
					})
					When("the user info of the new disruption is empty too", func() {
						It("should allow the update", func() {
							// Arrange
							Expect(oldDisruption.SetUserInfo(authv1.UserInfo{})).Should(Succeed())
							Expect(newDisruption.SetUserInfo(authv1.UserInfo{Username: "lorem"})).Should(Succeed())

							// Action
							Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
						})
					})
				})

				Context("with an old disruption with a valid user info", func() {
					When("the user info of the new disruption is updated", func() {
						It("should not allow the update", func() {
							// Arrange
							Expect(oldDisruption.SetUserInfo(authv1.UserInfo{Username: "lorem"})).Should(Succeed())
							Expect(newDisruption.SetUserInfo(authv1.UserInfo{Username: "ipsum"})).Should(Succeed())

							// Action
							err := newDisruption.ValidateUpdate(oldDisruption)

							// Assert
							By("return an error")
							Expect(err).Should(HaveOccurred())
							Expect(err).To(MatchError("the user info annotation is immutable"))
						})
					})
				})
			})

			When("count is updated", func() {
				BeforeEach(func() {
					newDisruption.Spec.Count = &intstr.IntOrString{IntVal: 2}
				})

				Context("DynamicTargeting (StaticTargeting=false)", func() {
					It("should succeed", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
					})
				})

				Context("StaticTargeting", func() {
					It("should fail", func() {
						oldDisruption.Spec.StaticTargeting = true
						newDisruption.Spec.StaticTargeting = true

						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})
			})

			When("StaticTargeting is updated", func() {
				When("static to dynamic", func() {
					BeforeEach(func() {
						oldDisruption.Spec.StaticTargeting = true
						newDisruption.Spec.StaticTargeting = false
					})

					It("should fail", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})

				When("dynamic to static", func() {
					BeforeEach(func() {
						oldDisruption.Spec.StaticTargeting = false
						newDisruption.Spec.StaticTargeting = true
					})

					It("should fail", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})
			})
		})
	})

	Context("ValidateCreate", func() {
		Describe("general errors expectations", func() {
			BeforeEach(func() {
				k8sClient = makek8sClientWithDisruptionPod()
				tracerSink = tracernoop.New(logger)
				deleteOnly = false
			})

			JustBeforeEach(func() {
				newDisruption = makeValidNetworkDisruption()
				controllerutil.AddFinalizer(newDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				k8sClient = nil
				newDisruption = nil
			})

			When("disruption has delete-only mode enable", func() {
				It("should return an error which deny the creation of a new disruption", func() {
					// Arrange
					deleteOnly = true

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("the controller is currently in delete-only mode, you can't create new disruptions for now"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("disruption has invalid name", func() {
				It("should return an error for an invalid d", func() {
					// Arrange
					newDisruption.Name = "invalid-name!"

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("invalid disruption name: found '!', expected: ',' or 'end of string'"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("disruption using the onInit feature without the handler being enabled", func() {
				It("should return an error", func() {
					// Arrange
					newDisruption.Spec.OnInit = true

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("the chaos handler is disabled but the disruption onInit field is set to true, please enable the handler by specifying the --handler-enabled flag to the controller if you want to use the onInit feature"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("disruption spec is invalid", func() {
				It("should return an error", func() {
					// Arrange
					invalidDisruption := newDisruption.DeepCopy()
					invalidDisruption.Spec.Selector = nil

					// Action
					err := invalidDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("1 error occurred:\n\t* Spec: either selector or advancedSelector field must be set\n\n"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("disruption spec duration is greater than the max default duration", func() {
				var originalMaxDefaultDuration time.Duration

				BeforeEach(func() {
					originalMaxDefaultDuration = maxDuration
				})

				It("should return an error", func() {
					// Arrange
					invalidDisruption := newDisruption.DeepCopy()
					maxDuration = time.Hour * 1
					invalidDisruption.Spec.Duration = "2h"

					// Action
					err := invalidDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("the maximum duration allowed is 1h0m0s, please specify a duration lower or equal than this value"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})

				AfterEach(func() {
					maxDuration = originalMaxDefaultDuration
				})
			})

			When("disruption selectors are invalid", func() {
				It("should return an error", func() {
					invalidDisruption := newDisruption.DeepCopy()
					invalidDisruption.Spec.Selector = map[string]string{"app": "demo-{nginx}"}

					err := invalidDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("1 error occurred:\n\t* Spec: unable to parse requirement: values[0][app]: Invalid value: \"demo-{nginx}\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')\n\n"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("disruption advanced selectors are invalid", func() {
				It("should return an error", func() {
					invalidDisruption := newDisruption.DeepCopy()
					invalidDisruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{{
						Key:      "app",
						Operator: "NotIn",
						Values:   []string{"*nginx"},
					}}

					err := invalidDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("1 error occurred:\n\t* Spec: error parsing given advanced selector to requirements: values[0][app]: Invalid value: \"*nginx\": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')\n\n"))
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 0)).To(BeTrue())
				})
			})

			When("ddmark return an error", func() {
				It("should catch this error and propagated it", func() {
					// Arrange
					ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{
						Errors: []error{
							fmt.Errorf("something bad happened"),
						},
					})

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("something bad happened"))
				})
			})

			When("triggers.inject.notBefore is before triggers.createPods.notBefore", func() {
				It("should return an error", func() {
					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = DisruptionTriggers{
						Inject: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 5)),
						},
						CreatePods: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 15)),
						},
					}

					Expect(newDisruption.ValidateCreate().Error()).Should(ContainSubstring("inject.notBefore must come after createPods.notBefore if both are specified"))
				})
			})
		})

		Describe("expectations with a disk failure disruption", func() {
			BeforeEach(func() {
				ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})
				k8sClient = makek8sClientWithDisruptionPod()
				recorder = record.NewFakeRecorder(1)
				metricsSink = metricsnoop.New(logger)
				tracerSink = tracernoop.New(logger)
				deleteOnly = false
				enableSafemode = true
			})

			JustBeforeEach(func() {
				newDisruption = makeValidDiskFailureDisruption()
				controllerutil.AddFinalizer(newDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				k8sClient = nil
				newDisruption = nil
			})

			When("the disruption target a 'node'", func() {
				JustBeforeEach(func() {
					newDisruption.Spec.Level = chaostypes.DisruptionLevelNode
				})
				Context("with the '/' path", func() {
					It("should deny the usage of '/' path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Paths = []string{"/test", "/"}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\"."))
					})
				})
				Context("with the '  /  ' path", func() {
					It("should deny the usage of '   /   ' path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Paths = []string{"/test", "   /   "}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\"."))
					})
				})
				Context("safe-mode disabled", func() {
					It("should allow the '/' path", func() {
						// Arrange
						newDisruption.Spec.Unsafemode = &UnsafemodeSpec{
							AllowRootDiskFailure: true,
						}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})

			When("the disruption target a 'pod'", func() {
				JustBeforeEach(func() {
					newDisruption.Spec.Level = chaostypes.DisruptionLevelPod
				})
				Context("with the '/' path", func() {
					It("should allow the usage of this path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Paths = []string{"/test", "/"}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
				Context("with the safe-mode disabled", func() {
					It("should allow the '/' path", func() {
						// Arrange
						newDisruption.Spec.Unsafemode = &UnsafemodeSpec{
							AllowRootDiskFailure: true,
						}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})
	})
})

// makeValidNetworkDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidNetworkDisruption() *Disruption {
	disruption := Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDisruptionName,
			Namespace: chaosNamespace,
		},
		Spec: DisruptionSpec{
			Count: &intstr.IntOrString{
				IntVal: 1,
			},
			Network: &NetworkDisruptionSpec{
				Drop: 100,
			},
			Selector: labels.Set{
				"name":      "random",
				"namespace": "random",
			},
		},
	}

	disruption.Annotations = map[string]string{}

	if err := disruption.SetUserInfo(authv1.UserInfo{
		Username: "lorem",
		UID:      "ipsum",
		Groups:   []string{"some"},
		Extra: map[string]authv1.ExtraValue{
			"dolores": []string{"sit"},
		},
	}); err != nil {
		Expect(err).ShouldNot(HaveOccurred())
	}

	return &disruption
}

// makeValidDiskFailureDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidDiskFailureDisruption() *Disruption {
	return &Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDisruptionName,
			Namespace: chaosNamespace,
		},
		Spec: DisruptionSpec{
			Count: &intstr.IntOrString{
				IntVal: 1,
			},
			Selector: labels.Set{
				"name":      "random",
				"namespace": "random",
			},
			DiskFailure: &DiskFailureSpec{
				Paths:       []string{"/"},
				Probability: "100%",
			},
		},
	}
}

// makek8sClientWithDisruptionPod is a help that creates a k8sClient returning at least one valid pod associated with the Disruption created with makeValidNetworkDisruption
func makek8sClientWithDisruptionPod() client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "like-disruption-pod",
				Namespace: chaosNamespace,
				Labels: map[string]string{
					chaostypes.DisruptionNameLabel:      testDisruptionName,
					chaostypes.DisruptionNamespaceLabel: chaosNamespace,
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "k8s.gcr.io/pause:3.4.1",
						Name:  "ctn1",
					},
				},
			},
		}).
		Build()
}

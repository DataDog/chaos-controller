// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/DataDog/chaos-controller/mocks"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testDisruptionName = "test-disruption"
)

var oldDisruption, newDisruption *Disruption

var _ = Describe("Disruption Webhook", func() {
	var (
		ddmarkMock   *ddmark.ClientMock
		recorderMock *mocks.EventRecorderMock
	)

	BeforeEach(func() {
		ddmarkMock = ddmark.NewClientMock(GinkgoT())
		ddmarkClient = ddmarkMock
		allowNodeLevel = true
		allowNodeFailure = true
		recorderMock = mocks.NewEventRecorderMock(GinkgoT())
		recorder = recorderMock
	})

	Describe("ValidateUpdate", func() {
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

					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})

			When("disruption is running and has pods", func() {
				It("should fail to remove finalizer", func() {
					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(HaveOccurred())
				})
			})

			When("disruption is deleting WITH associated disruption pods", func() {
				It("should fail to remove finalizer", func() {
					oldDisruption.DeletionTimestamp = &metav1.Time{
						Time: time.Now(),
					}

					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("unable to remove disruption finalizer"))
				})
			})

			When("disruption did not had finalizer", func() {
				It("should be OK to stays without finalizer", func() {
					oldDisruption.Finalizers = nil

					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})
		})

		Describe("ValidateCreate-only invariants shouldn't affect Update", func() {
			When("triggers.*.notBefore is in the past", func() {
				It("triggers.inject should not return an error", func() {
					triggers := DisruptionTriggers{
						Inject: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 5 * -1)),
						},
					}

					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &triggers

					oldDisruption.Spec.Duration = "30m"
					oldDisruption.Spec.Triggers = &triggers

					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})

				It("triggers.createPods should not return an error", func() {
					triggers := DisruptionTriggers{
						CreatePods: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Hour * -1)),
						},
					}
					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &triggers

					oldDisruption.Spec.Duration = "30m"
					oldDisruption.Spec.Triggers = &triggers

					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})
		})

		Describe("hash changes expectations", func() {
			When("nothing is updated", func() {
				It("should succeed", func() {
					_, err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})

			When("userInfo annotation is updated", func() {
				Context("with an old disruption without user info", func() {
					It("should allow the update", func() {
						// Arrange
						delete(oldDisruption.Annotations, annotationUserInfoKey)

						// Action
						_, err := newDisruption.ValidateUpdate(oldDisruption)

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
							_, err := newDisruption.ValidateUpdate(oldDisruption)

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
							_, err := newDisruption.ValidateUpdate(oldDisruption)
							Expect(err).Should(Succeed())
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
							_, err := newDisruption.ValidateUpdate(oldDisruption)

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
						_, err := newDisruption.ValidateUpdate(oldDisruption)
						Expect(err).Should(Succeed())
					})
				})

				Context("StaticTargeting", func() {
					It("should fail", func() {
						oldDisruption.Spec.StaticTargeting = true
						newDisruption.Spec.StaticTargeting = true

						_, err := newDisruption.ValidateUpdate(oldDisruption)
						Expect(err).Should(HaveOccurred())
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
						_, err := newDisruption.ValidateUpdate(oldDisruption)
						Expect(err).Should(HaveOccurred())
					})
				})

				When("dynamic to static", func() {
					BeforeEach(func() {
						oldDisruption.Spec.StaticTargeting = false
						newDisruption.Spec.StaticTargeting = true
					})

					It("should fail", func() {
						_, err := newDisruption.ValidateUpdate(oldDisruption)
						Expect(err).Should(HaveOccurred())
					})
				})
			})
		})
	})

	Describe("ValidateCreate", func() {
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
				permittedUserGroups = map[string]struct{}{}
			})

			When("disruption has delete-only mode enable", func() {
				It("should return an error which deny the creation of a new disruption", func() {
					// Arrange
					deleteOnly = true

					// Action
					_, err := newDisruption.ValidateCreate()

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
					_, err := newDisruption.ValidateCreate()

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
					_, err := newDisruption.ValidateCreate()

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
					_, err := invalidDisruption.ValidateCreate()

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
					_, err := invalidDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("you have specified a duration of 2h0m0s, but the maximum duration allowed is 1h0m0s, please specify a duration lower or equal than this value"))
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

					_, err := invalidDisruption.ValidateCreate()

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

					_, err := invalidDisruption.ValidateCreate()

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
					_, err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("something bad happened"))
				})
			})

			When("triggers.*.notBefore is in the past", func() {
				It("triggers.inject should return an error", func() {
					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &DisruptionTriggers{
						Inject: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 5 * -1)),
						},
					}

					_, err := newDisruption.ValidateCreate()
					Expect(err).Should(MatchError(ContainSubstring("only values in the future are accepted")))
				})

				It("triggers.createPods should return an error", func() {
					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &DisruptionTriggers{
						CreatePods: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Hour * -1)),
						},
					}

					_, err := newDisruption.ValidateCreate()
					Expect(err).Should(MatchError(ContainSubstring("only values in the future are accepted")))
				})
			})

			When("triggers.*.notBefore is in the future", func() {
				It("should not return an error", func() {
					// Arrange
					ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})

					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &DisruptionTriggers{
						Inject: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 5)),
						},
					}

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("triggers.inject.notBefore is before triggers.createPods.notBefore", func() {
				It("should return an error", func() {
					newDisruption.Spec.Duration = "30m"
					newDisruption.Spec.Triggers = &DisruptionTriggers{
						Inject: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 5)),
						},
						CreatePods: DisruptionTrigger{
							NotBefore: metav1.NewTime(time.Now().Add(time.Minute * 15)),
						},
					}

					_, err := newDisruption.ValidateCreate()
					Expect(err).Should(MatchError(ContainSubstring("inject.notBefore must come after createPods.notBefore if both are specified")))
				})
			})

			When("user group membership is invalid", func() {
				It("should return an error if they lack membership", func() {
					permittedUserGroups = map[string]struct{}{}
					permittedUserGroups["system:nobody"] = struct{}{}
					permittedUserGroupWarningString = "system:nobody"

					_, err := newDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("lacking sufficient authorization to create Disruption. your user groups are some, but you must be in one of the following groups: system:nobody"))
				})

				It("should not return an error if they are within a permitted group", func() {
					// Arrange
					ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})
					permittedUserGroups = map[string]struct{}{}
					permittedUserGroups["some"] = struct{}{}
					permittedUserGroups["any"] = struct{}{}
					permittedUserGroupWarningString = "some, any"

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
					Expect(ddmarkMock.AssertNumberOfCalls(GinkgoT(), "ValidateStructMultierror", 1)).To(BeTrue())
				})
			})
		})

		Describe("expectations with node disruptions", func() {
			BeforeEach(func() {
				ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})
				k8sClient = makek8sClientWithDisruptionPod()
				metricsSink = metricsnoop.New(logger)
				tracerSink = tracernoop.New(logger)
				deleteOnly = false
				enableSafemode = true
				allowNodeFailure = true
				allowNodeLevel = true
			})

			JustBeforeEach(func() {
				newDisruption = makeValidNetworkDisruption()
			})

			AfterEach(func() {
				k8sClient = nil
				newDisruption = nil
			})

			Context("allowNodeFailure is false", func() {
				JustBeforeEach(func() {
					newDisruption.Spec.NodeFailure = &NodeFailureSpec{}
				})

				It("should reject the disruption at node level", func() {
					allowNodeFailure = false
					newDisruption.Spec.Level = chaostypes.DisruptionLevelNode

					_, err := newDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError(ContainSubstring("at least one of the initial safety nets caught an issue")))
					Expect(err.Error()).Should(ContainSubstring("node failure disruptions are not allowed in this cluster"))

				})

				It("should reject the disruption at pod level", func() {
					allowNodeFailure = false

					_, err := newDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
					Expect(err.Error()).Should(ContainSubstring("node failure disruptions are not allowed in this cluster"))

				})
			})

			Context("allowNodeLevel is false", func() {
				It("should reject the disruption at node level", func() {
					// Arrange
					allowNodeLevel = false
					newDisruption.Spec.Level = chaostypes.DisruptionLevelNode

					// Action
					_, err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
					Expect(err.Error()).Should(ContainSubstring("node level disruptions are not allowed in this cluster"))
				})

				It("should allow the disruption at pod level", func() {
					// Arrange
					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Context("allowNodeFailure and allowNodeLevel are true", func() {
				It("should allow a node level node failure disruption", func() {
					// Arrange
					allowNodeFailure = true
					allowNodeLevel = true
					newDisruption.Spec.Level = chaostypes.DisruptionLevelNode
					newDisruption.Spec.NodeFailure = &NodeFailureSpec{}

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Describe("safemode expectations with a disk failure disruption", func() {
			BeforeEach(func() {
				ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})
				k8sClient = makek8sClientWithDisruptionPod()
				metricsSink = metricsnoop.New(logger)
				tracerSink = tracernoop.New(logger)
				deleteOnly = false
				enableSafemode = true
			})

			JustBeforeEach(func() {
				newDisruption = makeValidDiskFailureDisruption()
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
						_, err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\""))
					})
				})
				Context("with the '/test' path", func() {
					It("should allow the usage", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Paths = []string{"/test"}

						disruptionJSON, err := json.Marshal(newDisruption)
						Expect(err).ShouldNot(HaveOccurred())

						expectedAnnotations := map[string]string{
							EventDisruptionAnnotation: string(disruptionJSON),
						}

						recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

						// Action
						_, err = newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
				Context("with the '  /  ' path", func() {
					It("should deny the usage of '   /   ' path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Paths = []string{"/test", "   /   "}

						// Action
						_, err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\""))
					})
				})
				Context("safe-mode disabled", func() {
					It("should allow the '/' path", func() {
						// Arrange
						newDisruption.Spec.Unsafemode = &UnsafemodeSpec{
							AllowRootDiskFailure: true,
						}

						disruptionJSON, err := json.Marshal(newDisruption)
						Expect(err).ShouldNot(HaveOccurred())

						expectedAnnotations := map[string]string{
							EventDisruptionAnnotation: string(disruptionJSON),
						}

						recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

						// Action
						_, err = newDisruption.ValidateCreate()

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

						disruptionJSON, err := json.Marshal(newDisruption)
						Expect(err).ShouldNot(HaveOccurred())

						expectedAnnotations := map[string]string{
							EventDisruptionAnnotation: string(disruptionJSON),
						}

						recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

						// Action
						_, err = newDisruption.ValidateCreate()

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

						disruptionJSON, err := json.Marshal(newDisruption)
						Expect(err).ShouldNot(HaveOccurred())

						expectedAnnotations := map[string]string{
							EventDisruptionAnnotation: string(disruptionJSON),
						}

						recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

						// Action
						_, err = newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})

		Describe("safemode expectations with a network failure disruption", func() {
			BeforeEach(func() {
				ddmarkMock.EXPECT().ValidateStructMultierror(mock.Anything, mock.Anything).Return(&multierror.Error{})
				k8sClient = makek8sClientWithDisruptionPod()
				metricsSink = metricsnoop.New(logger)
				tracerSink = tracernoop.New(logger)
				deleteOnly = false
				enableSafemode = true
			})

			JustBeforeEach(func() {
				newDisruption = makeValidNetworkDisruption()
			})

			AfterEach(func() {
				k8sClient = nil
				newDisruption = nil
			})

			When("only services are defined", func() {
				It("should pass validation", func() {
					// Arrange
					newDisruption.Spec.Network.Hosts = nil
					newDisruption.Spec.Network.Services = []NetworkDisruptionServiceSpec{{Name: "foo", Namespace: chaosNamespace}}

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("various host filters are defined", func() {
				DescribeTable("should pass validation", func(hosts []NetworkDisruptionHostSpec) {
					// Arrange
					newDisruption.Spec.Network.Hosts = hosts

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
					Entry("with just a port filter", []NetworkDisruptionHostSpec{{Port: 80}}),
					Entry("with just a protocol filter", []NetworkDisruptionHostSpec{{Protocol: "tcp"}}),
					Entry("with just a hostname filter", []NetworkDisruptionHostSpec{{Host: "localhost"}}),
					Entry("with all possible host filters together", []NetworkDisruptionHostSpec{{Host: "localhost", Port: 443, Protocol: "udp"}}),
				)
			})

			When("no filters are defined", func() {
				It("should be rejected", func() {
					newDisruption.Spec.Network.Hosts = nil

					_, err := newDisruption.ValidateCreate()

					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
					Expect(err.Error()).Should(ContainSubstring("the specified disruption either contains no Host or Service filters. This will result in all network traffic being affected"))
				})

				It("should be allowed with DisableNeitherHostNorPort", func() {
					// Arrange
					newDisruption.Spec.Network.Hosts = nil
					newDisruption.Spec.Unsafemode = &UnsafemodeSpec{DisableNeitherHostNorPort: true}

					disruptionJSON, err := json.Marshal(newDisruption)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotations := map[string]string{
						EventDisruptionAnnotation: string(disruptionJSON),
					}

					recorderMock.EXPECT().AnnotatedEventf(newDisruption, expectedAnnotations, Events[EventDisruptionCreated].Type, string(EventDisruptionCreated), Events[EventDisruptionCreated].OnDisruptionTemplateMessage)

					// Action
					_, err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

		})
	})
})

// makeValidNetworkDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidNetworkDisruption() *Disruption {
	disruption := Disruption{
		TypeMeta: metav1.TypeMeta{
			Kind: DisruptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDisruptionName,
			Namespace: chaosNamespace,
		},
		Spec: DisruptionSpec{
			Count: &intstr.IntOrString{
				IntVal: 1,
			},
			Network: &NetworkDisruptionSpec{
				Drop:  100,
				Hosts: []NetworkDisruptionHostSpec{{Port: 80}},
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
		},
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: chaosNamespace,
				},
				Spec: v1.ServiceSpec{
					ClusterIP: "localhost",
					Type:      v1.ServiceTypeClusterIP,
				},
			},
		).
		Build()
}

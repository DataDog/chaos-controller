// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"encoding/json"
	"time"

	"github.com/DataDog/chaos-controller/mocks"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	authV1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DisruptionCron Webhook", func() {
	var (
		defaultUserGroups    map[string]struct{}
		defaultUserGroupsStr string
	)

	BeforeEach(func() {
		disruptionCronWebhookLogger = zaptest.NewLogger(GinkgoT()).Sugar()
		defaultUserGroups = map[string]struct{}{
			"group1": {},
			"group2": {},
		}
		defaultUserGroupsStr = "group1, group2"
	})

	AfterEach(func() {
		disruptionCronWebhookLogger = nil
		disruptionCronWebhookRecorder = nil
		disruptionCronWebhookDeleteOnly = false
		disruptionCronPermittedUserGroups = nil
		defaultUserGroups = nil
		defaultUserGroupsStr = ""
		minimumCronFrequency = time.Second
	})

	Describe("ValidateCreate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronCreated event to the broadcast", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("sending the EventDisruptionCronCreated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("permitted user groups is present", func() {

				BeforeEach(func() {
					disruptionCronPermittedUserGroups = defaultUserGroups
					disruptionCronPermittedUserGroupString = defaultUserGroupsStr
				})

				When("the userinfo is in the permitted user groups", func() {
					It("should send an EventDisruptionCronCreated event to the broadcast", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						Expect(disruptionCron.SetUserInfo(authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						})).To(Succeed())

						disruptionCronJSON, err := json.Marshal(disruptionCron)
						Expect(err).ShouldNot(HaveOccurred())

						expectedAnnotation := map[string]string{
							EventDisruptionCronAnnotation: string(disruptionCronJSON),
						}

						By("sending the EventDisruptionCronCreated event to the broadcast")
						mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
						mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
						disruptionCronWebhookRecorder = mockEventRecorder

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).ShouldNot(HaveOccurred())
						Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
						Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					})
				})
			})

			When("spec.delayedStartTolerance is set", func() {
				It("should send an EventDisruptionCronCreated event to the broadcast", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.DelayedStartTolerance = DisruptionDuration("1s")

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("sending the EventDisruptionCronCreated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("the disruption template duration is greater than 0", func() {
				It("should validate the disruption cron successfully", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.DisruptionTemplate.Duration = "1s"

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("sending the EventDisruptionCronCreated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().
						AnnotatedEventf(
							disruptionCron,
							expectedAnnotation,
							Events[EventDisruptionCronCreated].Type,
							string(EventDisruptionCronCreated),
							Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage,
						)

					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Describe("error cases", func() {
			When("the controller is in delete-only mode", func() {
				It("returns an error", func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true

					By("not emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.AssertNotCalled(GinkgoT(), "AnnotatedEventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := (&DisruptionCron{}).ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("the controller is currently in delete-only mode, you can't create new disruption cron for now"))
				})
			})

			DescribeTable("should return an error when disruption cron name is invalid",
				func(name, expectedError string) {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Name = name

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("when name is an empty string", "", "disruption cron name must be specified"),
				Entry("when name is too long", "a-very-long-disruptioncron-name-of-48-characters", "disruption cron name exceeds maximum length: must be no more than 47 characters"),
				Entry("when name is not a valid Kubernetes label", "invalid-name!", "disruption cron name must follow Kubernetes label format"),
			)

			When("disruption cron schedule is invalid", func() {
				It("should return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.Schedule = "****"

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("spec.Schedule must follow the standard cron syntax")))
				})
			})

			When("disruption cron schedule is too brief", func() {
				It("should return an error", func() {
					minimumCronFrequency = time.Hour * 24 * 365

					disruptionCron := makeValidDisruptionCron()
					warnings, err := disruptionCron.ValidateCreate()

					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("between disruptions, but the minimum tolerated frequency is 8760h")))
				})
			})

			When("disruption cron spec.delayedStartTolerance is invalid", func() {
				It("should return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.DelayedStartTolerance = DisruptionDuration("-1s")

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("spec.delayedStartTolerance must be a positive duration")))
				})
			})

			When("disruptionTemplate is invalid", func() {
				When("the count is invalid", func() {
					// Other forms of invalid disruptions are covered by the disruption_webook_test.go
					// we just want to confirm we do validate the disruptionTemplate as part of the disruption cron webhook
					It("should return an error", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						disruptionCron.Spec.DisruptionTemplate.Count = &intstr.IntOrString{
							StrVal: "2hr",
						}

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("spec.disruptionTemplate validation failed")))
					})
				})
			})

			When("permitted user groups is present", func() {

				BeforeEach(func() {
					disruptionCronPermittedUserGroups = defaultUserGroups
					disruptionCronPermittedUserGroupString = defaultUserGroupsStr
				})

				When("the userinfo is not present", func() {
					It("should not allow the create", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()

						By("not emit an event to the broadcast")
						mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
						mockEventRecorder.AssertNotCalled(GinkgoT(), "AnnotatedEventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
						disruptionCronWebhookRecorder = mockEventRecorder

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("user info not found in annotations")))
					})
				})

				When("the userinfo is not in the permitted user groups", func() {
					It("should not allow the create", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						Expect(disruptionCron.SetUserInfo(authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group3"},
						})).To(Succeed())

						By("not emit an event to the broadcast")
						mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
						mockEventRecorder.AssertNotCalled(GinkgoT(), "AnnotatedEventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
						disruptionCronWebhookRecorder = mockEventRecorder

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("lacking sufficient authorization to create DisruptionCron. your user groups are group3, but you must be in one of the following groups: group1, group2")))
					})
				})
			})
		})
	})

	Describe("ValidateUpdate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronUpdated event to the broadcast", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("sending the EventDisruptionCronUpdated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("the controller is in delete-only mode", func() {
				It("should send an EventDisruptionCronUpdated event to the broadcast", func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true
					disruptionCron := makeValidDisruptionCron()

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("sending the EventDisruptionCronUpdated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("the user info has not changed", func() {
				It("should not return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					userInfo := authV1.UserInfo{
						Username: "username@mail.com",
						Groups:   []string{"group1"},
					}
					Expect(disruptionCron.SetUserInfo(userInfo)).To(Succeed())

					oldDisruptionCron := makeValidDisruptionCron()
					Expect(oldDisruptionCron.SetUserInfo(userInfo)).To(Succeed())

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})

			})

			When("the schedule frequency becomes longer", func() {
				It("should not return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					oldDisruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.Schedule = "0 0 1 * *"

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("the schedule frequency does not change", func() {
				It("should not return an error", func() {
					// Arrange
					minimumCronFrequency = time.Hour * 24 * 365
					disruptionCron := makeValidDisruptionCron()
					oldDisruptionCron := makeValidDisruptionCron()

					disruptionCronJSON, err := json.Marshal(disruptionCron)
					Expect(err).ShouldNot(HaveOccurred())

					expectedAnnotation := map[string]string{
						EventDisruptionCronAnnotation: string(disruptionCronJSON),
					}

					By("emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, expectedAnnotation, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})
		})

		Describe("error cases", func() {
			When("the user info has changed", func() {
				DescribeTable("should return an error", func(userInfo, oldUserInfo authV1.UserInfo) {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					Expect(disruptionCron.SetUserInfo(userInfo)).To(Succeed())

					oldDisruptionCron := makeValidDisruptionCron()
					Expect(oldDisruptionCron.SetUserInfo(oldUserInfo)).To(Succeed())

					By("not emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.AssertNotCalled(GinkgoT(), "AnnotatedEventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("the user info annotation is immutable")))
				},
					Entry("when the username has changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "differentusername@mail.com",
							Groups:   []string{"group1"},
						},
					),
					Entry("when the groups have changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group2"},
						},
					),
					Entry("when the username and groups have changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "newusername@mail.com",
							Groups:   []string{"group2"},
						},
					),
				)
			})

			DescribeTable("should return an error when disruption cron name is invalid",
				func(name, expectedError string) {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Name = name

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring(expectedError)))
				},
				Entry("when name is an empty string", "", "disruption cron name must be specified"),
				Entry("when name is too long", "a-very-long-disruptioncron-name-of-48-characters", "disruption cron name exceeds maximum length: must be no more than 47 characters"),
				Entry("when name is not a valid Kubernetes label", "invalid-name!", "disruption cron name must follow Kubernetes label format"),
			)

			When("disruption cron schedule is invalid", func() {
				It("should return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.Schedule = "****"

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("spec.Schedule must follow the standard cron syntax")))
				})
			})

			When("disruption cron spec.delayedStartTolerance is invalid", func() {
				It("should return an error", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.DelayedStartTolerance = DisruptionDuration("-1s")

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("spec.delayedStartTolerance must be a positive duration")))
				})
			})

			When("the schedule frequency becomes shorter", func() {
				It("should not return an error", func() {
					// Arrange
					minimumCronFrequency = time.Hour * 24 * 365
					disruptionCron := makeValidDisruptionCron()
					oldDisruptionCron := makeValidDisruptionCron()
					disruptionCron.Spec.Schedule = "* * * * *"

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("between disruptions, but the minimum tolerated frequency is 8760h")))
				})
			})

			When("disruptionTemplate is invalid", func() {
				When("the count is invalid", func() {
					// Other forms of invalid disruptions are covered by the disruption_webook_test.go
					// we just want to confirm we do validate the disruptionTemplate as part of the disruption cron webhook
					It("should return an error", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						disruptionCron.Spec.DisruptionTemplate.Count = &intstr.IntOrString{
							StrVal: "2hr",
						}

						// Act
						warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("spec.disruptionTemplate validation failed")))
					})
				})
			})
		})
	})

	Describe("ValidateDelete", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronDeleted event to the broadcast", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					By("sending the EventDisruptionCronDeleted event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, mock.Anything, Events[EventDisruptionCronDeleted].Type, string(EventDisruptionCronDeleted), Events[EventDisruptionCronDeleted].OnDisruptionTemplateMessage).RunAndReturn(
						func(object runtime.Object, annotations map[string]string, _ string, _ string, _ string, _ ...interface{}) {
							inputDisruptionCron := object.(*DisruptionCron)

							inputDisruptionCronAnnotationString := annotations[EventDisruptionCronAnnotation]
							err := json.Unmarshal([]byte(inputDisruptionCronAnnotationString), inputDisruptionCron)
							Expect(err).ShouldNot(HaveOccurred())

							inputDisruptionCron.DeletionTimestamp = nil

							Expect(inputDisruptionCron).To(Equal(disruptionCron))
						},
					)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateDelete()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})

			When("the controller is in delete-only mode", func() {
				It("should send an EventDisruptionCronDeleted event to the broadcast", func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true
					disruptionCron := makeValidDisruptionCron()

					By("sending the EventDisruptionCronDeleted event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, mock.Anything, Events[EventDisruptionCronDeleted].Type, string(EventDisruptionCronDeleted), Events[EventDisruptionCronDeleted].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateDelete()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})
		})
	})

})

func makeValidDisruptionCron() *DisruptionCron {
	validDisruption := makeValidNetworkDisruption()
	return &DisruptionCron{
		TypeMeta: metav1.TypeMeta{
			Kind: DisruptionCronKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "valid-disruption-cron",
		},
		Spec: DisruptionCronSpec{
			Schedule:           "0 0 * * *",
			DisruptionTemplate: validDisruption.Spec,
		},
	}
}

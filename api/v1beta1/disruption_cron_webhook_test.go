// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	authV1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	})

	Describe("ValidateCreate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronCreated event to the broadcast", func() {
					// Arrange
					disruptionCron := &DisruptionCron{
						Spec: DisruptionCronSpec{},
					}

					By("sending the EventDisruptionCronCreated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, mock.Anything, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
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

						By("sending the EventDisruptionCronCreated event to the broadcast")
						mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
						mockEventRecorder.EXPECT().AnnotatedEventf(disruptionCron, mock.Anything, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
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

					By("sending the EventDisruptionCronUpdated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
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

					By("sending the EventDisruptionCronUpdated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
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

					By("emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
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
					mockEventRecorder.AssertNotCalled(GinkgoT(), "Event", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
		})
	})

	Describe("ValidateDelete", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronDeleted event to the broadcast", func() {

					disruptionCron := &DisruptionCron{
						// Arrange
						Spec: DisruptionCronSpec{},
					}

					By("sending the EventDisruptionCronDeleted event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronDeleted].Type, string(EventDisruptionCronDeleted), Events[EventDisruptionCronDeleted].OnDisruptionTemplateMessage)
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
					disruptionCronWebhookDeleteOnly = true
					disruptionCron := &DisruptionCron{
						// Arrange
						Spec: DisruptionCronSpec{},
					}

					By("sending the EventDisruptionCronDeleted event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronDeleted].Type, string(EventDisruptionCronDeleted), Events[EventDisruptionCronDeleted].OnDisruptionTemplateMessage)
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
	return &DisruptionCron{
		TypeMeta: metav1.TypeMeta{
			Kind: DisruptionCronKind,
		},
	}
}

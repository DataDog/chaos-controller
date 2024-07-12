// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	zaplog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("DisruptionCron Webhook", func() {

	BeforeEach(func() {
		var err error
		disruptionCronWebhookLogger, err = zaplog.NewZapLogger()
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		disruptionCronWebhookLogger = nil
		disruptionCronWebhookRecorder = nil
		disruptionCronWebhookDeleteOnly = false
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
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronCreated].Type, string(EventDisruptionCronCreated), Events[EventDisruptionCronCreated].OnDisruptionTemplateMessage)
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

		Describe("error cases", func() {
			When("the controller is in delete-only mode", func() {
				It("returns an error", func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true

					By("not emit an event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.AssertNotCalled(GinkgoT(), "Event", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := (&DisruptionCron{}).ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("the controller is currently in delete-only mode, you can't create new disruption cron for now"))
				})
			})
		})
	})

	Describe("ValidateUpdate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {
				It("should send an EventDisruptionCronUpdated event to the broadcast", func() {
					// Arrange
					disruptionCron := &DisruptionCron{
						Spec: DisruptionCronSpec{},
					}

					By("sending the EventDisruptionCronUpdated event to the broadcast")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.EXPECT().Event(disruptionCron, Events[EventDisruptionCronUpdated].Type, string(EventDisruptionCronUpdated), Events[EventDisruptionCronUpdated].OnDisruptionTemplateMessage)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := disruptionCron.ValidateUpdate(nil)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
					Expect(disruptionCronWebhookRecorder).ShouldNot(BeNil())
				})
			})
		})

		Describe("error cases", func() {
			When("the controller is in delete-only mode", func() {
				It("returns an error", func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true

					By("not emit an event")
					mockEventRecorder := mocks.NewEventRecorderMock(GinkgoT())
					mockEventRecorder.AssertNotCalled(GinkgoT(), "Event", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
					disruptionCronWebhookRecorder = mockEventRecorder

					// Act
					warnings, err := (&DisruptionCron{}).ValidateUpdate(nil)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("the controller is currently in delete-only mode, you can't update disruption cron for now"))
				})
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

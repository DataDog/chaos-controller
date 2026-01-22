// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package datadog_test

import (
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/datadog"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/DataDog/chaos-controller/mocks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Datadog", func() {
	Describe("New", func() {
		Describe("success cases", func() {
			It("should return a new datadog notifier", func() {
				// Arrange
				clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())

				// Act
				notifierDD, err := datadog.New(
					types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					clientStatsdMock,
				)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
				Expect(notifierDD).ShouldNot(BeNil())
			})
		})
	})

	Describe("GetNotifierName", func() {
		It("should return the driver's name", func() {
			// Arrange
			clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
			notifierDD, err := datadog.New(
				types.NotifiersCommonConfig{
					ClusterName: "cluster-name",
				},
				clientStatsdMock,
			)
			Expect(err).ShouldNot(HaveOccurred())

			// Act
			name := notifierDD.GetNotifierName()

			// Assert
			Expect(name).Should(Equal(string(types.NotifierDriverDatadog)))
		})
	})

	Describe("Notify", func() {
		Describe("success cases", func() {
			DescribeTable("with support object", func(ctx SpecContext, obj k8sclient.Object, event corev1.Event, expectedEventTags []string) {
				// Arrange
				expectedNotificationType := types.NotificationInfo

				clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
				clientStatsdMock.EXPECT().Event(&statsd.Event{
					Title:     utils.BuildHeaderMessageFromObjectEvent(obj, event, expectedNotificationType),
					Text:      utils.BuildBodyMessageFromObjectEvent(obj, event, false),
					AlertType: statsd.Info,
					Tags:      expectedEventTags,
				}).Return(nil)

				notifierDD, err := datadog.New(
					types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					clientStatsdMock,
				)
				Expect(err).ShouldNot(HaveOccurred())

				// Act
				err = notifierDD.Notify(ctx, obj, event, expectedNotificationType)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("with a disruption object and a generic event",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "disruption-name",
							Namespace: "disruption-namespace",
							UID:       "disruption-uid",
						},
					},
					corev1.Event{},
					[]string{"disruption_name:disruption-name"},
				),
				Entry("with a disruptionCron object and a generic event",
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "disruptioncron-name",
							Namespace: "disruptioncron-namespace",
							UID:       "disruptioncron-uid",
						},
					},
					corev1.Event{},
					[]string{"disruptioncron_name:disruptioncron-name"},
				),
			)

			DescribeTable("with unsupported object", func(ctx SpecContext, obj k8sclient.Object) {
				// Arrange
				expectedNotificationType := types.NotificationInfo

				clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
				clientStatsdMock.AssertNotCalled(GinkgoT(), "Event")

				notifierDD, err := datadog.New(
					types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					clientStatsdMock,
				)
				Expect(err).ShouldNot(HaveOccurred())

				// Act
				err = notifierDD.Notify(ctx, obj, corev1.Event{}, expectedNotificationType)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("with a pod object",
					&corev1.Pod{},
				),
				Entry("with a service object",
					&corev1.Service{},
				),
				Entry("with a node object",
					&corev1.Node{},
				),
				Entry("with a namespace object",
					&corev1.Namespace{},
				),
			)

			DescribeTable("verify event alert type selection", func(ctx SpecContext, notifType types.NotificationType, expectedEventTags statsd.EventAlertType) {
				// Arrange
				obj := &v1beta1.Disruption{}
				event := corev1.Event{}

				clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
				clientStatsdMock.EXPECT().Event(&statsd.Event{
					Title:     utils.BuildHeaderMessageFromObjectEvent(obj, event, notifType),
					Text:      utils.BuildBodyMessageFromObjectEvent(obj, event, false),
					AlertType: expectedEventTags,
					Tags:      []string{"disruption_name:"},
				}).Return(nil)

				notifierDD, err := datadog.New(
					types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					clientStatsdMock,
				)
				Expect(err).ShouldNot(HaveOccurred())

				// Act
				err = notifierDD.Notify(ctx, obj, event, notifType)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("with a NotificationInfo",
					types.NotificationInfo,
					statsd.Info,
				),
				Entry("with a NotificationCompletion",
					types.NotificationCompletion,
					statsd.Info,
				),
				Entry("with a NotificationSuccess",
					types.NotificationSuccess,
					statsd.Success,
				),
				Entry("with a NotificationWarning",
					types.NotificationWarning,
					statsd.Warning,
				),
				Entry("with a NotificationError",
					types.NotificationError,
					statsd.Error,
				),
				Entry("with a unknown",
					types.NotificationUnknown,
					statsd.Warning,
				),
			)

			DescribeTable("verify additional tags", func(ctx SpecContext, obj k8sclient.Object, event corev1.Event, expectedEventTags []string) {
				// Arrange
				notificationType := types.NotificationInfo

				clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
				clientStatsdMock.EXPECT().Event(&statsd.Event{
					Title:     utils.BuildHeaderMessageFromObjectEvent(obj, event, notificationType),
					Text:      utils.BuildBodyMessageFromObjectEvent(obj, event, false),
					AlertType: statsd.Info,
					Tags:      expectedEventTags,
				}).Return(nil)

				notifierDD, err := datadog.New(
					types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					clientStatsdMock,
				)
				Expect(err).ShouldNot(HaveOccurred())

				// Act
				err = notifierDD.Notify(ctx, obj, event, notificationType)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("a disruption with a team selector",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruption-name",
						},
						Spec: v1beta1.DisruptionSpec{
							Selector: map[string]string{
								"team": "team-name",
							},
						},
					},
					corev1.Event{},
					[]string{"team:team-name", "disruption_name:disruption-name"},
				),
				Entry("a disruption with a service",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruption-name",
						},
						Spec: v1beta1.DisruptionSpec{
							Selector: map[string]string{
								"service": "service-name",
							},
						},
					},
					corev1.Event{},
					[]string{"service:service-name", "disruption_name:disruption-name"},
				),
				Entry("a disruption with an app",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruption-name",
						},
						Spec: v1beta1.DisruptionSpec{
							Selector: map[string]string{
								"app": "app-name",
							},
						},
					},
					corev1.Event{},
					[]string{"app:app-name", "disruption_name:disruption-name"},
				),
				Entry("a disruption with an event which contains a target name",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruption-name",
						},
					},
					corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"target_name": "target-name",
							},
						},
					},
					[]string{"disruption_name:disruption-name", "target_name:target-name"},
				),
				Entry("a disruption with an event with all supported selectors and an event which contains target_name annotation",
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruption-name",
						},
						Spec: v1beta1.DisruptionSpec{
							Selector: map[string]string{
								"team":    "team-name",
								"service": "service-name",
								"app":     "app-name",
							},
						},
					},
					corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"target_name": "target-name",
							},
						},
					},
					[]string{"team:team-name", "service:service-name", "app:app-name", "disruption_name:disruption-name", "target_name:target-name"},
				),
				Entry("a disruption cron with an event which contains a target name",
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruptioncron-name",
						},
					},
					corev1.Event{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"target_name": "target-name",
							},
						},
					},
					[]string{"disruptioncron_name:disruptioncron-name", "target_name:target-name"},
				),
				Entry("a disruption cron with a simple event",
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name: "disruptioncron-name",
						},
					},
					corev1.Event{},
					[]string{"disruptioncron_name:disruptioncron-name"},
				),
			)
		})

		Describe("error cases", func() {
			DescribeTableSubtree("the client fails to send the event", func(obj k8sclient.Object) {
				It("should return an error", func(ctx SpecContext) {
					// Arrange
					event := corev1.Event{}

					clientStatsdMock := mocks.NewClientStatsdMock(GinkgoT())
					clientStatsdMock.EXPECT().Event(mock.Anything).Return(fmt.Errorf("error sending event"))

					notifierDD, err := datadog.New(
						types.NotifiersCommonConfig{
							ClusterName: "cluster-name",
						},
						clientStatsdMock,
					)
					Expect(err).ShouldNot(HaveOccurred())

					// Act
					err = notifierDD.Notify(ctx, obj, event, types.NotificationInfo)

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("error sending event"))
				})
			},
				Entry("with a disruption object",
					&v1beta1.Disruption{},
				),
				Entry("with a disruption cron object",
					&v1beta1.DisruptionCron{},
				),
			)
		})
	})
})

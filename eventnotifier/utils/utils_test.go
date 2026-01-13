// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package utils_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildBodyMessageFromObjectEvent", func() {
	var (
		event corev1.Event
	)

	BeforeEach(func() {
		// Setup a common event for testing
		event = corev1.Event{
			Reason:         "TestReason",
			Message:        "This is a test message",
			InvolvedObject: corev1.ObjectReference{},
		}
	})

	DescribeTable("Generating message body from object event",
		func(obj k8sclient.Object, expectedMarkdown, expectedNoMarkdown string) {
			// Arrange
			event.InvolvedObject.Kind = obj.GetObjectKind().GroupVersionKind().Kind
			obj.SetName("test-object")

			// Act
			By("Test with markdown enabled")
			result := utils.BuildBodyMessageFromObjectEvent(obj, event, true)
			Expect(result).To(Equal(expectedMarkdown))

			By("Test with markdown disabled")
			result = utils.BuildBodyMessageFromObjectEvent(obj, event, false)
			Expect(result).To(Equal(expectedNoMarkdown))
		},
		Entry("for a Pod object",
			&corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind: "Pod",
				},
			},
			"> Pod `test-object` emitted the event `TestReason`: This is a test message",
			"Pod 'test-object' emitted the event TestReason: This is a test message",
		),
		Entry("for a Deployment object",
			&appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
			},
			"> Deployment `test-object` emitted the event `TestReason`: This is a test message",
			"Deployment 'test-object' emitted the event TestReason: This is a test message",
		),
		Entry("for a DisruptionCron object",
			&v1beta1.DisruptionCron{
				TypeMeta: metav1.TypeMeta{
					Kind: v1beta1.DisruptionCronKind,
				},
			},
			"> DisruptionCron `test-object` emitted the event `TestReason`: This is a test message",
			"DisruptionCron 'test-object' emitted the event TestReason: This is a test message",
		),
	)
})

var _ = Describe("BuildHeaderMessageFromObjectEvent", func() {
	var (
		event corev1.Event
	)

	BeforeEach(func() {
		// Setup a fake Kubernetes object and event for testing
		event = corev1.Event{
			InvolvedObject: corev1.ObjectReference{},
			Reason:         "TestReason",
			Message:        "This is a test message",
		}
	})

	DescribeTable("Generating header message from object event",
		func(obj k8sclient.Object, notifType types.NotificationType, expectedMessage string) {
			// Arrange
			event.InvolvedObject.Kind = obj.GetObjectKind().GroupVersionKind().Kind
			obj.SetName("test-object")

			// Act
			result := utils.BuildHeaderMessageFromObjectEvent(obj, event, notifType)

			// Assert
			Expect(result).To(Equal(expectedMessage))
		},
		Entry("with NotificationCompletion type for a Pod object", &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
		}, types.NotificationCompletion, "Pod 'test-object' is finished or terminating."),
		Entry("with NotificationSuccess type for a Deployment object", &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "Deployment",
			},
		}, types.NotificationSuccess, "Deployment 'test-object' received a recovery notification."),
		Entry("with NotificationInfo type for a DisruptionCron object", &v1beta1.DisruptionCron{
			TypeMeta: metav1.TypeMeta{
				Kind: v1beta1.DisruptionCronKind,
			},
		}, types.NotificationInfo, "DisruptionCron 'test-object' received a notification."),
		Entry("with NotificationError type for a Service object", &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind: "Service",
			},
		}, types.NotificationError, "Service 'test-object' encountered an issue."),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package slack

import (
	"context"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	cLog "github.com/DataDog/chaos-controller/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Slack cron reporting threshold", func() {
	DescribeTable("filters the reporting channel without filtering the mirror", func(minimum, notification types.NotificationType, deliver bool) {
		client := newSlackNotifierMock(GinkgoT())
		client.EXPECT().PostMessage(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", nil)
		notifier := Notifier{client: client, config: NotifierSlackConfig{MirrorSlackChannelID: "mirror"}}
		cron := &v1beta1.DisruptionCron{
			TypeMeta:   metav1.TypeMeta{Kind: v1beta1.DisruptionCronKind},
			ObjectMeta: metav1.ObjectMeta{Name: "scheduled", Namespace: "default"},
		}
		cron.Spec.Reporting = &v1beta1.Reporting{SlackChannel: "reporting", MinNotificationType: minimum}
		event := corev1.Event{
			InvolvedObject: corev1.ObjectReference{Kind: v1beta1.DisruptionCronKind, Name: cron.Name},
			Reason:         string(v1beta1.EventDisruptionCronCreated),
		}
		ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
		Expect(notifier.Notify(ctx, cron, event, notification)).To(Succeed())
		var channels []string
		for _, call := range client.Calls {
			if call.Method == "PostMessage" {
				channels = append(channels, call.Arguments.String(0))
			}
		}
		expected := []string{"mirror"}
		if deliver {
			expected = append(expected, "reporting")
		}
		Expect(channels).To(ConsistOf(expected))
	},
		Entry("default suppresses Info", types.NotificationUnknown, types.NotificationInfo, false),
		Entry("Success suppresses Info", types.NotificationSuccess, types.NotificationInfo, false),
		Entry("Warning suppresses Info", types.NotificationWarning, types.NotificationInfo, false),
		Entry("Warning allows Warning", types.NotificationWarning, types.NotificationWarning, true),
		Entry("Error suppresses Warning", types.NotificationError, types.NotificationWarning, false),
		Entry("Info explicitly allows Info", types.NotificationInfo, types.NotificationInfo, true),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package slack

import (
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/slack-go/slack"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNotifier_Notify(t *testing.T) {
	t.Parallel()

	type callContext struct {
		mirrorSlackChannelID string
		userName             string
		notifType            types.NotificationType
		reporting            *v1beta1.Reporting
	}

	tests := []struct {
		name        string
		callContext callContext
		setup       func(mock.TestingT, *slackNotifierMock, callContext)
		wantErr     string
	}{
		{
			name: "notify info no mirror does not send any message",
			callContext: callContext{
				notifType: types.NotificationInfo,
			},
		},
		{
			name: "notify warn no mirror does not send any message and returns err invalid email",
			callContext: callContext{
				notifType: types.NotificationWarning,
			},
			wantErr: "slack notifier: invalid user info email in disruption : mail: no address",
		},
		{
			name: "notify info with mirror send message to provided channel",
			callContext: callContext{
				notifType:            types.NotificationInfo,
				mirrorSlackChannelID: "chaos-notif",
			},
			setup: func(t mock.TestingT, msn *slackNotifierMock, args callContext) {
				msn.EXPECT().PostMessage(
					args.mirrorSlackChannelID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once()
			},
		},
		{
			name: "notify warn with mirror send user and slack channel notifications",
			callContext: callContext{
				mirrorSlackChannelID: "chaos-notif",
				userName:             "valid@email.org",
				notifType:            types.NotificationWarning,
			},
			setup: func(t mock.TestingT, msn *slackNotifierMock, args callContext) {
				msn.EXPECT().PostMessage(
					args.mirrorSlackChannelID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once()

				userID := "slack-user-id"
				getUserEmailCall := msn.EXPECT().GetUserByEmail(args.userName).Return(&slack.User{
					ID: userID,
				}, nil).Call

				msn.EXPECT().PostMessage(
					userID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once().NotBefore(getUserEmailCall)
			},
		},
		{
			name: "notify warn with reporting send user and custom slack channel notifications",
			callContext: callContext{
				userName:  "valid@email.org",
				notifType: types.NotificationWarning,
				reporting: &v1beta1.Reporting{
					SlackChannel: "custom-slack-channel",
				},
			},
			setup: func(t mock.TestingT, msn *slackNotifierMock, args callContext) {
				msn.EXPECT().PostMessage(
					args.reporting.SlackChannel,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once()

				userID := "slack-user-id"
				getUserEmailCall := msn.EXPECT().GetUserByEmail(args.userName).Return(&slack.User{
					ID: userID,
				}, nil).Call

				msn.EXPECT().PostMessage(
					userID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once().NotBefore(getUserEmailCall)
			},
		},
		{
			name: "notify warn with reporting error send user channel notifications only",
			callContext: callContext{
				userName:  "valid@email.org",
				notifType: types.NotificationWarning,
				reporting: &v1beta1.Reporting{
					SlackChannel:        "custom-slack-channel",
					MinNotificationType: types.NotificationError,
				},
			},
			setup: func(t mock.TestingT, msn *slackNotifierMock, args callContext) {
				userID := "slack-user-id"
				getUserEmailCall := msn.EXPECT().GetUserByEmail(args.userName).Return(&slack.User{
					ID: userID,
				}, nil).Call

				msn.EXPECT().PostMessage(
					userID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once().NotBefore(getUserEmailCall)
			},
		},
		{
			name: "notify success with reporting and mirror send 3 notifications",
			callContext: callContext{
				mirrorSlackChannelID: "chaos-notif",
				userName:             "valid@email.org",
				notifType:            types.NotificationSuccess,
				reporting: &v1beta1.Reporting{
					SlackChannel: "custom-slack-channel",
				},
			},
			setup: func(t mock.TestingT, msn *slackNotifierMock, args callContext) {
				msn.EXPECT().PostMessage(
					args.mirrorSlackChannelID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once()

				msn.EXPECT().PostMessage(
					args.reporting.SlackChannel,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once()

				userID := "slack-user-id"
				getUserEmailCall := msn.EXPECT().GetUserByEmail(args.userName).Return(&slack.User{
					ID: userID,
				}, nil).Call

				msn.EXPECT().PostMessage(
					userID,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return("", "", nil).Once().NotBefore(getUserEmailCall)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require := require.New(t)

			slackClient := newSlackNotifierMock(t)

			logger := zaptest.NewLogger(t)

			n := &Notifier{
				client: slackClient,
				common: types.NotifiersCommonConfig{},
				config: NotifierSlackConfig{
					MirrorSlackChannelID: tt.callContext.mirrorSlackChannelID,
				},
				logger: logger.Sugar(),
			}

			e := corev1.Event{
				Message: "some message",
				Reason:  "some reason",
			}

			d := v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			}
			d.SetUserInfo(v1.UserInfo{
				Username: tt.callContext.userName,
			})
			d.Spec.Reporting = tt.callContext.reporting

			if tt.setup != nil {
				tt.setup(t, slackClient, tt.callContext)
			}

			if err := n.Notify(d, e, tt.callContext.notifType); (err != nil) || tt.wantErr != "" {
				require.EqualError(err, tt.wantErr)
			}
		})
	}
}

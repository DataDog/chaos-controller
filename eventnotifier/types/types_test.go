// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationType_Allows(t *testing.T) {
	t.Parallel()

	allNotificationTypes := []NotificationType{
		NotificationUnknown,
		NotificationInfo,
		NotificationSuccess,
		NotificationWarning,
		NotificationError,
	}

	tests := []struct {
		name        string
		level       NotificationType
		wantAllowed map[NotificationType]struct{}
	}{
		{
			name:  "Unknown notification does not allow info",
			level: NotificationUnknown,
			wantAllowed: map[NotificationType]struct{}{
				NotificationUnknown: {},
				NotificationSuccess: {},
				NotificationWarning: {},
				NotificationError:   {},
			},
		},
		{
			name:  "Info notification allows everything",
			level: NotificationInfo,
			wantAllowed: map[NotificationType]struct{}{
				NotificationUnknown: {},
				NotificationInfo:    {},
				NotificationSuccess: {},
				NotificationWarning: {},
				NotificationError:   {},
			},
		},
		{
			name:  "Success notification does not allow info",
			level: NotificationSuccess,
			wantAllowed: map[NotificationType]struct{}{
				NotificationUnknown: {},
				NotificationSuccess: {},
				NotificationWarning: {},
				NotificationError:   {},
			},
		},
		{
			name:  "Warning notification allows warning and error",
			level: NotificationWarning,
			wantAllowed: map[NotificationType]struct{}{
				NotificationWarning: {},
				NotificationError:   {},
			},
		},
		{
			name:  "Error notification allows only error",
			level: NotificationError,
			wantAllowed: map[NotificationType]struct{}{
				NotificationError: {},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, notifType := range allNotificationTypes {
				got := tt.level.Allows(notifType)

				_, ok := tt.wantAllowed[notifType]
				require.Equal(t, ok, got, "Notification type [%s] expect [%s] to be allowed==%v", tt.level, notifType, ok)
			}
		})
	}
}

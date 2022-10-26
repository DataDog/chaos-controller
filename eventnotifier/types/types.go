// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package types

type NotifiersCommonConfig struct {
	ClusterName string
}

type NotifierDriver string

const (
	// NotifierDriverSlack is the Slack driver
	NotifierDriverSlack NotifierDriver = "slack"

	// NotifierDriverNoop is a noop driver mainly used for testing
	NotifierDriverNoop NotifierDriver = "noop"

	// NotifierDriverDatadog is the Datadog driver
	NotifierDriverDatadog NotifierDriver = "datadog"

	// NotifierDriverHTTP is the HTTP driver
	NotifierDriverHTTP NotifierDriver = "http"
)

type NotificationType string

const (
	NotificationUnknown NotificationType = ""
	NotificationSuccess NotificationType = "Success"
	NotificationInfo    NotificationType = "Info"
	NotificationWarning NotificationType = "Warning"
	NotificationError   NotificationType = "Error"
)

// Allows determines if provided notif is above or equal to checked notificationType
// We treat notifications similarly to log levels excluding all NotificationType strictly below defined NotificationType
// Hence, Success will allow Success and above (Warning, Error)...
func (n NotificationType) Allows(notif NotificationType) bool {
	switch n {
	case NotificationSuccess, NotificationUnknown:
		return notif != NotificationInfo
	case NotificationWarning:
		return notif == NotificationWarning || notif == NotificationError
	case NotificationError:
		return notif == NotificationError
	default:
		return true
	}
}

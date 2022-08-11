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
	NotificationSuccess NotificationType = "Success"
	NotificationInfo    NotificationType = "Info"
	NotificationWarning NotificationType = "Warning"
	NotificationError   NotificationType = "Error"
)

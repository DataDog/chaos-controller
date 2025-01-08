// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package utils

import (
	"fmt"

	"github.com/DataDog/chaos-controller/eventnotifier/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildBodyMessageFromObjectEvent Templated body text to send to notifiers
func BuildBodyMessageFromObjectEvent(obj client.Object, event corev1.Event, isMarkdown bool) string {
	messagePrefix := formatMessagePrefix(obj, event, isMarkdown)

	if isMarkdown {
		return messagePrefix + " emitted the event `" + event.Reason + "`: " + event.Message
	}

	return messagePrefix + " emitted the event " + event.Reason + ": " + event.Message
}

// BuildHeaderMessageFromObjectEvent Templated header text to send to notifiers
func BuildHeaderMessageFromObjectEvent(obj client.Object, event corev1.Event, notifType types.NotificationType) string {
	messagePrefix := formatMessagePrefix(obj, event, false)

	switch notifType {
	case types.NotificationCompletion:
		return messagePrefix + " is finished or terminating."
	case types.NotificationSuccess:
		return messagePrefix + " received a recovery notification."
	case types.NotificationInfo:
		return messagePrefix + " received a notification."
	default:
		return messagePrefix + " encountered an issue."
	}
}

func formatMessagePrefix(obj client.Object, event corev1.Event, isMarkdown bool) string {
	template := "%s '%s'"

	if isMarkdown {
		template = "> %s `%s`"
	}

	return fmt.Sprintf(template, event.InvolvedObject.Kind, obj.GetName())
}

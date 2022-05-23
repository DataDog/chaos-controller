// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.
package utils

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// BuildBodyMessageFromDisruptionEvent Templated body text to send to notifiers
func BuildBodyMessageFromDisruptionEvent(dis v1beta1.Disruption, event corev1.Event) string {
	return "> Disruption `" + dis.Name + "` emitted the event `" + event.Reason + "`: " + event.Message
}

// BuildHeaderMessageFromDisruptionEvent Templated header text to send to notifiers
func BuildHeaderMessageFromDisruptionEvent(dis v1beta1.Disruption, event corev1.Event) string {
	if event.Type == corev1.EventTypeWarning {
		return "Disruption '" + dis.Name + "' encountered an issue."
	}

	return "Disruption '" + dis.Name + "' an unusual event."
}

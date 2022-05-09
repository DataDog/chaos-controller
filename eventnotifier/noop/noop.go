// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"log"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	corev1 "k8s.io/api/core/v1"
)

type NotifierNoopConfig struct {
	Enabled bool
}

// Notifier describes a NOOP notifier
type Notifier struct{}

// New NOOP Notifier
func New() *Notifier {
	return &Notifier{}
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverNoop)
}

// NotifyWarning generates a notification for generiv k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	notify("Notifier Warning: "+event.Reason+" - "+event.Message, dis.Name)

	return nil
}

// helper for noop notifier
func notify(notificationName string, disName string) {
	log.Printf("\nNOOP: %s for disruption %s\n", notificationName, disName)
}

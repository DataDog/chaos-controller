// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	corev1 "k8s.io/api/core/v1"
)

// Notifier describes a NOOP notifier
type Notifier struct{}

// New NOOP Notifier
func New() *Notifier {
	return &Notifier{}
}

// Close returns nil
func (n *Notifier) Clean() error {
	return nil
}

// GetNotifierName returns NOOP
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverNoop)
}

func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	notify("warning: "+event.Name+" - "+event.Message, dis.Name)

	return nil
}

// NotifyNoTarget signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyNoTarget(dis v1beta1.Disruption) error {
	notify("NotifyNoTarget", dis.Name)

	return nil
}

// NotifyStuckOnRemoval signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyStuckOnRemoval(dis v1beta1.Disruption) error {
	notify("NotifyStuckOnRemoval", dis.Name)

	return nil
}

// helper for noop notifier
func notify(notificationName string, disName string) {
	fmt.Printf("NOOP: %s for disruption %s\n", notificationName, disName)
}

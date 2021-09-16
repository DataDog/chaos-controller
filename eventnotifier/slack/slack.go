// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
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
	return string(types.NotifierDriverSlack)
}

// NotifyNotInjected signals a disruption was injected successfully
func (n *Notifier) NotifyNotInjected(dis v1beta1.Disruption) error {
	notify("NotifyNotInjected", dis.Name)

	return nil
}

// NotifyInjected signals a disruption was injected successfully
func (n *Notifier) NotifyInjected(dis v1beta1.Disruption) error {
	notify("NotifyInjected", dis.Name)

	return nil
}

// NotifyCleanedUp signals a disruption's been cleaned up successfully
func (n *Notifier) NotifyCleanedUp(dis v1beta1.Disruption) error {
	notify("NotifyCleanedUp", dis.Name)

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
	fmt.Printf("SLACK: %s for disruption %s\n", notificationName, disName)
}

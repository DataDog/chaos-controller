// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/notifier/types"
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

// NotifyInjected signals a disruption was injected successfully
func (n *Notifier) NotifyInjected(dis v1beta1.Disruption) error {
	notify("NotifyInjected", dis.Name)

	return nil
}

// // NotifyInvalidated signals a disruption was rejected by the admission controller validation
// func (n *Notifier) NotifyInvalidated(dis v1beta1.Disruption) error {
// 	notify("NotifyInvalidated", dis.Name)

// 	return nil
// }

// // NotifyNoTarget signals a disruption's selector found no target
// func (n *Notifier) NotifyNoTarget(dis v1beta1.Disruption) error {
// 	notify("NotifyNoTarget", dis.Name)

// 	return nil
// }

// NotifyCleaned signals a disruption's been cleaned up suffessfully
func (n *Notifier) NotifyCleaned(dis v1beta1.Disruption) error {
	notify("NotifyCleaned", dis.Name)

	return nil
}

// // NotifyNotCleaned signals a disruption's cleanup has failed
// func (n *Notifier) NotifyNotCleaned(dis v1beta1.Disruption) error {
// 	notify("NotifyNotCleaned", dis.Name)

// 	return nil
// }

// helper for noop notifier
func notify(notificationName string, disName string) {
	fmt.Printf("NOOP: %s for disruption %s\n", notificationName, disName)
}

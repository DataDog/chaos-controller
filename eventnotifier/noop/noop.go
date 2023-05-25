// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifierNoopConfig struct {
	Enabled bool
}

// Notifier describes a NOOP notifier
type Notifier struct {
	log *zap.SugaredLogger
}

// New NOOP Notifier
func New(log *zap.SugaredLogger) Notifier {
	return Notifier{
		log,
	}
}

// GetNotifierName returns the driver's name
func (n Notifier) GetNotifierName() string {
	return string(types.NotifierDriverNoop)
}

// NotifyWarning generates a notification for generiv k8s Warning events
func (n Notifier) Notify(dis v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	n.log.Debugf("NOOP: %s for disruption %s\n", fmt.Sprintf("Notifier %s: %s - %s", string(notifType), event.Reason, event.Message), dis.Name)

	return nil
}

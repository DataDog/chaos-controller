// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotifierNoopConfig struct {
	Enabled bool `yaml:"enabled"`
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

// Notify generates a notification for generic k8s events
func (n Notifier) Notify(obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	notifierMessage := fmt.Sprintf("Notifier %s: %s - %s", string(notifType), event.Reason, event.Message)

	switch d := obj.(type) {
	case *v1beta1.Disruption:
		n.log.Debugf("NOOP: %s for disruption %s\n", notifierMessage, d.Name)
	case *v1beta1.DisruptionCron:
		n.log.Debugf("NOOP: %s for disruption cron %s\n", notifierMessage, d.Name)
	}

	return nil
}

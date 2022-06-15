// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package datadog

import (
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/DataDog/datadog-go/statsd"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifierDatadogConfig struct {
	Enabled bool
}

// Notifier describes a Datadog notifier
type Notifier struct {
	client *statsd.Client
	common types.NotifiersCommonConfig
	logger *zap.SugaredLogger
}

// New Datadog Notifier
func New(commonConfig types.NotifiersCommonConfig, datadogConfig NotifierDatadogConfig, logger *zap.SugaredLogger) (*Notifier, error) {
	not := &Notifier{
		common: commonConfig,
		logger: logger,
	}

	url := os.Getenv("STATSD_URL")

	instance, err := statsd.New(url, statsd.WithTags([]string{"app:chaos-controller"}))
	if err != nil {
		return nil, err
	}

	not.client = instance
	not.logger.Info("notifier: datadog notifier connected to datadog")

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverDatadog)
}

func (n *Notifier) buildDatadogEventTags(dis v1beta1.Disruption) {
	if n.common.ClusterName == "" {
		if dis.ClusterName != "" {
			n.common.ClusterName = dis.ClusterName
		} else {
			n.common.ClusterName = "n/a"
		}
	}

	if team := dis.Spec.Selector.Get("team"); team != "" {
		n.client.Tags = append(n.client.Tags, "team:"+team)
	}

	if service := dis.Spec.Selector.Get("app"); service != "" {
		n.client.Tags = append(n.client.Tags, "service:"+service)
	}
}

func (n *Notifier) sendEvent(headerText, bodyText string, alertType statsd.EventAlertType) error {
	event := statsd.Event{
		Title:     headerText,
		Text:      bodyText,
		AlertType: alertType,
	}

	return n.client.Event(&event)
}

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := utils.BuildHeaderMessageFromDisruptionEvent(dis, event)
	bodyText := utils.BuildBodyMessageFromDisruptionEvent(dis, event, false)

	n.buildDatadogEventTags(dis)
	n.logger.Debugw("notifier: sending notifier event to datadog", "disruption", dis.Name, "eventType", event.Type, "message", bodyText)

	return n.sendEvent(headerText, bodyText, statsd.Warning)
}

// NotifyRecovery generates a notification for generic k8s Normal events
func (n *Notifier) NotifyRecovery(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := utils.BuildHeaderMessageFromDisruptionEvent(dis, event)
	bodyText := utils.BuildBodyMessageFromDisruptionEvent(dis, event, false)

	n.buildDatadogEventTags(dis)
	n.logger.Debugw("notifier: sending notifier event to datadog", "disruption", dis.Name, "eventType", event.Type, "message", bodyText)

	return n.sendEvent(headerText, bodyText, statsd.Success)
}

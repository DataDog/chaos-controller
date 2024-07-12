// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package datadog

import (
	"os"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/DataDog/datadog-go/statsd"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func New(commonConfig types.NotifiersCommonConfig, _ NotifierDatadogConfig, logger *zap.SugaredLogger) (*Notifier, error) {
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

// Notify generates a notification for generic k8s events
func (n *Notifier) Notify(obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	dis, ok := obj.(*v1beta1.Disruption)
	if !ok {
		return nil
	}

	eventType := statsd.Warning

	switch notifType {
	case types.NotificationInfo, types.NotificationCompletion:
		eventType = statsd.Info
	case types.NotificationSuccess:
		eventType = statsd.Success
	case types.NotificationWarning:
		eventType = statsd.Warning
	case types.NotificationError:
		eventType = statsd.Error
	}

	headerText := utils.BuildHeaderMessageFromObjectEvent(dis, event, notifType)
	bodyText := utils.BuildBodyMessageFromObjectEvent(dis, event, false)
	additionalTags := n.buildDatadogEventTags(*dis, event)

	n.logger.Debugw("notifier: sending notifier event to datadog", "disruptionName", dis.Name, "eventType", event.Type, "message", bodyText, "datadogTags", strings.Join(additionalTags, ", "))

	return n.sendEvent(headerText, bodyText, eventType, additionalTags)
}

func (n *Notifier) buildDatadogEventTags(dis v1beta1.Disruption, event corev1.Event) []string {
	additionalTags := []string{}

	if team := dis.Spec.Selector.Get("team"); team != "" {
		additionalTags = append(additionalTags, "team:"+team)
	}

	if service := dis.Spec.Selector.Get("service"); service != "" {
		additionalTags = append(additionalTags, "service:"+service)
	}

	if app := dis.Spec.Selector.Get("app"); app != "" {
		additionalTags = append(additionalTags, "app:"+app)
	}

	additionalTags = append(additionalTags, "disruption_name:"+dis.Name)

	if targetName, ok := event.Annotations["target_name"]; ok {
		additionalTags = append(additionalTags, "target_name:"+targetName)
	}

	return additionalTags
}

func (n *Notifier) sendEvent(headerText, bodyText string, alertType statsd.EventAlertType, tags []string) error {
	event := statsd.Event{
		Title:     headerText,
		Text:      bodyText,
		AlertType: alertType,
		Tags:      tags,
	}

	return n.client.Event(&event)
}

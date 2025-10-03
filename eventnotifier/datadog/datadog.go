// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package datadog

import (
	"os"
	"strings"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	cLog "github.com/DataDog/chaos-controller/log"
	tagutil "github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/datadog-go/statsd"
)

type NotifierDatadogConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Notifier describes a Datadog notifier
type Notifier struct {
	client statsd.ClientInterface
	common types.NotifiersCommonConfig
	logger *zap.SugaredLogger
}

// New Datadog Notifier
func New(commonConfig types.NotifiersCommonConfig, _ NotifierDatadogConfig, logger *zap.SugaredLogger, statsdClient statsd.ClientInterface) (*Notifier, error) {
	not := &Notifier{
		common: commonConfig,
		logger: logger,
	}

	url := os.Getenv("STATSD_URL")

	if statsdClient == nil {
		instance, err := statsd.New(url, statsd.WithTags([]string{"app:chaos-controller"}))
		if err != nil {
			return nil, err
		}

		not.client = instance
	} else {
		not.client = statsdClient
	}

	not.logger.Info("notifier: datadog notifier connected to datadog")

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverDatadog)
}

// Notify generates a notification for generic k8s events
func (n *Notifier) Notify(obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	switch d := obj.(type) {
	case *v1beta1.Disruption:
		return n.notifyDisruption(d, event, notifType)
	case *v1beta1.DisruptionCron:
		return n.notifyDisruptionCron(d, event, notifType)
	}

	n.logger.Debugw("notifier: skipping datadog notification for object", "object", obj)

	return nil
}

func (n *Notifier) notifyDisruption(d *v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	n.logger.With(cLog.DisruptionNameKey, d.Name, cLog.DisruptionNamespaceKey, d.Namespace)

	eventType := n.getEventAlertType(notifType)

	headerText := utils.BuildHeaderMessageFromObjectEvent(d, event, notifType)
	bodyText := utils.BuildBodyMessageFromObjectEvent(d, event, false)
	additionalTags := n.buildDatadogEventTagsForDisruption(*d, event)

	return n.sendEvent(headerText, bodyText, eventType, additionalTags)
}

func (n *Notifier) notifyDisruptionCron(d *v1beta1.DisruptionCron, event corev1.Event, notifType types.NotificationType) error {
	n.logger.With(cLog.DisruptionCronNameKey, d.Name, cLog.DisruptionCronNamespaceKey, d.Namespace)

	eventType := n.getEventAlertType(notifType)

	headerText := utils.BuildHeaderMessageFromObjectEvent(d, event, notifType)
	bodyText := utils.BuildBodyMessageFromObjectEvent(d, event, false)
	additionalTags := n.buildDatadogEventTagsForDisruptionCron(*d, event)

	n.logger.Debugw("notifier: sending notifier event to datadog", "eventType", event.Type, "message", bodyText, "datadogTags", strings.Join(additionalTags, ", "))

	return n.sendEvent(headerText, bodyText, eventType, additionalTags)
}

func (n *Notifier) getEventAlertType(notifType types.NotificationType) statsd.EventAlertType {
	switch notifType {
	case types.NotificationInfo, types.NotificationCompletion:
		return statsd.Info
	case types.NotificationSuccess:
		return statsd.Success
	case types.NotificationError:
		return statsd.Error
	}

	return statsd.Warning
}

func (n *Notifier) buildDatadogEventTagsForDisruption(dis v1beta1.Disruption, event corev1.Event) []string {
	var additionalTags []string

	if team := dis.Spec.Selector.Get("team"); team != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag("team", team))
	}

	if service := dis.Spec.Selector.Get("service"); service != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag("service", service))
	}

	if app := dis.Spec.Selector.Get("app"); app != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag("app", app))
	}

	additionalTags = append(additionalTags, tagutil.FormatTag("disruption_name", dis.Name))

	if targetName, ok := event.Annotations["target_name"]; ok {
		additionalTags = append(additionalTags, tagutil.FormatTag("target_name", targetName))
	}

	return additionalTags
}

func (n *Notifier) buildDatadogEventTagsForDisruptionCron(dis v1beta1.DisruptionCron, event corev1.Event) []string {
	additionalTags := []string{
		tagutil.FormatTag("disruptioncron_name", dis.Name),
	}

	if targetName, ok := event.Annotations["target_name"]; ok {
		additionalTags = append(additionalTags, tagutil.FormatTag("target_name", targetName))
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

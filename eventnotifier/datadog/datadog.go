// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package datadog

import (
	"context"
	"os"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	cLog "github.com/DataDog/chaos-controller/log"
	tagutil "github.com/DataDog/chaos-controller/o11y/tags"
)

type NotifierDatadogConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Notifier describes a Datadog notifier
type Notifier struct {
	client statsd.ClientInterface
	common types.NotifiersCommonConfig
}

// New Datadog Notifier
func New(commonConfig types.NotifiersCommonConfig, statsdClient statsd.ClientInterface) (*Notifier, error) {
	not := &Notifier{
		common: commonConfig,
	}

	url := os.Getenv("STATSD_URL")

	if statsdClient == nil {
		instance, err := statsd.New(url, statsd.WithTags([]string{
			tagutil.FormatTag(tagutil.AppKey, "chaos-controller"),
		}))
		if err != nil {
			return nil, err
		}

		not.client = instance
	} else {
		not.client = statsdClient
	}

	return not, nil
}

// GetNotifierName returns the driver's name
func (n *Notifier) GetNotifierName() string {
	return string(types.NotifierDriverDatadog)
}

// Notify generates a notification for generic k8s events
func (n *Notifier) Notify(ctx context.Context, obj client.Object, event corev1.Event, notifType types.NotificationType) error {
	switch d := obj.(type) {
	case *v1beta1.Disruption:
		return n.notifyDisruption(ctx, d, event, notifType)
	case *v1beta1.DisruptionCron:
		return n.notifyDisruptionCron(ctx, d, event, notifType)
	}

	cLog.FromContext(ctx).Debugw("notifier: skipping datadog notification for object", tagutil.ObjectKey, obj)

	return nil
}

func (n *Notifier) notifyDisruption(_ context.Context, d *v1beta1.Disruption, event corev1.Event, notifType types.NotificationType) error {
	eventType := n.getEventAlertType(notifType)

	headerText := utils.BuildHeaderMessageFromObjectEvent(d, event, notifType)
	bodyText := utils.BuildBodyMessageFromObjectEvent(d, event, false)
	additionalTags := n.buildDatadogEventTagsForDisruption(*d, event)

	return n.sendEvent(headerText, bodyText, eventType, additionalTags)
}

func (n *Notifier) notifyDisruptionCron(ctx context.Context, d *v1beta1.DisruptionCron, event corev1.Event, notifType types.NotificationType) error {
	logger := cLog.FromContext(ctx).With(
		tagutil.DisruptionCronNameKey, d.Name,
		tagutil.DisruptionCronNamespaceKey, d.Namespace,
		tagutil.EventTypeKey, event.Type,
	)

	eventType := n.getEventAlertType(notifType)

	headerText := utils.BuildHeaderMessageFromObjectEvent(d, event, notifType)
	bodyText := utils.BuildBodyMessageFromObjectEvent(d, event, false)
	additionalTags := n.buildDatadogEventTagsForDisruptionCron(*d, event)

	logger.Debugw("notifier: sending notifier event to datadog",
		tagutil.MessageKey, bodyText,
		tagutil.DatadogTagsKey, strings.Join(additionalTags, ", "),
	)

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

	if team := dis.Spec.Selector.Get(tagutil.TeamKey); team != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.TeamKey, team))
	}

	if service := dis.Spec.Selector.Get(tagutil.ServiceKey); service != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.ServiceKey, service))
	}

	if app := dis.Spec.Selector.Get(tagutil.AppKey); app != "" {
		additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.AppKey, app))
	}

	additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.DisruptionNameKey, dis.Name))

	if targetName, ok := event.Annotations[tagutil.TargetNameKey]; ok {
		additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.TargetNameKey, targetName))
	}

	return additionalTags
}

func (n *Notifier) buildDatadogEventTagsForDisruptionCron(dis v1beta1.DisruptionCron, event corev1.Event) []string {
	additionalTags := []string{
		tagutil.FormatTag(tagutil.DisruptionCronNameKey, dis.Name),
	}

	if targetName, ok := event.Annotations[tagutil.TargetNameKey]; ok {
		additionalTags = append(additionalTags, tagutil.FormatTag(tagutil.TargetNameKey, targetName))
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

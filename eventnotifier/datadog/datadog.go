// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package datadog

import (
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
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

// NotifyWarning generates a notification for generic k8s Warning events
func (n *Notifier) NotifyWarning(dis v1beta1.Disruption, event corev1.Event) error {
	headerText := "Disruption '" + dis.Name + "' encountered an issue."
	bodyText := "> Disruption `" + dis.Name + "` emitted the event " + event.Reason + ": " + event.Message

	if n.common.ClusterName == "" {
		if dis.ClusterName != "" {
			n.common.ClusterName = dis.ClusterName
		} else {
			n.common.ClusterName = "n/a"
		}
	}

	n.logger.Info("notifier: sending notifier event to datadog")

	if team := dis.Spec.Selector.Get("team"); team != "" {
		n.client.Tags = append(n.client.Tags, "team:"+team)
	}

	if service := dis.Spec.Selector.Get("app"); service != "" {
		n.client.Tags = append(n.client.Tags, "service:"+service)
	}

	err := n.client.SimpleEvent(headerText, bodyText)
	if err != nil {
		return err
	}

	return nil
}

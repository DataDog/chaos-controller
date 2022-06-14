// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventnotifier

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/datadog"
	"github.com/DataDog/chaos-controller/eventnotifier/noop"
	"github.com/DataDog/chaos-controller/eventnotifier/slack"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NotifiersConfig struct {
	Common  types.NotifiersCommonConfig   `json:"notifiersCommonConfig"`
	Noop    noop.NotifierNoopConfig       `json:"notifierNoopConfig"`
	Slack   slack.NotifierSlackConfig     `json:"notifierSlackConfig"`
	Datadog datadog.NotifierDatadogConfig `json:"notifierDatadogConfig"`
}

type Notifier interface {
	GetNotifierName() string
	NotifyWarning(v1beta1.Disruption, corev1.Event) error
	NotifyRecovery(v1beta1.Disruption, corev1.Event) error
}

// GetNotifier returns an initiated Notifier instance
func GetNotifiers(config NotifiersConfig, logger *zap.SugaredLogger) (notifiers []Notifier, err error) {
	err = nil

	if config.Noop.Enabled {
		not := noop.New()
		notifiers = append(notifiers, not)
	}

	if config.Slack.Enabled {
		not, slackErr := slack.New(config.Common, config.Slack, logger)
		if slackErr != nil {
			err = slackErr
		} else {
			notifiers = append(notifiers, not)
		}
	}

	if config.Datadog.Enabled {
		not, ddogErr := datadog.New(config.Common, config.Datadog, logger)
		if ddogErr != nil {
			err = ddogErr
		} else {
			notifiers = append(notifiers, not)
		}
	}

	return
}

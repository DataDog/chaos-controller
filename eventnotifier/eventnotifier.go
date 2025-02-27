// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package eventnotifier

import (
	"github.com/DataDog/chaos-controller/eventnotifier/datadog"
	"github.com/DataDog/chaos-controller/eventnotifier/http"
	"github.com/DataDog/chaos-controller/eventnotifier/noop"
	"github.com/DataDog/chaos-controller/eventnotifier/slack"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotifiersConfig struct {
	Common  types.NotifiersCommonConfig   `json:"common" yaml:"common"`
	Noop    noop.NotifierNoopConfig       `json:"noop" yaml:"noop"`
	Slack   slack.NotifierSlackConfig     `json:"slack" yaml:"slack"`
	Datadog datadog.NotifierDatadogConfig `json:"datadog" yaml:"datadog"`
	HTTP    http.Config                   `json:"http" yaml:"http"`
}

type Notifier interface {
	GetNotifierName() string
	Notify(client.Object, corev1.Event, types.NotificationType) error
}

// CreateNotifiers creates and returns a list of Notifier instances
func CreateNotifiers(config NotifiersConfig, logger *zap.SugaredLogger) (notifiers []Notifier, err error) {
	err = nil

	if config.Noop.Enabled {
		not := noop.New(logger)
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
		not, ddogErr := datadog.New(config.Common, config.Datadog, logger, nil)
		if ddogErr != nil {
			err = ddogErr
		} else {
			notifiers = append(notifiers, not)
		}
	}

	if config.HTTP.IsEnabled() {
		not, httpErr := http.New(config.Common, config.HTTP, logger)
		if httpErr != nil {
			err = httpErr
		} else {
			notifiers = append(notifiers, not)
		}
	}

	return notifiers, err
}

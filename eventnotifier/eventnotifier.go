// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventnotifier

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/noop"
	"github.com/DataDog/chaos-controller/eventnotifier/slack"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	corev1 "k8s.io/api/core/v1"
)

type Context struct {
	Cluster string
}

type Notifier interface {
	GetNotifierName() string
	Clean() error

	NotifyWarning(v1beta1.Disruption, corev1.Event) error

	// NotifyNoTarget(v1beta1.Disruption) error
	// NotifyStuckOnRemoval(v1beta1.Disruption) error
}

// GetNotifier returns an initiated Notifier instance
func GetNotifier(driver types.NotifierDriver, filePath string) (notifier Notifier, err error) {
	switch driver {
	case types.NotifierDriverNoop:
		notifier, err = noop.New(), nil
	case types.NotifierDriverSlack:
		notifier, err = slack.New(filePath)
	default:
		notifier, err = nil, fmt.Errorf("unsupported notifier driver: %s", driver)
	}

	return
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventnotifier

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/noop"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
)

type Context struct {
	Cluster string
}

type Notifier interface {
	GetNotifierName() string
	Clean() error

	NotifyNotInjected(v1beta1.Disruption) error
	NotifyNoTarget(v1beta1.Disruption) error
	NotifyStuckOnRemoval(v1beta1.Disruption) error
	NotifyInjected(v1beta1.Disruption) error
	NotifyCleanedUp(v1beta1.Disruption) error
}

// GetNotifier returns an initiated Notifier instance
func GetNotifier(driver types.NotifierDriver) (Notifier, error) {
	switch driver {
	case types.NotifierDriverNoop:
		return noop.New(), nil
	case types.NotifierDriverSlack:
		return nil, fmt.Errorf("NotifierDriverSlack not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported notifier driver: %s", driver)
	}
}

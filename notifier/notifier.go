// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package notifier

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/notifier/types"
)

type Context struct {
	Cluster string
}

type Notifier interface {
	GetNotifierName() string
	Clean() error

	NotifyInjected(v1beta1.Disruption) error
	NotifyInvalidated(v1beta1.Disruption) error
	NotifyNoTarget(v1beta1.Disruption) error
	NotifyCleaned(v1beta1.Disruption) error
	NotifyNotCleaned(v1beta1.Disruption) error

	// IDEAS:
	// CleanedAfterTimeout(v1beta1.Disruption) error
}

// GetNotifier returns an initiated Notifier instance
func GetNotifier(driver types.NotifierDriver) (Notifier, error) {
	switch driver {
	case types.NotifierDriverSlack:
		return nil, fmt.Errorf("NotifierDriverSlack not implemented yet")
	case types.NotifierDriverNoop:
		return nil, fmt.Errorf("NotifierDriverNoop not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported notifier driver: %s", driver)
	}
}

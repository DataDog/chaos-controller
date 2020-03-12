// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package metrics

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/metrics/datadog"
	"github.com/DataDog/chaos-controller/metrics/noop"
	"github.com/DataDog/chaos-controller/types"
)

// Sink describes a metric sink
type Sink interface {
	EventCleanFailure(containerID, uid string)
	EventInjectFailure(containerID, uid string)
	EventWithTags(title, text string, tags []string)
	GetSinkName() string
	MetricCleaned(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string)
	MetricCleanupDuration(duration time.Duration, tags []string)
	MetricInjectDuration(duration time.Duration, tags []string)
	MetricInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string)
	MetricIPTablesRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string)
	MetricPodsCreated(targetPod, instanceName, namespace, phase string, succeed bool)
	MetricReconcile()
	MetricReconcileDuration(duration time.Duration, tags []string)
	MetricRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string)
}

// GetSink returns an initiated sink
func GetSink(name string) (Sink, error) {
	switch name {
	case "datadog":
		return datadog.New(), nil
	case "noop":
		return noop.New(), nil
	default:
		return nil, fmt.Errorf("unsupported metrics sink: %s", name)
	}
}

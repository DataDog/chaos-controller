// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package metrics

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/o11y/metrics/datadog"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// Sink describes a metric sink
type Sink interface {
	Close() error
	GetSinkName() string
	GetPrefix() string
	MetricCleaned(succeed bool, kind string, tags []string) error
	MetricCleanedForReinjection(succeed bool, kind string, tags []string) error
	MetricCleanupDuration(duration time.Duration, tags []string) error
	MetricInjectDuration(duration time.Duration, tags []string) error
	MetricInjected(succeed bool, kind string, tags []string) error
	MetricReinjected(succeed bool, kind string, tags []string) error
	MetricPodsCreated(target, instanceName, namespace string, succeed bool) error
	MetricReconcile() error
	MetricReconcileDuration(duration time.Duration, tags []string) error
	MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error
	MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error
	MetricStuckOnRemoval(tags []string) error
	MetricStuckOnRemovalGauge(gauge float64) error
	MetricDisruptionsGauge(gauge float64, tags []string) error
	MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error
	MetricSelectorCacheGauge(gauge float64) error
	MetricWatcherCalls(tags []string) error
	MetricPodsGauge(gauge float64) error
	MetricRestart() error
	MetricValidationFailed(tags []string) error
	MetricValidationCreated(tags []string) error
	MetricValidationUpdated(tags []string) error
	MetricValidationDeleted(tags []string) error
	MetricInformed(tags []string) error
	MetricOrphanFound(tags []string) error
	MetricTooLate(tags []string) error
	MetricTargetMissing(duration time.Duration, tags []string) error
	MetricMissingTargetFound(tags []string) error
	MetricMissingTargetDeleted(tags []string) error
	MetricNextScheduledTime(time time.Duration, tags []string) error
	MetricDisruptionScheduled(tags []string) error
	MetricPausedCron(tags []string) error
}

// GetSink returns an initiated sink
func GetSink(log *zap.SugaredLogger, driver types.SinkDriver, app types.SinkApp) (Sink, error) {
	switch driver {
	case types.SinkDriverDatadog:
		return datadog.New(app)
	case types.SinkDriverNoop:
		return noop.New(log), nil
	default:
		return nil, fmt.Errorf("unsupported metrics sink: %s", driver)
	}
}

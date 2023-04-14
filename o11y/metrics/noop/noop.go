// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// Sink describes a no-op sink
type Sink struct {
	log *zap.SugaredLogger
}

// New ...
func New(log *zap.SugaredLogger) Sink {
	return Sink{
		log,
	}
}

// Close returns nil
func (n Sink) Close() error {
	return nil
}

// GetSinkName returns the name of the sink
func (n Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

// MetricInjected increments the injected metric
func (n Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricInjected %v\n", succeed)

	return nil
}

// MetricReinjected increments the reinjected metric
func (n Sink) MetricReinjected(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricReinjected %v\n", succeed)

	return nil
}

// MetricCleaned increments the cleaned metric
func (n Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricCleaned %v\n", succeed)

	return nil
}

// MetricCleanedForReinjection increments the cleanedForReinjection metric
func (n Sink) MetricCleanedForReinjection(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricCleanedForReinjection %v\n", succeed)

	return nil
}

// MetricCleanupDuration send timing metric for cleanup duration
func (n Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricCleanupDuration %v\n", duration)

	return nil
}

// MetricInjectDuration send timing metric for inject duration
func (n Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricInjectDuration %v\n", duration)

	return nil
}

// MetricDisruptionCompletedDuration sends timing metric for entire disruption duration
func (n Sink) MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionCompletedDuration %v\n", duration)

	return nil
}

// MetricDisruptionOngoingDuration sends timing metric for disruption duration so far
func (n Sink) MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionOngoingDuration %v %s\n", duration, tags)

	return nil
}

// MetricReconcile increment reconcile metric
func (n Sink) MetricReconcile() error {
	fmt.Println("NOOP: MetricReconcile +1")

	return nil
}

// MetricReconcileDuration send timing metric for reconcile loop
func (n Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricReconcileDuration %v\n", duration)

	return nil
}

// MetricPodsCreated increment pods.created metric
func (n Sink) MetricPodsCreated(target, instanceName, namespace string, succeed bool) error {
	fmt.Println("NOOP: MetricPodsCreated +1")

	return nil
}

// MetricStuckOnRemoval increments disruptions.stuck_on_removal metric
func (n Sink) MetricStuckOnRemoval(tags []string) error {
	fmt.Println("NOOP: MetricStuckOnRemoval +1")

	return nil
}

// MetricStuckOnRemovalGauge sends disruptions.stuck_on_removal_total metric
func (n Sink) MetricStuckOnRemovalGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricStuckOnRemovalGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsGauge sends disruptions.gauge metric
func (n Sink) MetricDisruptionsGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricDisruptionsGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
func (n Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionsCount %s %s\n", kind, tags)

	return nil
}

// MetricPodsGauge sends pods.gauge metric
func (n Sink) MetricPodsGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricPodsGauge %f\n", gauge)

	return nil
}

// MetricRestart sends restart metric
func (n Sink) MetricRestart() error {
	fmt.Println("NOOP: MetricRestart")

	return nil
}

func (n Sink) MetricValidationFailed(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationFailed %s\n", tags)

	return nil
}

func (n Sink) MetricValidationCreated(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationCreated %s\n", tags)

	return nil
}

func (n Sink) MetricValidationUpdated(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationUpdated %s\n", tags)

	return nil
}

func (n Sink) MetricValidationDeleted(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationDeleted %s\n", tags)

	return nil
}

func (n Sink) MetricInformed(tags []string) error {
	n.log.Debugf("NOOP: MetricInformed %s\n", tags)

	return nil
}

func (n Sink) MetricOrphanFound(tags []string) error {
	n.log.Debugf("NOOP: MetricOrphanFound %s\n", tags)

	return nil
}

// MetricCacheTriggered signals a selector cache trigger
func (n Sink) MetricSelectorCacheTriggered(tags []string) error {
	n.log.Debugf("NOOP: MetricCacheTriggered %s\n", tags)

	return nil
}

// MetricSelectorCacheGauge reports how many caches are still in the cache array to prevent leaks
func (n Sink) MetricSelectorCacheGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricSelectorCacheGauge %f\n", gauge)

	return nil
}

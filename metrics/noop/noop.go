// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// Sink describes a no-op sink
type Sink struct {
}

// New ...
func New() *Sink {
	return &Sink{}
}

// Close returns nil
func (n *Sink) Close() error {
	return nil
}

// GetSinkName returns the name of the sink
func (n *Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

// MetricInjected increments the injected metric
func (n *Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricInjected %v\n", succeed)

	return nil
}

// MetricReinjected increments the reinjected metric
func (n *Sink) MetricReinjected(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricReinjected %v\n", succeed)

	return nil
}

// MetricCleaned increments the cleaned metric
func (n *Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricCleaned %v\n", succeed)

	return nil
}

// MetricCleanedForReinjection increments the cleanedForReinjection metric
func (d *Sink) MetricCleanedForReinjection(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricCleanedForReinjection %v\n", succeed)

	return nil
}

// MetricCleanupDuration send timing metric for cleanup duration
func (n *Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	fmt.Printf("NOOP: MetricCleanupDuration %v\n", duration)

	return nil
}

// MetricInjectDuration send timing metric for inject duration
func (n *Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	fmt.Printf("NOOP: MetricInjectDuration %v\n", duration)

	return nil
}

// MetricDisruptionCompletedDuration sends timing metric for entire disruption duration
func (n *Sink) MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error {
	fmt.Printf("NOOP: MetricDisruptionCompletedDuration %v\n", duration)

	return nil
}

// MetricDisruptionOngoingDuration sends timing metric for disruption duration so far
func (n *Sink) MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error {
	fmt.Printf("NOOP: MetricDisruptionOngoingDuration %v %s\n", duration, tags)

	return nil
}

// MetricReconcile increment reconcile metric
func (n *Sink) MetricReconcile() error {
	fmt.Println("NOOP: MetricReconcile +1")

	return nil
}

// MetricReconcileDuration send timing metric for reconcile loop
func (n *Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	fmt.Printf("NOOP: MetricReconcileDuration %v\n", duration)

	return nil
}

// MetricPodsCreated increment pods.created metric
func (n *Sink) MetricPodsCreated(target, instanceName, namespace string, succeed bool) error {
	fmt.Println("NOOP: MetricPodsCreated +1")

	return nil
}

// MetricStuckOnRemoval increments disruptions.stuck_on_removal metric
func (n *Sink) MetricStuckOnRemoval(tags []string) error {
	fmt.Println("NOOP: MetricStuckOnRemoval +1")

	return nil
}

// MetricStuckOnRemovalGauge sends disruptions.stuck_on_removal_total metric
func (n *Sink) MetricStuckOnRemovalGauge(gauge float64) error {
	fmt.Printf("NOOP: MetricStuckOnRemovalGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsGauge sends disruptions.gauge metric
func (n *Sink) MetricDisruptionsGauge(gauge float64) error {
	fmt.Printf("NOOP: MetricDisruptionsGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
func (n *Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error {
	fmt.Printf("NOOP: MetricDisruptionsCount %s %s\n", kind, tags)

	return nil
}

// MetricPodsGauge sends pods.gauge metric
func (n *Sink) MetricPodsGauge(gauge float64) error {
	fmt.Printf("NOOP: MetricPodsGauge %f\n", gauge)

	return nil
}

// MetricRestart sends restart metric
func (n *Sink) MetricRestart() error {
	fmt.Println("NOOP: MetricRestart")

	return nil
}

func (n *Sink) MetricValidationFailed(tags []string) error {
	fmt.Printf("NOOP: MetricValidationFailed %s\n", tags)

	return nil
}

func (n *Sink) MetricValidationCreated(tags []string) error {
	fmt.Printf("NOOP: MetricValidationCreated %s\n", tags)

	return nil
}

func (n *Sink) MetricValidationUpdated(tags []string) error {
	fmt.Printf("NOOP: MetricValidationUpdated %s\n", tags)

	return nil
}

func (n *Sink) MetricValidationDeleted(tags []string) error {
	fmt.Printf("NOOP: MetricValidationDeleted %s\n", tags)

	return nil
}

func (n *Sink) MetricInformed(tags []string) error {
	fmt.Printf("NOOP: MetricInformed %s\n", tags)

	return nil
}

func (n *Sink) MetricOrphanFound(tags []string) error {
	fmt.Printf("NOOP: MetricOrphanFound %s\n", tags)

	return nil
}

// MetricCacheTriggered signals a selector cache trigger
func (n *Sink) MetricSelectorCacheTriggered(tags []string) error {
	fmt.Printf("NOOP: MetricCacheTriggered %s\n", tags)

	return nil
}

// MetricSelectorCacheGauge reports how many caches are still in the cache array to prevent leaks
func (n *Sink) MetricSelectorCacheGauge(gauge float64) error {
	fmt.Printf("NOOP: MetricSelectorCacheGauge %f\n", gauge)

	return nil
}

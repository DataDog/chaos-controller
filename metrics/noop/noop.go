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

// Flush returns nil
func (n *Sink) Flush() error {
	return nil
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (n *Sink) EventWithTags(title, text string, tags []string) error {
	fmt.Printf("NOOP: Event %s\n", title)

	return nil
}

// MetricInjected increments the injected metric
func (n *Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricInjected %v\n", succeed)

	return nil
}

// MetricCleaned increments the cleaned metric
func (n *Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	fmt.Printf("NOOP: MetricCleaned %v\n", succeed)

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
func (n *Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKind, tags []string) error {
	fmt.Printf("NOOP: MetricDisruptionsCount %s %s\n", kind, tags)

	return nil
}

// MetricPodsGauge sends pods.gauge metric
func (n *Sink) MetricPodsGauge(gauge float64) error {
	fmt.Printf("NOOP: MetricPodsGauge %f\n", gauge)

	return nil
}

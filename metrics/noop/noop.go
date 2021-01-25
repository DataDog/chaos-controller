// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package noop

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/metrics/types"
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
	return nil
}

// MetricInjected increments the injected metric
func (n *Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	return nil
}

// MetricCleaned increments the cleaned metric
func (n *Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	return nil
}

// MetricCleanupDuration send timing metric for cleanup duration
func (n *Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	fmt.Println("NOOP: MetricCleanupDuration +1")

	return nil
}

// MetricInjectDuration send timing metric for inject duration
func (n *Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	fmt.Println("NOOP: MetricInjectDuration +1")

	return nil
}

// MetricReconcile increment reconcile metric
func (n *Sink) MetricReconcile() error {
	fmt.Println("NOOP: MetricReconcile +1")

	return nil
}

// MetricReconcileDuration send timing metric for reconcile loop
func (n *Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	fmt.Println("NOOP: MetricReconcileDuration +1")

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

// MetricStuckOnRemovalCount sends disruptions.stuck_on_removal_count metric
func (n *Sink) MetricStuckOnRemovalCount(count float64) error {
	fmt.Println("NOOP: MetricStuckOnRemovalCount +1")

	return nil
}

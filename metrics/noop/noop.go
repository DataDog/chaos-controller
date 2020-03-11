// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package noop

import (
	"github.com/DataDog/chaos-controller/types"
)

// Sink describes a no-op sink
type Sink struct{}

// New ...
func New() *Sink {
	return &Sink{}
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (n *Sink) EventWithTags(title, text string, tags []string) {}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func (n *Sink) EventCleanFailure(containerID, uid string) {}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func (n *Sink) EventInjectFailure(containerID, uid string) {}

// MetricInjected increments the injected metric
func (n *Sink) MetricInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
}

// MetricRulesInjected rules.increments the injected metric
func (n *Sink) MetricRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
}

// MetricCleaned increments the cleaned metric
func (n *Sink) MetricCleaned(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
}

// MetricIPTablesRulesInjected increment iptables_rules metrics
func (n *Sink) MetricIPTablesRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
}

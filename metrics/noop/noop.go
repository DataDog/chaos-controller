// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package noop

import (
	"log"

	"github.com/DataDog/chaos-controller/types"
)

// Sink describes a no-op sink
type Sink struct{}

// New ...
func New() *Sink {
	return &Sink{}
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (n *Sink) EventWithTags(title, text string, tags []string) {
	log.Printf("%v %v %v", title, text, tags)
}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func (n *Sink) EventCleanFailure(containerID, uid string) {
	log.Printf("EventCleanFailure %v", containerID)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func (n *Sink) EventInjectFailure(containerID, uid string) {
	log.Printf("EventInjectFailure %v", containerID)
}

// MetricInjected increments the injected metric
func (n *Sink) MetricInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	log.Printf("MetricInjected %v", containerID)
}

// MetricRulesInjected rules.increments the injected metric
func (n *Sink) MetricRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	log.Printf("MetricRulesInjected %v", containerID)
}

// MetricCleaned increments the cleaned metric
func (n *Sink) MetricCleaned(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	log.Printf("MetricCleaned %v", containerID)
}

// MetricIPTablesRulesInjected increment iptables_rules metrics
func (n *Sink) MetricIPTablesRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	log.Printf("MetricIPTablesRulesInjected %v", containerID)
}

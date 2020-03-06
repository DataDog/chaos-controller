// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package datadog

import (
	"log"
	"os"

	"github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/datadog-go/statsd"
)

const metricPrefix = "chaos.injector."

// Sink describes a Datadog sink (statsd)
type Sink struct {
	*statsd.Client
}

// New instantiate a new datadog statsd provider
func New() *Sink {
	url := os.Getenv("STATSD_URL")
	instance, err := statsd.New(url, statsd.WithTags([]string{"app:chaos-controller"}))

	if err != nil {
		log.Fatal(err)
	}

	return &Sink{Client: instance}
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (d *Sink) EventWithTags(title, text string, tags []string) {
	e := &statsd.Event{
		Title: title,
		Text:  text,
		Tags:  tags,
	}
	err := d.Event(e)

	if err != nil {
		log.Printf("error sending an event to datadog: %v", err)
	}
}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func (d *Sink) EventCleanFailure(containerID, uid string) {
	d.EventWithTags("network failure clean failed", "please check the cleanup pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func (d *Sink) EventInjectFailure(containerID, uid string) {
	d.EventWithTags("network failure injection failed", "please check the inject pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

func (d *Sink) metricWithStatus(name, containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	var status string
	if succeed {
		status = "succeed"
	} else {
		status = "failed"
	}

	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status, "kind:" + string(kind)}
	t = append(t, tags...)

	_ = d.Incr(name, t, 1)
}

// MetricInjected increments the injected metric
func (d *Sink) MetricInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	d.metricWithStatus(metricPrefix+"injected", containerID, uid, succeed, kind, tags)
}

// MetricRulesInjected rules.increments the injected metric
func (d *Sink) MetricRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	d.metricWithStatus(metricPrefix+"rules.injected", containerID, uid, succeed, kind, tags)
}

// MetricCleaned increments the cleaned metric
func (d *Sink) MetricCleaned(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	d.metricWithStatus(metricPrefix+"cleaned", containerID, uid, succeed, kind, tags)
}

// MetricIPTablesRulesInjected increment iptables_rules metrics
func (d *Sink) MetricIPTablesRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	d.metricWithStatus(metricPrefix+"iptables_rules.injected", containerID, uid, succeed, kind, tags)
}

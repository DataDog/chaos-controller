// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package datadog

import (
	"log"
	"os"

	"github.com/DataDog/datadog-go/statsd"
)

const metricPrefix = "chaos.injector."

// DatadogProvider ...
type DatadogSink struct {
	*statsd.Client
}

// New instantiate a new datadog statsd provider
func New() *DatadogSink {
	url := os.Getenv("STATSD_URL")
	instance, err := statsd.New(url, statsd.WithTags([]string{"app:chaos-controller"}))
	if err != nil {
		log.Fatal(err)
	}

	return &DatadogSink{Client: instance}
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (d *DatadogSink) EventWithTags(title, text string, tags []string) {
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
func (d *DatadogSink) EventCleanFailure(containerID, uid string) {
	d.EventWithTags("network failure clean failed", "please check the cleanup pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func (d *DatadogSink) EventInjectFailure(containerID, uid string) {
	d.EventWithTags("network failure injection failed", "please check the inject pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

func (d *DatadogSink) metricWithStatus(name, containerID, uid string, succeed bool, tags []string) {
	var status string
	if succeed {
		status = "succeed"
	} else {
		status = "failed"
	}
	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status}
	t = append(t, tags...)

	d.Incr(name, t, 1)
}

// MetricInjected increments the injected metric
func (d *DatadogSink) MetricInjected(containerID, uid string, succeed bool, tags []string) {
	d.metricWithStatus(metricPrefix+"injected", containerID, uid, succeed, tags)
}

// MetricRulesInjected rules.increments the injected metric
func (d *DatadogSink) MetricRulesInjected(containerID, uid string, succeed bool, tags []string) {
	d.metricWithStatus(metricPrefix+"rules.injected", containerID, uid, succeed, tags)
}

// MetricCleaned increments the cleaned metric
func (d *DatadogSink) MetricCleaned(containerID, uid string, succeed bool, tags []string) {
	d.metricWithStatus(metricPrefix+"cleaned", containerID, uid, succeed, tags)
}

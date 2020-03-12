// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package datadog

import (
	"log"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/datadog-go/statsd"
)

const (
	metricPrefixInjector   = "chaos.injector."
	metricPrefixController = "chaos.controller."
)

// Sink describes a Datadog sink (statsd)
type Sink struct {
	client   *statsd.Client
	sinkName string
}

// New instantiate a new datadog statsd provider
func New() *Sink {
	url := os.Getenv("STATSD_URL")
	instance, err := statsd.New(url, statsd.WithTags([]string{"app:chaos-controller"}))

	if err != nil {
		log.Fatal(err)
	}

	return &Sink{client: instance, sinkName: "datadog"}
}

// GetSinkName returns the name of the sink
func (d *Sink) GetSinkName() string {
	return d.sinkName
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (d *Sink) EventWithTags(title, text string, tags []string) {
	e := &statsd.Event{
		Title: title,
		Text:  text,
		Tags:  tags,
	}
	err := d.client.Event(e)

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

func boolToStatus(succeed bool) string {
	var status string
	if succeed {
		status = "succeed"
	} else {
		status = "failed"
	}

	return status
}

func (d *Sink) metricWithStatus(name string, tags []string) {
	_ = d.client.Incr(name, tags, 1)
}

func (d *Sink) timing(name string, duration time.Duration, tags []string) {
	_ = d.client.Timing(name, duration, tags, 1)
}

// MetricInjected increments the injected metric
func (d *Sink) MetricInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	status := boolToStatus(succeed)
	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status, "kind:" + string(kind)}
	t = append(t, tags...)

	d.metricWithStatus(metricPrefixInjector+"injected", t)
}

// MetricRulesInjected rules.increments the injected metric
func (d *Sink) MetricRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	status := boolToStatus(succeed)
	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status, "kind:" + string(kind)}
	t = append(t, tags...)

	d.metricWithStatus(metricPrefixInjector+"rules.injected", t)
}

// MetricCleaned increments the cleaned metric
func (d *Sink) MetricCleaned(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	status := boolToStatus(succeed)
	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status, "kind:" + string(kind)}
	t = append(t, tags...)

	d.metricWithStatus(metricPrefixInjector+"cleaned", t)
}

// MetricIPTablesRulesInjected increment iptables_rules metrics
func (d *Sink) MetricIPTablesRulesInjected(containerID, uid string, succeed bool, kind types.DisruptionKind, tags []string) {
	status := boolToStatus(succeed)
	t := []string{"containerID:" + containerID, "UID:" + uid, "status:" + status, "kind:" + string(kind)}
	t = append(t, tags...)

	d.metricWithStatus(metricPrefixInjector+"iptables_rules.injected", t)
}

// MetricReconcile increment reconcile metric
func (d *Sink) MetricReconcile() {
	d.metricWithStatus(metricPrefixController+"reconcile", []string{})
}

// MetricReconcileDuration send timing metric for reconcile loop
func (d *Sink) MetricReconcileDuration(duration time.Duration, tags []string) {
	d.timing(metricPrefixController+"reconcile.duration", duration, tags)
}

// MetricCleanupDuration send timing metric for cleanup duration
func (d *Sink) MetricCleanupDuration(duration time.Duration, tags []string) {
	d.timing(metricPrefixController+"cleanup.duration", duration, tags)
}

// MetricInjectDuration send timing metric for inject duration
func (d *Sink) MetricInjectDuration(duration time.Duration, tags []string) {
	d.timing(metricPrefixController+"inject.duration", duration, tags)
}

// MetricPodsCreated increment pods.created metric
func (d *Sink) MetricPodsCreated(targetPod, instanceName, namespace, phase string, succeed bool) {
	status := boolToStatus(succeed)
	tags := []string{"phase:" + phase, "target_pod:" + targetPod, "name:" + instanceName, "status:" + status, "namespace:" + namespace}

	d.metricWithStatus(metricPrefixController+"pods.created", tags)
}

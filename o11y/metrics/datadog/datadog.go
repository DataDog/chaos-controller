// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package datadog

import (
	"fmt"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/datadog-go/statsd"
)

const (
	metricPrefixInjector          = "chaos.injector."
	metricPrefixController        = "chaos.controller."
	metricPrefixRolloutController = "chaos.rollout.controller."
	metricPrefixCronController    = "chaos.cron.controller."
)

// Sink describes a Datadog sink (statsd)
type Sink struct {
	client *statsd.Client
	prefix string
}

// New instantiate a new datadog statsd provider
func New(app types.SinkApp) (Sink, error) {
	url := os.Getenv("STATSD_URL")

	instance, err := statsd.New(url, statsd.WithTags([]string{"app:" + string(app)}))
	if err != nil {
		return Sink{}, err
	}

	prefix, err := GetPrefixFromApp(app)
	if err != nil {
		return Sink{}, err
	}

	return Sink{
		client: instance,
		prefix: prefix,
	}, nil
}

// GetPrefixFromApp returns the datadog metrics prefix given the App
func GetPrefixFromApp(app types.SinkApp) (string, error) {
	switch app {
	case types.SinkAppController:
		return metricPrefixController, nil
	case types.SinkAppRolloutController:
		return metricPrefixRolloutController, nil
	case types.SinkAppCronController:
		return metricPrefixCronController, nil
	case types.SinkAppInjector:
		return metricPrefixInjector, nil
	default:
		return "", fmt.Errorf("unknown sink app")
	}
}

// Close closes the statsd client
func (d Sink) Close() error {
	return d.client.Close()
}

// GetSinkName returns the name of the sink
func (d Sink) GetSinkName() string {
	return string(types.SinkDriverDatadog)
}

// MetricInjected increments the injected metric
func (d Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"injected", t)
}

// MetricReinjected increments the reinjected metric
func (d Sink) MetricReinjected(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"reinjected", t)
}

// MetricCleanedForReinjection increments the cleanedForReinjection metric
func (d Sink) MetricCleanedForReinjection(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"cleaned_for_reinjection", t)
}

// MetricCleaned increments the cleaned metric
func (d Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"cleaned", t)
}

// MetricReconcile increment reconcile metric
func (d Sink) MetricReconcile(tags []string) error {
	return d.metricWithStatus(d.prefix+"reconcile", tags)
}

// MetricReconcileDuration send timing metric for reconcile loop
func (d Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"reconcile.duration", duration, tags)
}

// MetricCleanupDuration send timing metric for cleanup duration
func (d Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"cleanup.duration", duration, tags)
}

// MetricInjectDuration send timing metric for inject duration
func (d Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"inject.duration", duration, tags)
}

// MetricDisruptionCompletedDuration sends timing metric for entire disruption duration
func (d Sink) MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"disruption.completed_duration", duration, tags)
}

// MetricDisruptionOngoingDuration sends timing metric for disruption duration so far
func (d Sink) MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"disruption.ongoing_duration", duration, tags)
}

// MetricPodsCreated increment pods.created metric
func (d Sink) MetricPodsCreated(target, instanceName, namespace string, succeed bool) error {
	status := boolToStatus(succeed)
	tags := []string{"target:" + target, "disruptionName:" + instanceName, "status:" + status, "namespace:" + namespace}

	return d.metricWithStatus(d.prefix+"pods.created", tags)
}

// MetricStuckOnRemoval increments disruptions.stuck_on_removal metric
func (d Sink) MetricStuckOnRemoval(tags []string) error {
	return d.metricWithStatus(d.prefix+"disruptions.stuck_on_removal", tags)
}

// MetricStuckOnRemovalGauge sends disruptions.stuck_on_removal_total metric containing the gauge of stuck disruptions
func (d Sink) MetricStuckOnRemovalGauge(gauge float64) error {
	return d.client.Gauge(d.prefix+"disruptions.stuck_on_removal_total", gauge, []string{}, 1)
}

// MetricDisruptionsGauge sends the disruptions.gauge metric counting ongoing disruptions
func (d Sink) MetricDisruptionsGauge(gauge float64) error {
	return d.client.Gauge(d.prefix+"disruptions.gauge", gauge, []string{}, 1)
}

// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
func (d Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error {
	tags = append(tags, fmt.Sprintf("disruption_kind:%s", kind))
	return d.metricWithStatus(d.prefix+"disruptions.count", tags)
}

// MetricPodsGauge sends the pods.gauge metric counting existing chaos pods
func (d Sink) MetricPodsGauge(gauge float64) error {
	return d.client.Gauge(d.prefix+"pods.gauge", gauge, []string{}, 1)
}

// MetricRestart sends an increment of the controller restart metric
func (d Sink) MetricRestart() error {
	return d.metricWithStatus(d.prefix+"restart", []string{})
}

// MetricValidationFailed increments the failed validation metric
func (d Sink) MetricValidationFailed(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.failed", tags)
}

// MetricValidationCreated increments the created validation metric
func (d Sink) MetricValidationCreated(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.created", tags)
}

// MetricValidationUpdated increments the updated validation metric
func (d Sink) MetricValidationUpdated(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.updated", tags)
}

// MetricValidationDeleted increments the deleted validation metric
func (d Sink) MetricValidationDeleted(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.deleted", tags)
}

// MetricInformed increments when the pod informer receives an event to process before reconciliation
func (d Sink) MetricInformed(tags []string) error {
	return d.metricWithStatus(d.prefix+"informed", tags)
}

// MetricOrphanFound increments when a chaos pod without a corresponding disruption resource is found
func (d Sink) MetricOrphanFound(tags []string) error {
	return d.metricWithStatus(d.prefix+"orphan.found", tags)
}

// MetricWatcherCalls is a counter of watcher calls.
func (d Sink) MetricWatcherCalls(tags []string) error {
	return d.metricWithStatus(d.prefix+"watcher.calls_total", tags)
}

// MetricSelectorCacheGauge reports how many caches are still in the cache array to prevent leaks
func (d Sink) MetricSelectorCacheGauge(gauge float64) error {
	return d.client.Gauge(d.prefix+"selector.cache.gauge", gauge, []string{}, 1)
}

// MetricTooLate reports when a scheduled disruption misses its aloted time to be scheduled
// specific to cron and rollout controllers
func (d Sink) MetricTooLate(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.too_late", tags)
}

// MetricTargetMissing reports when a scheduled Disruption can not find its specific target
// either for the first time or multiple times. A deletion occurs on the final alert
func (d Sink) MetricTargetMissing(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"schedule.target_missing", duration, tags)
}

// MetricMissingTargetFound reports when a scheduled Disruption which had initially been deemed missing
// is "found" and running in the kubernetes namespace
func (d Sink) MetricMissingTargetFound(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.missing_target_found", tags)
}

// MetricNextScheduledTime reports the duration until the next scheduled disruption will run
func (d Sink) MetricNextScheduledTime(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"schedule.next_scheduled", duration, tags)
}

// MetricDisruptionScheduled reports when a new disruption is scheduled
func (d Sink) MetricDisruptionScheduled(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.disruption_scheduled", tags)
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

func (d Sink) metricWithStatus(name string, tags []string) error {
	return d.client.Incr(name, tags, 1)
}

func (d Sink) timing(name string, duration time.Duration, tags []string) error {
	return d.client.Timing(name, duration, tags, 1)
}

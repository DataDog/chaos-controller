// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package datadog

import (
	"fmt"
	"os"
	"time"

	"github.com/DataDog/datadog-go/statsd"

	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	metricPrefixInjector       = "chaos.injector."
	metricPrefixController     = "chaos.controller."
	metricPrefixCronController = "chaos.cron.controller."
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

	prefixFromApp, err := GetPrefixFromApp(app)
	if err != nil {
		return Sink{}, err
	}

	return Sink{
		client: instance,
		prefix: prefixFromApp,
	}, nil
}

// GetPrefixFromApp returns the datadog metrics prefix given the App
func GetPrefixFromApp(app types.SinkApp) (string, error) {
	switch app {
	case types.SinkAppController:
		return metricPrefixController, nil
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

// GetPrefix returns the prefix used when sending metrics to datadog
func (d Sink) GetPrefix() string {
	return d.prefix
}

// MetricInjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
// the `succeed` bool argument is false if there was an error while injecting.
func (d Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"injected", t)
}

// MetricReinjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
// the `succeed` bool argument is false if there was an error while injecting. This metric is used instead of MetricInjected
// if the chaos-injector pod is performing any injection after its first, i.e., when using the pulse feature
func (d Sink) MetricReinjected(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"reinjected", t)
}

// MetricCleanedForReinjection is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
// but expects to reinject, i.e., when using the spec.pulse feature
func (d Sink) MetricCleanedForReinjection(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"cleaned_for_reinjection", t)
}

// MetricCleaned is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
// and does not intend to re-inject.
func (d Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	status := boolToStatus(succeed)
	t := []string{"status:" + status, "kind:" + kind}
	t = append(t, tags...)

	return d.metricWithStatus(d.prefix+"cleaned", t)
}

// MetricReconcile is used to count how many times the controller enters any reconcile loop
func (d Sink) MetricReconcile() error {
	return d.metricWithStatus(d.prefix+"reconcile", []string{})
}

// MetricReconcileDuration is used at the end of every reconcile loop to indicate the duration that Reconcile() call spent
func (d Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"reconcile.duration", duration, tags)
}

// MetricCleanupDuration indicates the duration between a Disruption's deletion timestamp, and when the chaos-controller
// removes its finalizer
func (d Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"cleanup.duration", duration, tags)
}

// MetricInjectDuration indicates the duration between a Disruption's creation timestamp, and when it reaches a status
// of Injected, indicating all chaos-injector pods have injected into their targets, and we've reached the expected count
func (d Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"inject.duration", duration, tags)
}

// MetricDisruptionCompletedDuration indicates the duration between a Disruption's creation timestamp, and when the chaos-controller
// removes its finalizer
func (d Sink) MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"disruption.completed_duration", duration, tags)
}

// MetricDisruptionOngoingDuration indicates the duration between a Disruption's creation timestamp, and the current time.
// This is emitted approximately every one minute
func (d Sink) MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"disruption.ongoing_duration", duration, tags)
}

// MetricPodsCreated is used every time the chaos-controller finishes sending a Create request to the k8s api to
// schedule a new chaos-injector pod. The `succeed` bool argument is false if there was an error returned.
func (d Sink) MetricPodsCreated(target, instanceName, namespace string, succeed bool) error {
	status := boolToStatus(succeed)
	tags := []string{"target:" + target, "disruptionName:" + instanceName, "status:" + status, "disruptionNamespace:" + namespace}

	return d.metricWithStatus(d.prefix+"pods.created", tags)
}

// MetricStuckOnRemovalCurrent is emitted once per minute counting the number of disruptions _per namespace_
// that are "stuck on removal", i.e.,
// we have attempted to clean and delete the disruption, but that has not worked, and a human needs to intervene.
func (d Sink) MetricStuckOnRemovalCurrent(gauge float64, tags []string) error {
	return d.client.Gauge(d.prefix+"disruptions.stuck_on_removal_current", gauge, tags, 1)
}

// MetricDisruptionsGauge is emitted once per minute counting the total number of ongoing disruptions per namespace,
// or if we fail to determine the namespaced metrics, simply the total number of disruptions found
func (d Sink) MetricDisruptionsGauge(gauge float64, tags []string) error {
	return d.client.Gauge(d.prefix+"disruptions.gauge", gauge, tags, 1)
}

// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
func (d Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error {
	tags = append(tags, fmt.Sprintf("disruption_kind:%s", kind))
	return d.metricWithStatus(d.prefix+"disruptions.count", tags)
}

// MetricPodsGauge is emitted once per minute counting the total number of live chaos pods for all ongoing disruptions
func (d Sink) MetricPodsGauge(gauge float64) error {
	return d.client.Gauge(d.prefix+"pods.gauge", gauge, []string{}, 1)
}

// MetricRestart is emitted once, every time the manager container of the chaos-controller starts up
func (d Sink) MetricRestart() error {
	return d.metricWithStatus(d.prefix+"restart", []string{})
}

// MetricValidationFailed is emitted in ValidateCreate and ValidateUpdate in the disruption_webhook, specifically and
// only when DisruptionSpec.Validate() returns an error, OR when trying to remove the finalizer from a disruption with
// chaos pods.
func (d Sink) MetricValidationFailed(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.failed", tags)
}

// MetricValidationCreated is emitted once per created Disruption, in the webhook after validation completes.
func (d Sink) MetricValidationCreated(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.created", tags)
}

// MetricValidationUpdated is emitted once per Disruption update, in the webhook after validation completes
func (d Sink) MetricValidationUpdated(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.updated", tags)
}

// MetricValidationDeleted is emitted once per Disruption delete, in the webhook
func (d Sink) MetricValidationDeleted(tags []string) error {
	return d.metricWithStatus(d.prefix+"validation.deleted", tags)
}

// MetricInformed is emitted every time the manager container's informer is called to check a pod in the chaos-controller's
// namespace, to see if that pod is a chaos-injector pod that needs its Disruption reconciled.
func (d Sink) MetricInformed(tags []string) error {
	return d.metricWithStatus(d.prefix+"informed", tags)
}

// MetricOrphanFound increments when a chaos pod without a corresponding disruption resource is found
func (d Sink) MetricOrphanFound(tags []string) error {
	return d.metricWithStatus(d.prefix+"orphan.found", tags)
}

// MetricWatcherCalls is a counter of watcher calls. This is emitted by every OnChange event for all of our watchers,
// e.g., the chaos pod watcher, the target pod watcher, the disruption watcher.
func (d Sink) MetricWatcherCalls(tags []string) error {
	return d.metricWithStatus(d.prefix+"watcher.calls_total", tags)
}

// MetricTooLate reports when a scheduled Disruption misses its configured time to be run,
// specific to cron controllers
func (d Sink) MetricTooLate(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.too_late", tags)
}

// MetricTargetMissing reports anytime scheduled Disruption can not find its specified target
func (d Sink) MetricTargetMissing(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"schedule.target_missing", duration, tags)
}

// MetricMissingTargetFound reports when a scheduled Disruption's target which had initially been deemed missing
// is "found" and running in the kubernetes namespace
func (d Sink) MetricMissingTargetFound(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.missing_target_found", tags)
}

// MetricMissingTargetDeleted reports when a scheduled Disruption has been deleted by the chaos-controller,
// because its target has been missing for too long
func (d Sink) MetricMissingTargetDeleted(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.missing_target_deleted", tags)
}

// MetricNextScheduledTime reports the duration until this scheduled Disruption's next scheduled disruption should run
func (d Sink) MetricNextScheduledTime(duration time.Duration, tags []string) error {
	return d.timing(d.prefix+"schedule.next_scheduled", duration, tags)
}

// MetricDisruptionScheduled reports when a new disruption is scheduled
func (d Sink) MetricDisruptionScheduled(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.disruption_scheduled", tags)
}

// MetricPausedCron reports when a disruption cron has reconciled in a paused state
func (d Sink) MetricPausedCron(tags []string) error {
	return d.metricWithStatus(d.prefix+"schedule.paused", tags)
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

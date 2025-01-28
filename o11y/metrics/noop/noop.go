// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package noop

import (
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// Sink describes a no-op sink
type Sink struct {
	log *zap.SugaredLogger
}

// New ...
func New(log *zap.SugaredLogger) Sink {
	return Sink{
		log,
	}
}

// Close returns nil
func (n Sink) Close() error {
	return nil
}

// GetSinkName returns the name of the sink
func (n Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

// GetPrefix returns the prefix used when sending metrics to datadog
func (n Sink) GetPrefix() string {
	return "noop"
}

// MetricInjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
// the `succeed` bool argument is false if there was an error while injecting.
func (n Sink) MetricInjected(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricInjected %v\n", succeed)

	return nil
}

// MetricReinjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
// the `succeed` bool argument is false if there was an error while injecting. This metric is used instead of MetricInjected
// if the chaos-injector pod is performing any injection after its first, i.e., when using the pulse feature
func (n Sink) MetricReinjected(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricReinjected %v\n", succeed)

	return nil
}

// MetricCleaned is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
// and does not intend to re-inject.
func (n Sink) MetricCleaned(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricCleaned %v\n", succeed)

	return nil
}

// MetricCleanedForReinjection is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
// but expects to reinject, i.e., when using the spec.pulse feature
func (n Sink) MetricCleanedForReinjection(succeed bool, kind string, tags []string) error {
	n.log.Debugf("NOOP: MetricCleanedForReinjection %v\n", succeed)

	return nil
}

// MetricCleanupDuration indicates the duration between a Disruption's deletion timestamp, and when the chaos-controller
// removes its finalizer
func (n Sink) MetricCleanupDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricCleanupDuration %v\n", duration)

	return nil
}

// MetricInjectDuration indicates the duration between a Disruption's creation timestamp, and when it reaches a status
// of Injected, indicating all chaos-injector pods have injected into their targets, and we've reached the expected count
func (n Sink) MetricInjectDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricInjectDuration %v\n", duration)

	return nil
}

// MetricDisruptionCompletedDuration indicates the duration between a Disruption's creation timestamp, and when the chaos-controller
// removes its finalizer
func (n Sink) MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionCompletedDuration %v\n", duration)

	return nil
}

// MetricDisruptionOngoingDuration indicates the duration between a Disruption's creation timestamp, and the current time.
// This is emitted approximately every one minute
func (n Sink) MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionOngoingDuration %v %s\n", duration, tags)

	return nil
}

// MetricReconcile is used to count how many times the controller enters any reconcile loop
func (n Sink) MetricReconcile() error {
	n.log.Debugf("NOOP: MetricReconcile +1")

	return nil
}

// MetricReconcileDuration is used at the end of every reconcile loop to indicate the duration that Reconcile() call spent
func (n Sink) MetricReconcileDuration(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricReconcileDuration %v\n", duration)

	return nil
}

// MetricPodsCreated is used every time the chaos-controller finishes sending a Create request to the k8s api to
// schedule a new chaos-injector pod. The `succeed` bool argument is false if there was an error returned.
func (n Sink) MetricPodsCreated(target, instanceName, namespace string, succeed bool) error {
	fmt.Println("NOOP: MetricPodsCreated +1")

	return nil
}

// MetricStuckOnRemoval is emitted once per minute per disruption, if that disruption is "stuck on removal", i.e.,
// we have attempted to clean and delete the disruption, but that has not worked, and a human needs to intervene.
func (n Sink) MetricStuckOnRemoval(tags []string) error {
	fmt.Println("NOOP: MetricStuckOnRemoval +1")

	return nil
}

// MetricStuckOnRemovalGauge is emitted once per minute counting the total number of disruptions that are
// "stuck on removal", i.e., we have attempted to clean and delete the disruption, but that has not worked,
// and a human needs to intervene.
func (n Sink) MetricStuckOnRemovalGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricStuckOnRemovalGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsGauge is emitted once per minute counting the total number of ongoing disruptions per namespace,
// or if we fail to determine the namespaced metrics, simply the total number of disruptions found
func (n Sink) MetricDisruptionsGauge(gauge float64, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionsGauge %f\n", gauge)

	return nil
}

// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
func (n Sink) MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionsCount %s %s\n", kind, tags)

	return nil
}

// MetricPodsGauge is emitted once per minute counting the total number of live chaos pods for all ongoing disruptions
func (n Sink) MetricPodsGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricPodsGauge %f\n", gauge)

	return nil
}

// MetricRestart is emitted once, every time the manager container of the chaos-controller starts up
func (n Sink) MetricRestart() error {
	fmt.Println("NOOP: MetricRestart")

	return nil
}

// MetricValidationFailed is emitted in ValidateCreate and ValidateUpdate in the disruption_webhook, specifically and
// only when DisruptionSpec.Validate() returns an error, OR when trying to remove the finalizer from a disruption with
// chaos pods
func (n Sink) MetricValidationFailed(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationFailed %s\n", tags)

	return nil
}

// MetricValidationCreated is emitted once per created Disruption, in the webhook after validation completes.
func (n Sink) MetricValidationCreated(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationCreated %s\n", tags)

	return nil
}

// MetricValidationUpdated is emitted once per Disruption update, in the webhook after validation completes
func (n Sink) MetricValidationUpdated(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationUpdated %s\n", tags)

	return nil
}

// MetricValidationDeleted is emitted once per Disruption delete, in the webhook
func (n Sink) MetricValidationDeleted(tags []string) error {
	n.log.Debugf("NOOP: MetricValidationDeleted %s\n", tags)

	return nil
}

// MetricInformed is emitted every time the manager container's informer is called to check a pod in the chaos-controller's
// namespace, to see if that pod is a chaos-injector pod that needs its Disruption reconciled.
func (n Sink) MetricInformed(tags []string) error {
	n.log.Debugf("NOOP: MetricInformed %s\n", tags)

	return nil
}

// MetricOrphanFound increments when a chaos pod without a corresponding disruption resource is found
func (n Sink) MetricOrphanFound(tags []string) error {
	n.log.Debugf("NOOP: MetricOrphanFound %s\n", tags)

	return nil
}

// MetricWatcherCalls is a counter of watcher calls. This is emitted by every OnChange event for all of our watchers,
// e.g., the chaos pod watcher, the target pod watcher, the disruption watcher.
func (n Sink) MetricWatcherCalls(tags []string) error {
	n.log.Debugf("NOOP: MetricWatcherCalls %s\n", tags)

	return nil
}

// MetricSelectorCacheGauge reports how many caches are still in the cache array to prevent leaks
func (n Sink) MetricSelectorCacheGauge(gauge float64) error {
	n.log.Debugf("NOOP: MetricSelectorCacheGauge %f\n", gauge)

	return nil
}

// MetricTooLate reports when a scheduled Disruption misses its configured time to be run,
// specific to cron and rollout controllers
func (n Sink) MetricTooLate(tags []string) error {
	n.log.Debugf("NOOP: MetricTooLate %s\n", tags)

	return nil
}

// MetricTargetMissing reports when a scheduled Disruption can not find its specific target
// either for the first time or multiple times. A deletion occurs on the final alert
func (n Sink) MetricTargetMissing(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricTargetMissing %v,%s\n", duration, tags)

	return nil
}

// MetricMissingTargetFound reports when a scheduled Disruption which had initially been deemed missing
// is "found" and running in the kubernetes namespace
func (n Sink) MetricMissingTargetFound(tags []string) error {
	n.log.Debugf("NOOP: MetricMissingTargetFound %s\n", tags)

	return nil
}

// MetricMissingTargetDeleted reports when a scheduled Disruption has been deleted by the chaos-controller,
// because its target has been missing for too long
func (n Sink) MetricMissingTargetDeleted(tags []string) error {
	n.log.Debugf("NOOP: MetricMissingTargetDeleted %s\n", tags)

	return nil
}

// MetricNextScheduledTime reports the duration until the next scheduled disruption will run
func (n Sink) MetricNextScheduledTime(duration time.Duration, tags []string) error {
	n.log.Debugf("NOOP: MetricNextScheduledRun %v, s%s\n", duration, tags)

	return nil
}

// MetricDisruptionScheduled reports when a new disruption is scheduled
func (n Sink) MetricDisruptionScheduled(tags []string) error {
	n.log.Debugf("NOOP: MetricDisruptionScheduled %s\n", tags)

	return nil
}

// MetricPausedCron reports when a disruption cron has reconciled in a paused state
func (n Sink) MetricPausedCron(tags []string) error {
	n.log.Debugf("NOOP: MetricPausedCron %s\n", tags)

	return nil
}

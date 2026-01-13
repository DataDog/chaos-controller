// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package metrics

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/DataDog/chaos-controller/o11y/metrics/datadog"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/o11y/metrics/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// Sink describes a metric sink
type Sink interface {
	// Close closes any clients used by this sink
	Close() error
	// GetSinkName returns the name of the sink, either "datadog" or "noop"
	GetSinkName() string
	// GetPrefix returns the prefix used by this sink for all metrics
	GetPrefix() string
	// MetricCleaned is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
	// and does not intend to re-inject.
	MetricCleaned(succeed bool, kind string, tags []string) error
	// MetricCleanedForReinjection is used by the chaos-injector to indicate an injector has cleaned the disruptions from the target,
	// but expects to reinject, i.e., when using the spec.pulse feature
	MetricCleanedForReinjection(succeed bool, kind string, tags []string) error
	// MetricCleanupDuration indicates the duration between a Disruption's deletion timestamp, and when the chaos-controller
	// removes its finalizer
	MetricCleanupDuration(duration time.Duration, tags []string) error
	// MetricInjectDuration indicates the duration between a Disruption's creation timestamp, and when it reaches a status
	// of Injected, indicating all chaos-injector pods have injected into their targets, and we've reached the expected count
	MetricInjectDuration(duration time.Duration, tags []string) error
	// MetricInjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
	// the `succeed` bool argument is false if there was an error while injecting.
	MetricInjected(succeed bool, kind string, tags []string) error
	// MetricReinjected is used by the chaos-injector to indicate it has finished trying to inject the disruption into the target,
	// the `succeed` bool argument is false if there was an error while injecting. This metric is used instead of MetricInjected
	// if the chaos-injector pod is performing any injection after its first, i.e., when using the pulse feature
	MetricReinjected(succeed bool, kind string, tags []string) error
	// MetricPodsCreated is used every time the chaos-controller finishes sending a Create request to the k8s api to
	// schedule a new chaos-injector pod. The `succeed` bool argument is false if there was an error returned.
	MetricPodsCreated(target, instanceName, namespace string, succeed bool) error
	// MetricReconcile is used to count how many times the controller enters any reconcile loop
	MetricReconcile() error
	// MetricReconcileDuration is used at the end of every reconcile loop to indicate the duration that Reconcile() call spent
	MetricReconcileDuration(duration time.Duration, tags []string) error
	// MetricDisruptionCompletedDuration indicates the duration between a Disruption's creation timestamp, and when the chaos-controller
	// removes its finalizer
	MetricDisruptionCompletedDuration(duration time.Duration, tags []string) error
	// MetricDisruptionOngoingDuration indicates the duration between a Disruption's creation timestamp, and the current time.
	// This is emitted approximately every one minute
	MetricDisruptionOngoingDuration(duration time.Duration, tags []string) error
	// MetricStuckOnRemovalCurrent is emitted once per minute counting the number of disruptions _per namespace_
	// that are "stuck on removal", i.e.,
	// we have attempted to clean and delete the disruption, but that has not worked, and a human needs to intervene.
	MetricStuckOnRemovalCurrent(gauge float64, tags []string) error
	// MetricDisruptionsGauge is emitted once per minute counting the total number of ongoing disruptions per namespace,
	// or if we fail to determine the namespaced metrics, simply the total number of disruptions found
	MetricDisruptionsGauge(gauge float64, tags []string) error
	// MetricDisruptionsCount counts finished disruptions, and tags the disruption kind
	MetricDisruptionsCount(kind chaostypes.DisruptionKindName, tags []string) error
	// MetricWatcherCalls is a counter of watcher calls. This is emitted by every OnChange event for all of our watchers,
	// e.g., the chaos pod watcher, the target pod watcher, the disruption watcher.
	MetricWatcherCalls(tags []string) error
	// MetricPodsGauge is emitted once per minute counting the total number of live chaos pods for all ongoing disruptions
	MetricPodsGauge(gauge float64) error
	// MetricRestart is emitted once, every time the manager container of the chaos-controller starts up
	MetricRestart() error
	// MetricValidationFailed is emitted in ValidateCreate and ValidateUpdate in the disruption_webhook, specifically and
	// only when DisruptionSpec.Validate() returns an error, OR when trying to remove the finalizer from a disruption with
	// chaos pods. In a later release, this should be fixed to emit any time any validation in the webhook fails.
	MetricValidationFailed(tags []string) error
	// MetricValidationCreated is emitted once per created Disruption, in the webhook after validation completes.
	MetricValidationCreated(tags []string) error
	// MetricValidationUpdated is emitted once per Disruption update, in the webhook after validation completes
	MetricValidationUpdated(tags []string) error
	// MetricValidationDeleted is emitted once per Disruption delete, in the webhook
	MetricValidationDeleted(tags []string) error
	// MetricInformed is emitted every time the manager container's informer is called to check a pod in the chaos-controller's
	// namespace, to see if that pod is a chaos-injector pod that needs its Disruption reconciled.
	MetricInformed(tags []string) error
	// MetricOrphanFound increments when a chaos pod without a corresponding disruption resource is found
	MetricOrphanFound(tags []string) error
	// MetricTooLate reports when a scheduled Disruption misses its configured time to be run,
	// specific to cron and rollout controllers
	MetricTooLate(tags []string) error
	// MetricTargetMissing reports anytime scheduled Disruption can not find its specified target
	MetricTargetMissing(duration time.Duration, tags []string) error
	// MetricMissingTargetFound reports when a scheduled Disruption's target which had initially been deemed missing
	// is "found" and running in the kubernetes namespace
	MetricMissingTargetFound(tags []string) error
	// MetricMissingTargetDeleted reports when a scheduled Disruption has been deleted by the chaos-controller,
	// because its target has been missing for too long
	MetricMissingTargetDeleted(tags []string) error
	// MetricNextScheduledTime reports the duration until this scheduled Disruption's next scheduled disruption should run
	MetricNextScheduledTime(time time.Duration, tags []string) error
	// MetricDisruptionScheduled reports when a new disruption is scheduled by a cron or rollout
	MetricDisruptionScheduled(tags []string) error
	// MetricPausedCron reports when a disruption cron has reconciled in a paused state
	MetricPausedCron(tags []string) error
}

// GetSink returns an initiated sink
func GetSink(log *zap.SugaredLogger, driver types.SinkDriver, app types.SinkApp) (Sink, error) {
	switch driver {
	case types.SinkDriverDatadog:
		return datadog.New(app)
	case types.SinkDriverNoop:
		return noop.New(log), nil
	default:
		return nil, fmt.Errorf("unsupported metrics sink: %s", driver)
	}
}

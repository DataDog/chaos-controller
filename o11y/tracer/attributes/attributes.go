// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package attributes

// Span attribute keys following OpenTelemetry naming conventions (dot-separated namespaces).
// Use these constants wherever attribute.String/Bool/Int key arguments are needed to avoid
// typos and keep keys consistent across the codebase.
const (
	// === DISRUPTION IDENTITY ===

	DisruptionName            = "disruption.name"
	DisruptionNamespace       = "disruption.namespace"
	DisruptionResourceVersion = "disruption.resource_version"
	DisruptionUser            = "disruption.user"
	DisruptionTarget          = "disruption.target"

	// === DISRUPTION STATE ===

	DisruptionLevel                = "chaos.disruption.level"
	DisruptionKinds                = "chaos.disruption.kinds"
	DisruptionInjectionStatus      = "chaos.disruption.injection_status"
	DisruptionDeleting             = "chaos.disruption.deleting"
	DisruptionHasParentTrace       = "chaos.disruption.has_parent_trace"
	DisruptionChaosPodCount        = "chaos.disruption.chaos_pods_count"
	DisruptionInjStatusBefore      = "chaos.disruption.injection_status.before"
	DisruptionInjStatusAfter       = "chaos.disruption.injection_status.after"
	DisruptionTargetCount          = "chaos.disruption.target_count"
	DisruptionInjectorPodsToCreate = "chaos.disruption.injector_pods_to_create"
	DisruptionCleaned              = "chaos.disruption.cleaned"
	DisruptionStaticTargeting      = "chaos.disruption.static_targeting"
	DisruptionMatchingTargets      = "chaos.disruption.matching_targets_count"
	DisruptionTotalAvailTargets    = "chaos.disruption.total_available_targets"
	DisruptionDesiredTargets       = "chaos.disruption.desired_targets_count"
	DisruptionSelectedTargets      = "chaos.disruption.selected_targets_count"
	DisruptionPotentialTargets     = "chaos.disruption.potential_targets"
	DisruptionEligibleTargets      = "chaos.disruption.eligible_targets_count"

	// === WATCHERS ===

	WatchersKind           = "chaos.watchers.kind"
	WatchersManagerFound   = "chaos.watchers.manager_found"
	WatchersStoredManagers = "chaos.watchers.stored_managers"
	WatchersOrphansRemoved = "chaos.watchers.orphans_removed"
)

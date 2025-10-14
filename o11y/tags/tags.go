// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package tags

import "fmt"

// FormatTag formats a tag with key:value format for metrics and observability
func FormatTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

// Tag keys following Datadog naming conventions (lowercase with underscores)
const (
	// === CHAOS CONTROLLER CORE ENTITIES ===

	// Disruption
	disruptionPrefixKey    = "disruption"
	DisruptionKey          = disruptionPrefixKey
	DisruptionNameKey      = disruptionPrefixKey + "_name"
	DisruptionNamespaceKey = disruptionPrefixKey + "_namespace"
	DisruptionKindKey      = disruptionPrefixKey + "_kind"

	// DisruptionCron
	disruptionCronPrefixKey    = "disruption_cron"
	DisruptionCronNameKey      = disruptionCronPrefixKey + "_name"
	DisruptionCronNamespaceKey = disruptionCronPrefixKey + "_namespace"

	// DisruptionRollout
	disruptionRolloutPrefixKey    = "disruption_rollout"
	DisruptionRolloutNameKey      = disruptionRolloutPrefixKey + "_name"
	DisruptionRolloutNamespaceKey = disruptionRolloutPrefixKey + "_namespace"

	// ChaosPod
	chaosPodPrefixKey         = "chaos_pod"
	ChaosPodNameKey           = chaosPodPrefixKey + "_name"
	ChaosPodNamespaceKey      = chaosPodPrefixKey + "_namespace"
	ChaosPodSpecKey           = chaosPodPrefixKey + "_spec"
	ChaosPodLabelsKey         = chaosPodPrefixKey + "_labels"
	ChaosPodNamesKey          = chaosPodPrefixKey + "_names"
	ChaosPodArgsKey           = chaosPodPrefixKey + "_args"
	ChaosPodContainerCountKey = chaosPodPrefixKey + "_container_count"

	// Controller and Injection
	ControllerKey      = "controller"
	injectionPrefixKey = "injection"
	InjectionStatusKey = injectionPrefixKey + "_status"

	// === KUBERNETES RESOURCES ===

	// Target (generic resource being disrupted)
	targetPrefixKey           = "target"
	TargetNameKey             = targetPrefixKey + "_name"
	TargetNodeNameKey         = targetPrefixKey + "_node_name"
	TargetPodNameKey          = targetPrefixKey + "_pod_name"
	TargetPodUIDKey           = targetPrefixKey + "_pod_uid"
	TargetKindKey             = targetPrefixKey + "_kind"
	TargetLabelsKey           = targetPrefixKey + "_labels"
	TargetNamespaceKey        = targetPrefixKey + "_namespace"
	TargetContainersKey       = targetPrefixKey + "_containers"
	TargetContainerIDKey      = targetPrefixKey + "_container_id"
	TargetDisruptedByKindsKey = targetPrefixKey + "_disrupted_by_kinds"
	TargetLevelKey            = targetPrefixKey + "_level"
	TargetsKey                = "targets"

	// Target selection
	EstimatedEligibleTargetsCountKey = "estimated_eligible_targets_count"
	EstimatedTargetCountKey          = "estimated_target_count"
	EligibleTargetsCountKey          = "eligible_targets_count"
	EligibleTargetsKey               = "eligible_targets"
	PotentialTargetsCountKey         = "potential_targets_count"
	PotentialTargetsKey              = "potential_targets"

	// Pod
	podPrefixKey    = "pod"
	PodNameKey      = podPrefixKey + "_name"
	PodNamespaceKey = podPrefixKey + "_namespace"

	// Container
	containerPrefix  = "container"
	ContainerIDKey   = containerPrefix + "_id"
	ContainerNameKey = containerPrefix + "_name"
	ContainerKey     = "container"

	// Node
	nodePrefixKey = "node"
	NodeNameKey   = nodePrefixKey + "_name"

	// StatefulSet
	statefulSetPrefixKey = "stateful_set"
	StatefulSetNameKey   = statefulSetPrefixKey + "_name"

	// Deployment
	deploymentPrefixKey = "deployment"
	DeploymentNameKey   = deploymentPrefixKey + "_name"

	// Service
	servicePrefixKey    = "service"
	ServiceNamespaceKey = servicePrefixKey + "_namespace"
	ServiceNameKey      = servicePrefixKey + "_name"

	// PVC
	pvcPrefixKey = "pvc"
	PVCNameKey   = pvcPrefixKey + "_name"

	// === OBSERVABILITY AND SYSTEM ===

	// Watcher
	watcherPrefix       = "watcher"
	WatcherNameKey      = watcherPrefix + "_name"
	WatcherNamespaceKey = watcherPrefix + "_namespace"

	// Event notifications
	notifierPrefix                 = "notifier"
	NotifierDisruptionEventKey     = notifierPrefix + "_disruption_event"
	NotifierDisruptionCronEventKey = notifierPrefix + "_disruption_cron_event"

	// Cloud Services Provider
	cloudServicesProviderPrefixKey = "cloud_services_provider"
	CloudProviderNameKey           = cloudServicesProviderPrefixKey + "_name"
	CloudProviderURLKey            = cloudServicesProviderPrefixKey + "_url"
	CloudProviderVersionKey        = cloudServicesProviderPrefixKey + "_version"

	// === COMMON FIELDS ===

	// Generic fields
	ConfigKey          = "config"
	DataKey            = "data"
	HostKey            = "host"
	IndexedValueKey    = "indexed_value"
	KindKey            = "kind"
	LabelsKey          = "labels"
	LevelKey           = "level"
	ObjectKey          = "object"
	PathKey            = "path"
	RequestKey         = "request"
	ResourceVersionKey = "resource_version"
	RuleSpecKey        = "rule_spec"
	SelectorKey        = "selector"
	SpecKey            = "spec"
	TypeKey            = "type"
	ValueKey           = "value"
	WebhookKey         = "webhook"

	// Event and logging
	DatadogTagsKey = "datadog_tags"
	ErrorKey       = "error"
	EventKey       = "event"
	EventObjectKey = "event_object"
	EventTypeKey   = "event_type"
	ExpectedKey    = "expected"
	InformerKey    = "informer"
	MessageKey     = "message"
	ReasonKey      = "reason"
	ReceivedKey    = "received"
	SinkKey        = "sink"
	SourceKey      = "source"

	// Time and scheduling
	CurrentRunKey                 = "current_run"
	DelayedStartToleranceKey      = "delayed_start_tolerance"
	DurationKey                   = "duration"
	EarliestTimeKey               = "earliest_time"
	HistoryKey                    = "history"
	InitialDelayKey               = "initial_delay"
	IntervalKey                   = "interval"
	LastContainerChangeTimeKey    = "last_container_change_time"
	LastScheduleTimeKey           = "last_schedule_time"
	MaxRunsKey                    = "max_runs"
	NextRunKey                    = "next_run"
	NowKey                        = "now"
	PauseDurationKey              = "pause_duration"
	RemainingDurationKey          = "remaining_duration"
	RequeueAfterKey               = "requeue_after"
	RequeueDelayKey               = "requeue_delay"
	RetryIntervalKey              = "retry_interval"
	RunCountKey                   = "run_count"
	ScheduleKey                   = "schedule"
	StressDurationKey             = "stress_duration"
	TimeMissingKey                = "time_missing"
	TimeoutKey                    = "timeout"
	TimestampKey                  = "timestamp"
	TimeUntilNotInjectedBeforeKey = "time_until_not_injected_before"

	// System and process
	AppKey              = "app"
	CpuKey              = "cpu"
	DeviceKey           = "device"
	DriverKey           = "driver"
	ParentPidKey        = "parent_pid"
	PidKey              = "pid"
	SignalKey           = "signal"
	StateKey            = "state"
	StressCpusetKey     = "stress_cpuset"
	StressPidKey        = "stress_pid"
	SysrqPathKey        = "sysrq_path"
	SysrqTriggerPathKey = "sysrq_trigger_path"
	ThreadIdKey         = "thread_id"

	// Network and connectivity
	BandwidthLimitKey      = "bandwidth_limit"
	ChainKey               = "chain"
	CorruptKey             = "corrupt"
	DelayJitterKey         = "delay_jitter"
	DelayKey               = "delay"
	DestinationPodNameKey  = "destination_pod_name"
	DropKey                = "drop"
	DuplicateKey           = "duplicate"
	EndpointAlterationsKey = "endpoint_alterations"
	FilterIPKey            = "filter_ip"
	FilterPriorityKey      = "filter_priority"
	FiltersKey             = "filters"
	HostRecordPairsKey     = "host_record_pairs"
	HostsKey               = "hosts"
	InterfacesKey          = "interfaces"
	LinkNameKey            = "link_name"
	NewIPsKey              = "new_ips"
	OldIpsKey              = "old_ips"
	OutdatedIPKey          = "outdated_ip"
	ResolvedEndpointKey    = "resolved_endpoint"
	ResolvedServiceKey     = "resolved_service"
	RootNsKey              = "rootns"
	SubsystemKey           = "subsystem"
	TableKey               = "table"
	TargetNsKey            = "targetns"
	TargetNsPathKey        = "targetns_path"
	TcServiceFilterKey     = "tc_service_filter"

	// Container and pod lifecycle
	DeleteStorageKey         = "delete_storage"
	ForceDeleteKey           = "force_delete"
	GracePeriodSecondsKey    = "grace_period_seconds"
	NewContainerExistsKey    = "new_container_exists"
	NewContainerIDKey        = "new_container_id"
	OldContainerIDKey        = "old_container_id"
	RestartsKey              = "restarts"

	// State tracking
	ConditionTypeKey     = "condition_type"
	LastStateKey         = "last_state"
	NewHashKey           = "new_hash"
	NewPhaseKey          = "new_phase"
	NewStateKey          = "new_state"
	NewStatusKey         = "new_status"
	NewTargetKindKey     = "new_target_kind"
	NewTargetNameKey     = "new_target_name"
	OldHashKey           = "old_hash"
	OldPhaseKey          = "old_phase"
	OldStatusKey         = "old_status"
	OldTargetKindKey     = "old_target_kind"
	OldTargetNameKey     = "old_target_name"
	StatusKey            = "status"

	// Metrics and counting
	AssignedCpusKey               = "assigned_cpus"
	BpsKey                        = "bps"
	CalculatedPercentOfTotalKey   = "calculated_percent_of_total"
	ClusterThresholdKey           = "cluster_threshold"
	CountKey                      = "count"
	FoundPodsKey                  = "found_pods"
	NamespaceCountKey             = "namespace_count"
	NamespaceThresholdKey         = "namespace_threshold"
	NumActiveDisruptionsKey       = "num_active_disruptions"
	PercentageKey                 = "percentage"
	ProvidedValueKey              = "provided_value"
	StressCountKey                = "stress_count"
	TotalCountKey                 = "total_count"

	// Authentication and user management
	GroupKey               = "group"
	GroupsKey              = "groups"
	PermittedUserGroupsKey = "permitted_user_groups"
	ReqKey                 = "req"
	SlackChannelKey        = "slack_channel"
	UserAddressKey         = "user_address"
	UserGroupsKey          = "user_groups"
	UsernameKey            = "username"

	// Search and query
	IntersectionOfKindsKey  = "intersection_of_kinds"
	OffendingArgumentKey    = "offending_argument"
	QueryPercentParsingKey  = "query_percent_parsing_fail"
	SelectedTargetsKey      = "selected_targets"
	ToFindKey               = "to_find"

	// Control and safety
	AdmissionControllerKey = "admission_controller"
	MaxTimeoutKey          = "max_timeout"
	RetryingKey            = "retrying"
	SafetyNetCatchKey      = "safety_net_catch"
)

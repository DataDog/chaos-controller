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
	DisruptionKey          = "disruption"
	DisruptionNameKey      = "disruption_name"
	DisruptionNamespaceKey = "disruption_namespace"
	DisruptionKindKey      = "disruption_kind"

	// DisruptionCron
	DisruptionCronNameKey      = "disruptioncron_name"
	DisruptionCronNamespaceKey = "disruptioncron_namespace"

	// DisruptionRollout
	DisruptionRolloutNameKey      = "disruption_rollout_name"
	DisruptionRolloutNamespaceKey = "disruption_rollout_namespace"

	// ChaosPod
	ChaosPodNameKey           = "chaos_pod_name"
	ChaosPodNamespaceKey      = "chaos_pod_namespace"
	ChaosPodSpecKey           = "chaos_pod_spec"
	ChaosPodLabelsKey         = "chaos_pod_labels"
	ChaosPodNamesKey          = "chaos_pod_names"
	ChaosPodArgsKey           = "chaos_pod_args"
	ChaosPodContainerCountKey = "chaos_pod_container_count"

	// Controller
	ControllerKey = "controller"

	// Injection
	InjectionStatusKey = "injection_status"

	// === KUBERNETES RESOURCES ===

	// Target (generic resource being disrupted)
	TargetNameKey             = "target_name"
	TargetNodeNameKey         = "target_node_name"
	TargetPodNameKey          = "target_pod_name"
	TargetPodUIDKey           = "target_pod_uid"
	TargetKindKey             = "target_kind"
	TargetLabelsKey           = "target_labels"
	TargetNamespaceKey        = "target_namespace"
	TargetContainersKey       = "target_containers"
	TargetContainerIDKey      = "target_container_id"
	TargetDisruptedByKindsKey = "target_disrupted_by_kinds"
	TargetLevelKey            = "target_level"
	TargetsKey                = "targets"

	// Target selection
	EstimatedEligibleTargetsCountKey = "estimated_eligible_targets_count"
	EstimatedTargetCountKey          = "estimated_target_count"
	EligibleTargetsCountKey          = "eligible_targets_count"
	EligibleTargetsKey               = "eligible_targets"
	PotentialTargetsCountKey         = "potential_targets_count"
	PotentialTargetsKey              = "potential_targets"

	// Pod
	PodNameKey      = "pod_name"
	PodNamespaceKey = "pod_namespace"

	// Container
	ContainerIDKey   = "container_id"
	ContainerNameKey = "container_name"
	ContainerKey     = "container"

	// Node
	NodeNameKey = "node_name"

	// StatefulSet
	StatefulSetNameKey = "stateful_set_name"

	// Deployment
	DeploymentNameKey = "deployment_name"

	// Service
	ServiceNamespaceKey = "service_namespace"
	ServiceNameKey      = "service_name"
	ServiceKey          = "service"

	// PVC
	PVCNameKey = "pvc_name"

	// === OBSERVABILITY AND SYSTEM ===

	// Watcher
	WatcherNameKey      = "watcher_name"
	WatcherNamespaceKey = "watcher_namespace"

	// Event notifications
	NotifierDisruptionEventKey     = "notifier_disruption_event"
	NotifierDisruptionCronEventKey = "notifier_disruption_cron_event"

	// Cloud Services Provider
	CloudProviderNameKey    = "cloud_services_provider_name"
	CloudProviderURLKey     = "cloud_services_provider_url"
	CloudProviderVersionKey = "cloud_services_provider_version"

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
	PulseNextActionTimestampKey   = "pulse_next_action_timestamp"

	// System and process
	AppKey              = "app"
	CPUKey              = "cpu"
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
	ThreadIDKey         = "thread_id"

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
	GrpcEndpointsKey       = "grpc_endpoints"
	GrpcEndpointKey        = "grpc_endpoint"
	GrpcServerAddrKey      = "grpc_server_addr"
	GrpcPortKey            = "grpc_port"
	GrpcErrorKey           = "grpc_error"

	// Container and pod lifecycle
	DeleteStorageKey      = "delete_storage"
	ForceDeleteKey        = "force_delete"
	GracePeriodSecondsKey = "grace_period_seconds"
	NewContainerExistsKey = "new_container_exists"
	NewContainerIDKey     = "new_container_id"
	OldContainerIDKey     = "old_container_id"
	RestartsKey           = "restarts"

	// State tracking
	ConditionTypeKey = "condition_type"
	LastStateKey     = "last_state"
	NewHashKey       = "new_hash"
	NewPhaseKey      = "new_phase"
	NewStateKey      = "new_state"
	NewStatusKey     = "new_status"
	NewTargetKindKey = "new_target_kind"
	NewTargetNameKey = "new_target_name"
	OldHashKey       = "old_hash"
	OldPhaseKey      = "old_phase"
	OldStatusKey     = "old_status"
	OldTargetKindKey = "old_target_kind"
	OldTargetNameKey = "old_target_name"
	StatusKey        = "status"

	// Metrics and counting
	AssignedCpusKey             = "assigned_cpus"
	BpsKey                      = "bps"
	CalculatedPercentOfTotalKey = "calculated_percent_of_total"
	ClusterThresholdKey         = "cluster_threshold"
	CountKey                    = "count"
	FoundPodsKey                = "found_pods"
	NamespaceCountKey           = "namespace_count"
	NamespaceThresholdKey       = "namespace_threshold"
	NumActiveDisruptionsKey     = "num_active_disruptions"
	PercentageKey               = "percentage"
	ProvidedValueKey            = "provided_value"
	StressCountKey              = "stress_count"
	TotalCountKey               = "total_count"

	// Authentication and user management
	GroupKey               = "group"
	GroupsKey              = "groups"
	PermittedUserGroupsKey = "permitted_user_groups"
	ReqKey                 = "req"
	SlackChannelKey        = "slack_channel"
	UserAddressKey         = "user_address"
	UserGroupsKey          = "user_groups"
	UsernameKey            = "username"
	TeamKey                = "team"

	// Search and query
	IntersectionOfKindsKey = "intersection_of_kinds"
	OffendingArgumentKey   = "offending_argument"
	QueryPercentParsingKey = "query_percent_parsing_fail"
	SelectedTargetsKey     = "selected_targets"
	ToFindKey              = "to_find"

	// Control and safety
	AdmissionControllerKey = "admission_controller"
	MaxTimeoutKey          = "max_timeout"
	RetryingKey            = "retrying"
	SafetyNetCatchKey      = "safety_net_catch"
)

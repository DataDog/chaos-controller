// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package env

//nolint:golint
const (
	InjectorMountHost         = "CHAOS_INJECTOR_MOUNT_HOST"
	InjectorMountProc         = "CHAOS_INJECTOR_MOUNT_PROC"
	InjectorMountCgroup       = "CHAOS_INJECTOR_MOUNT_CGROUP"
	InjectorMountSysrq        = "CHAOS_INJECTOR_MOUNT_SYSRQ"
	InjectorMountSysrqTrigger = "CHAOS_INJECTOR_MOUNT_SYSRQ_TRIGGER"
	InjectorTargetPodHostIP   = "TARGET_POD_HOST_IP"
	InjectorChaosPodIP        = "CHAOS_POD_IP"
	InjectorPodName           = "INJECTOR_POD_NAME"
)

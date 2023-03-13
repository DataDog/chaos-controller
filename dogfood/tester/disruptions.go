// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

import (
	"fmt"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Globals

var SELECTOR = []string{"app", "chaos-dogfood-client"}

const CONTAINER = "client-deploy"
const NAMESPACE = "chaos-engineering"
const NAME_PREFIX = "e2etest-"
const DURATION v1beta1.DisruptionDuration = "3m"
const DISK_PRESSURE_PATH = "/mnt/data"

var COUNT = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}
var UNSAFEMODE = &v1beta1.UnsafemodeSpec{
	DisableAll: true,
}
var NETWORK_HOST_SPEC = []v1beta1.NetworkDisruptionHostSpec{
	{
		Host:     fmt.Sprintf("chaos-dogfood-server.%s.svc.cluster.local", NAMESPACE),
		Port:     50051,
		Protocol: "tcp",
	},
}

// Network Disruptions
var network1 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "network-drop30"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts:          NETWORK_HOST_SPEC,
			Drop:           30,
			Corrupt:        0,
			Delay:          0,
			BandwidthLimit: 0,
		},
	},
}

var network2 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "network-drop70"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts:          NETWORK_HOST_SPEC,
			Drop:           70,
			Corrupt:        0,
			Delay:          0,
			BandwidthLimit: 0,
		},
	},
}

var network3 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "network-delay1000"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts:          NETWORK_HOST_SPEC,
			Drop:           0,
			Corrupt:        0,
			Delay:          1000,
			BandwidthLimit: 0,
		},
	},
}

var NETWORK_DISRUPTIONS = []v1beta1.Disruption{network1, network2, network3}

// Disk Disruptions
var diskReadsThresholds = []int{1024, 2048, 4098}

var disk1 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "disk-read1024"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		DiskPressure: &v1beta1.DiskPressureSpec{
			Path: DISK_PRESSURE_PATH,
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				ReadBytesPerSec: &diskReadsThresholds[0],
			},
		},
	},
}

var disk2 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "disk-write2048"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		DiskPressure: &v1beta1.DiskPressureSpec{
			Path: DISK_PRESSURE_PATH,
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				WriteBytesPerSec: &diskReadsThresholds[1],
			},
		},
	},
}

var disk3 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "disk-write4098"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		DiskPressure: &v1beta1.DiskPressureSpec{
			Path: "/mnt/data",
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				WriteBytesPerSec: &diskReadsThresholds[2],
			},
		},
	},
}

var DISK_DISRUPTIONS = []v1beta1.Disruption{disk1, disk2, disk3}

// CPU Disruptions

var cpu1 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprint(NAME_PREFIX, "cpu-cores4"),
		Namespace: NAMESPACE,
	},
	Spec: v1beta1.DisruptionSpec{
		Count:      COUNT,
		Unsafemode: UNSAFEMODE,
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   DURATION,
		CPUPressure: &v1beta1.CPUPressureSpec{
			Count: &intstr.IntOrString{IntVal: 4},
		},
	},
}

var CPU_DISRUPTIONS = []v1beta1.Disruption{cpu1}

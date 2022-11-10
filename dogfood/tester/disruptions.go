// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Globals

var SELECTOR = []string{"app", "chaos-dogfood-client"}
var CONTAINER = "client-deploy"

// Network Disruptions
var network1 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "e2etest-network1",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{
					Host:     "chaos-dogfood-server.chaos-demo.svc.cluster.local",
					Port:     50051,
					Protocol: "tcp",
				},
			},
			Drop:           30,
			Corrupt:        0,
			Delay:          0,
			BandwidthLimit: 0,
		},
	},
}

var network2 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "e2etest-network2",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{
					Host:     "chaos-dogfood-server.chaos-demo.svc.cluster.local",
					Port:     50051,
					Protocol: "tcp",
				},
			},
			Drop:           70,
			Corrupt:        0,
			Delay:          0,
			BandwidthLimit: 0,
		},
	},
}

var network3 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "e2etest-network3",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		Network: &v1beta1.NetworkDisruptionSpec{
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{
					Host:     "chaos-dogfood-server.chaos-demo.svc.cluster.local",
					Port:     50051,
					Protocol: "tcp",
				},
			},
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
		Name:      "e2etest-disk1",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		DiskPressure: &v1beta1.DiskPressureSpec{
			Path: "/mnt/data",
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				ReadBytesPerSec: &diskReadsThresholds[0],
			},
		},
	},
}

var disk2 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "e2etest-disk2",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		DiskPressure: &v1beta1.DiskPressureSpec{
			Path: "/mnt/data",
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				WriteBytesPerSec: &diskReadsThresholds[1],
			},
		},
	},
}

var disk3 = v1beta1.Disruption{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "e2etest-disk3",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
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
		Name:      "e2etest-cpu1",
		Namespace: "chaos-engineering",
	},
	Spec: v1beta1.DisruptionSpec{
		Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		Unsafemode: &v1beta1.UnsafemodeSpec{
			DisableAll: true,
		},
		Selector:   map[string]string{SELECTOR[0]: SELECTOR[1]},
		Containers: []string{CONTAINER},
		Duration:   "3m",
		CPUPressure: &v1beta1.CPUPressureSpec{
			Count: &intstr.IntOrString{IntVal: 4},
		},
	},
}

var CPU_DISRUPTIONS = []v1beta1.Disruption{cpu1}

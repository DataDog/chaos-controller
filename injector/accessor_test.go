// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
)

var _ = Describe("Config.TargetName", func() {
	It("returns TargetNodeName when level is node", func() {
		cfg := Config{
			Log: log,
			Disruption: chaosapi.DisruptionArgs{
				Level:          types.DisruptionLevelNode,
				TargetNodeName: "node-1",
			},
		}
		Expect(cfg.TargetName()).To(Equal("node-1"))
	})

	It("returns UnknownTargetName when level is pod and no container", func() {
		cfg := Config{
			Log: log,
			Disruption: chaosapi.DisruptionArgs{
				Level: types.DisruptionLevelPod,
			},
		}
		Expect(cfg.TargetName()).To(Equal(UnknownTargetName))
	})

	It("returns UnknownTargetName when neither level nor container set", func() {
		cfg := Config{Log: log}
		Expect(cfg.TargetName()).To(Equal(UnknownTargetName))
	})
})

// injectorFactory defines a factory function to create an injector at runtime.
type injectorFactory struct {
	name    string
	create  func() Injector
	expKind types.DisruptionKindName
}

var _ = Describe("Injector accessors", func() {
	var nodeConfig Config

	BeforeEach(func() {
		nodeConfig = Config{
			Log: log,
			Disruption: chaosapi.DisruptionArgs{
				Level:          types.DisruptionLevelNode,
				TargetNodeName: "node-1",
			},
		}
	})

	factories := []injectorFactory{
		{
			name:    "CPUPressure",
			create:  func() Injector { return NewCPUPressureInjector(Config{Log: log}, "50%", nil, nil) },
			expKind: types.DisruptionKindName(types.DisruptionKindCPUPressure),
		},
		{
			name:    "CPUStress",
			create:  func() Injector { return NewCPUStressInjector(Config{Log: log}, 50, nil, nil) },
			expKind: types.DisruptionKindName(types.DisruptionKindCPUStress),
		},
		{
			name:    "MemoryStress",
			create:  func() Injector { return NewMemoryStressInjector(Config{Log: log}, 50, 0, nil) },
			expKind: types.DisruptionKindName(types.DisruptionKindMemoryStress),
		},
		{
			name:    "MemoryPressure",
			create:  func() Injector { return NewMemoryPressureInjector(Config{Log: log}, "50%", 0, nil, nil) },
			expKind: types.DisruptionKindName(types.DisruptionKindMemoryPressure),
		},
		{
			name: "ContainerFailure",
			create: func() Injector {
				return NewContainerFailureInjector(v1beta1.ContainerFailureSpec{}, ContainerFailureInjectorConfig{Config: Config{Log: log}})
			},
			expKind: types.DisruptionKindName(types.DisruptionKindContainerFailure),
		},
		{
			name: "GRPC",
			create: func() Injector {
				return NewGRPCDisruptionInjector(v1beta1.GRPCDisruptionSpec{}, GRPCDisruptionInjectorConfig{Config: Config{Log: log}})
			},
			expKind: types.DisruptionKindName(types.DisruptionKindGRPCDisruption),
		},
		{
			name: "DNS",
			create: func() Injector {
				return NewDNSDisruptionInjector(v1beta1.DNSDisruptionSpec{}, DNSDisruptionInjectorConfig{Config: Config{Log: log}})
			},
			expKind: types.DisruptionKindName(types.DisruptionKindDNSDisruption),
		},
	}

	// DiskFailure: provide mock BPFConfigInformer to bypass eBPF init
	factories = append(factories, injectorFactory{
		name: "DiskFailure",
		create: func() Injector {
			bpfMock := ebpf.NewConfigInformerMock(GinkgoT())
			inj, err := NewDiskFailureInjector(v1beta1.DiskFailureSpec{}, DiskFailureInjectorConfig{
				Config:            Config{Log: log},
				BPFConfigInformer: bpfMock,
			})
			Expect(err).NotTo(HaveOccurred())
			return inj
		},
		expKind: types.DisruptionKindName(types.DisruptionKindDiskFailure),
	})

	// NetworkDisruption: provide mocked IPTables + TrafficController
	factories = append(factories, injectorFactory{
		name: "NetworkDisruption",
		create: func() Injector {
			iptMock := network.NewIPTablesMock(GinkgoT())
			tcMock := network.NewTrafficControllerMock(GinkgoT())
			inj, err := NewNetworkDisruptionInjector(v1beta1.NetworkDisruptionSpec{}, NetworkDisruptionInjectorConfig{
				Config:            Config{Log: log},
				IPTables:          iptMock,
				TrafficController: tcMock,
			})
			Expect(err).NotTo(HaveOccurred())
			return inj
		},
		expKind: types.DisruptionKindName(types.DisruptionKindNetworkDisruption),
	})

	// DiskPressure: provide mock Informer to bypass disk.FromPath (requires real device)
	factories = append(factories, injectorFactory{
		name: "DiskPressure",
		create: func() Injector {
			Expect(os.Setenv(env.InjectorMountHost, "/tmp")).To(Succeed())
			diskMock := disk.NewInformerMock(GinkgoT())
			inj, err := NewDiskPressureInjector(v1beta1.DiskPressureSpec{Path: "/tmp"}, DiskPressureInjectorConfig{
				Config: Config{
					Log:        log,
					Disruption: chaosapi.DisruptionArgs{Level: types.DisruptionLevelNode},
				},
				Informer: diskMock,
			})
			Expect(err).NotTo(HaveOccurred())
			return inj
		},
		expKind: types.DisruptionKindName(types.DisruptionKindDiskPressure),
	})

	// NodeFailure: set required env vars
	factories = append(factories, injectorFactory{
		name: "NodeFailure",
		create: func() Injector {
			Expect(os.Setenv(env.InjectorMountSysrq, "/tmp/sysrq")).To(Succeed())
			Expect(os.Setenv(env.InjectorMountSysrqTrigger, "/tmp/sysrq-trigger")).To(Succeed())
			inj, err := NewNodeFailureInjector(v1beta1.NodeFailureSpec{}, NodeFailureInjectorConfig{
				Config: Config{Log: log},
			})
			Expect(err).NotTo(HaveOccurred())
			return inj
		},
		expKind: types.DisruptionKindName(types.DisruptionKindNodeFailure),
	})

	for _, f := range factories {
		f := f // capture loop var
		It("GetDisruptionKind: "+f.name, func() {
			inj := f.create()
			Expect(inj.GetDisruptionKind()).To(Equal(f.expKind))
		})

		It("TargetName after UpdateConfig: "+f.name, func() {
			inj := f.create()
			inj.UpdateConfig(nodeConfig)
			Expect(inj.TargetName()).To(Equal("node-1"))
		})
	}
})

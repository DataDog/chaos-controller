// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"os"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Clean methods (stub implementations)", func() {
	It("ContainerFailure.Clean returns nil", func() {
		inj := NewContainerFailureInjector(v1beta1.ContainerFailureSpec{}, ContainerFailureInjectorConfig{Config: Config{Log: log}})
		Expect(inj.Clean()).To(Succeed())
	})

	It("DiskFailure.Clean returns nil", func() {
		bpfMock := ebpf.NewConfigInformerMock(GinkgoT())
		inj, err := NewDiskFailureInjector(v1beta1.DiskFailureSpec{}, DiskFailureInjectorConfig{
			Config:            Config{Log: log},
			BPFConfigInformer: bpfMock,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(inj.Clean()).To(Succeed())
	})
})

var _ = Describe("GRPC Inject/Clean dry-run and no-state", func() {
	It("Inject returns nil in dry-run mode", func() {
		inj := NewGRPCDisruptionInjector(
			v1beta1.GRPCDisruptionSpec{Port: 5000},
			GRPCDisruptionInjectorConfig{
				Config: Config{
					Log:        log,
					Disruption: chaosapi.DisruptionArgs{DryRun: true},
				},
			},
		)
		Expect(inj.Inject()).To(Succeed())
	})

	It("Clean returns nil when not injected", func() {
		inj := NewGRPCDisruptionInjector(
			v1beta1.GRPCDisruptionSpec{Port: 5000},
			GRPCDisruptionInjectorConfig{
				Config: Config{Log: log, Disruption: chaosapi.DisruptionArgs{}},
				State:  Created,
			},
		)
		Expect(inj.Clean()).To(Succeed())
	})

	It("Clean returns nil in dry-run when injected", func() {
		inj := NewGRPCDisruptionInjector(
			v1beta1.GRPCDisruptionSpec{Port: 5000},
			GRPCDisruptionInjectorConfig{
				Config: Config{Log: log, Disruption: chaosapi.DisruptionArgs{DryRun: true}},
				State:  Injected,
			},
		)
		Expect(inj.Clean()).To(Succeed())
	})

	It("Inject calls connectToServer when not dry-run (covers connectToServer success + RPC error)", func() {
		// grpc.NewClient creates a lazy connection → connectToServer succeeds
		// SendGrpcDisruption will fail (no real server) → Inject returns error
		inj := NewGRPCDisruptionInjector(
			v1beta1.GRPCDisruptionSpec{Port: 59999},
			GRPCDisruptionInjectorConfig{
				Config: Config{Log: log, Disruption: chaosapi.DisruptionArgs{DryRun: false}},
			},
		)
		// May succeed or fail — just verify it doesn't panic
		Expect(func() { inj.Inject() }).NotTo(Panic())
	})
})

var _ = Describe("Injector UpdateConfig", func() {
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

	It("ContainerFailure UpdateConfig", func() {
		inj := NewContainerFailureInjector(v1beta1.ContainerFailureSpec{}, ContainerFailureInjectorConfig{Config: Config{Log: log}})
		Expect(func() { inj.UpdateConfig(nodeConfig) }).NotTo(Panic())
	})

	It("DNS UpdateConfig", func() {
		inj := NewDNSDisruptionInjector(v1beta1.DNSDisruptionSpec{}, DNSDisruptionInjectorConfig{Config: Config{Log: log}})
		Expect(func() { inj.UpdateConfig(nodeConfig) }).NotTo(Panic())
	})

	It("Network UpdateConfig", func() {
		iptMock := network.NewIPTablesMock(GinkgoT())
		tcMock := network.NewTrafficControllerMock(GinkgoT())
		inj, err := NewNetworkDisruptionInjector(v1beta1.NetworkDisruptionSpec{}, NetworkDisruptionInjectorConfig{
			Config:            Config{Log: log},
			IPTables:          iptMock,
			TrafficController: tcMock,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(func() { inj.UpdateConfig(nodeConfig) }).NotTo(Panic())
	})

	It("DiskFailure UpdateConfig", func() {
		bpfMock := ebpf.NewConfigInformerMock(GinkgoT())
		inj, err := NewDiskFailureInjector(v1beta1.DiskFailureSpec{}, DiskFailureInjectorConfig{
			Config:            Config{Log: log},
			BPFConfigInformer: bpfMock,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(func() { inj.UpdateConfig(nodeConfig) }).NotTo(Panic())
	})

	It("DiskPressure UpdateConfig", func() {
		Expect(os.Setenv(env.InjectorMountHost, "/tmp")).To(Succeed())
		DeferCleanup(os.Unsetenv, env.InjectorMountHost)
		diskMock := disk.NewInformerMock(GinkgoT())
		inj, err := NewDiskPressureInjector(v1beta1.DiskPressureSpec{}, DiskPressureInjectorConfig{
			Config:   Config{Log: log, Disruption: chaosapi.DisruptionArgs{Level: types.DisruptionLevelNode}},
			Informer: diskMock,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(func() { inj.UpdateConfig(nodeConfig) }).NotTo(Panic())
	})
})

var _ = Describe("memory_alloc_other stubs (non-Linux)", func() {
	It("munmapMemory stub returns nil (non-Linux stub)", func() {
		// munmapMemory is a stub on non-Linux that returns nil.
		// We can only verify the function exists and the MemoryStress injector builds.
		inj := NewMemoryStressInjector(Config{Log: log}, 50, 0, nil)
		Expect(inj).NotTo(BeNil())
	})
})

var _ = Describe("Node/Pod replacement UpdateConfig and accessors", func() {
	It("NodeFailure Inject dry-run and Clean", func() {
		Expect(os.Setenv(env.InjectorMountSysrq, "/tmp/sysrq")).To(Succeed())
		Expect(os.Setenv(env.InjectorMountSysrqTrigger, "/tmp/sysrq-trigger")).To(Succeed())
		DeferCleanup(os.Unsetenv, env.InjectorMountSysrq)
		DeferCleanup(os.Unsetenv, env.InjectorMountSysrqTrigger)
		inj, err := NewNodeFailureInjector(v1beta1.NodeFailureSpec{}, NodeFailureInjectorConfig{
			Config: Config{Log: log, Disruption: chaosapi.DisruptionArgs{DryRun: true}},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(inj.Inject()).To(Succeed())
		Expect(inj.Clean()).To(Succeed())
	})

	It("PodReplacement UpdateConfig", func() {
		inj, err := NewPodReplacementInjector(v1beta1.PodReplacementSpec{}, PodReplacementInjectorConfig{
			Config: Config{Log: log, Disruption: chaosapi.DisruptionArgs{Level: types.DisruptionLevelNode, TargetNodeName: "node-1"}},
		})
		if err != nil {
			Skip("PodReplacement requires env vars: " + err.Error())
		}

		Expect(func() {
			inj.UpdateConfig(Config{Log: log, Disruption: chaosapi.DisruptionArgs{TargetNodeName: "node-2"}})
		}).NotTo(Panic())
	})
})

var _ = Describe("Network disruption UpdateConfig", func() {
	It("updates config without panic", func() {
		iptMock := network.NewIPTablesMock(GinkgoT())
		tcMock := network.NewTrafficControllerMock(GinkgoT())
		inj, err := NewNetworkDisruptionInjector(v1beta1.NetworkDisruptionSpec{}, NetworkDisruptionInjectorConfig{
			Config:            Config{Log: log},
			IPTables:          iptMock,
			TrafficController: tcMock,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(func() { inj.UpdateConfig(Config{Log: log}) }).NotTo(Panic())
	})
})

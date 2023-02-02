// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"os"

	"github.com/DataDog/chaos-controller/cgroup/mocks"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Failure", func() {
	var (
		inj           Injector
		config        DNSDisruptionInjectorConfig
		spec          v1beta1.DNSDisruptionSpec
		cgroupManager *mocks.ManagerMock
		netnsManager  *netns.ManagerMock
		iptables      *network.IptablesMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &mocks.ManagerMock{}
		cgroupManager.On("RelativePath", mock.Anything).Return("/kubepod.slice/foo")

		// netns
		netnsManager = &netns.ManagerMock{}
		netnsManager.On("Enter").Return(nil)
		netnsManager.On("Exit").Return(nil)

		// container
		ctn := &container.ContainerMock{}

		// pythonRunner
		pythonRunner := NewMockPythonRunner(GinkgoT())
		pythonRunner.EXPECT().RunPython(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// iptables
		iptables = &network.IptablesMock{}
		iptables.On("CreateChain", mock.Anything).Return(nil)
		iptables.On("ClearAndDeleteChain", mock.Anything).Return(nil)
		iptables.On("AddRuleWithIP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("AddWideFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("AddCgroupFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("PrependRuleSpec", mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteRuleSpec", mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteCgroupFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("ListRules", mock.Anything, mock.Anything).Return([]string{}, nil)

		// environment variables
		Expect(os.Setenv(env.InjectorChaosPodIP, "10.0.0.2")).To(BeNil())

		// config
		config = DNSDisruptionInjectorConfig{
			Config: Config{
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
				Netns:           netnsManager,
				Cgroup:          cgroupManager,
				Level:           chaostypes.DisruptionLevelNode,
			},
			Iptables:     iptables,
			PythonRunner: pythonRunner,
		}

		spec = v1beta1.DNSDisruptionSpec{}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewDNSDisruptionInjector(spec, config)
		Expect(err).To(BeNil())
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		It("should enter and exit the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should create and set the CHAOS-DNS Chain", func() {
			iptables.AssertCalled(GinkgoT(), "CreateChain", "CHAOS-DNS")
			iptables.AssertCalled(GinkgoT(), "AddRuleWithIP", "CHAOS-DNS", "udp", "53", "DNAT", "10.0.0.2")
		})

		Context("disruption is node-level", func() {
			It("creates node-level iptable filter rules", func() {
				iptables.AssertCalled(GinkgoT(), "PrependRuleSpec", "CHAOS-DNS", []string{"-s", "10.0.0.2", "-j", "RETURN"})
				iptables.AssertCalled(GinkgoT(), "PrependRuleSpec", "OUTPUT", []string{"-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"})
				iptables.AssertCalled(GinkgoT(), "PrependRuleSpec", "PREROUTING", []string{"-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"})
			})
		})

		Context("disruption is pod-level", func() {
			BeforeEach(func() {
				config.Level = chaostypes.DisruptionLevelPod
			})
			It("creates pod-level iptable filter rules", func() {
				iptables.AssertCalled(GinkgoT(), "AddCgroupFilterRule", "nat", "OUTPUT", "/kubepod.slice/foo", []string{"-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"})
			})
		})
	})

	Describe("inj.Clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(BeNil())
		})

		It("should enter the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should clear and delete the CHAOS-DNS Chain", func() {
			iptables.AssertCalled(GinkgoT(), "ClearAndDeleteChain", "CHAOS-DNS")
		})

		Context("disruption is node-level", func() {
			It("should clear the node-level iptable rules", func() {
				iptables.AssertCalled(GinkgoT(), "DeleteRule", "OUTPUT", "udp", "53", "CHAOS-DNS")
				iptables.AssertCalled(GinkgoT(), "DeleteRule", "PREROUTING", "udp", "53", "CHAOS-DNS")
			})
		})

		Context("disruption is pod-level", func() {
			BeforeEach(func() {
				config.Level = chaostypes.DisruptionLevelPod
			})
			It("should clear the pod-level iptables rules", func() {
				iptables.AssertCalled(GinkgoT(), "DeleteCgroupFilterRule", "nat", "OUTPUT", "/kubepod.slice/foo", []string{"-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"})
			})
		})

		Context("clean should be idempotent", func() {
			It("should not error even on repeated calls", func() {
				Expect(inj.Clean()).To(BeNil())
				Expect(inj.Clean()).To(BeNil())
				Expect(inj.Clean()).To(BeNil())
			})
		})
	})
})

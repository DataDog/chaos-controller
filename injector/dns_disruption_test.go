// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
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
		inj          Injector
		config       DNSDisruptionInjectorConfig
		spec         v1beta1.DNSDisruptionSpec
		netnsManager *netns.ManagerMock
		iptables     *network.IptablesMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager := &cgroup.ManagerMock{}
		cgroupManager.On("Exists", "net_cls").Return(true, nil)
		cgroupManager.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// netns
		netnsManager = &netns.ManagerMock{}
		netnsManager.On("Enter").Return(nil)
		netnsManager.On("Exit").Return(nil)

		// container
		ctn := &container.ContainerMock{}

		// pythonRunner
		pythonRunner := &PythonRunnerMock{}
		pythonRunner.On("RunPython", mock.Anything).Return(0, "", nil)

		// iptables
		iptables = &network.IptablesMock{}
		iptables.On("CreateChain", mock.Anything).Return(nil)
		iptables.On("ClearAndDeleteChain", mock.Anything).Return(nil)
		iptables.On("AddRuleWithIP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("AddRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// environment variables
		Expect(os.Setenv(chaostypes.ChaosPodIPEnv, "10.0.0.2")).To(BeNil())

		// config
		config = DNSDisruptionInjectorConfig{
			Config: Config{
				Container:   ctn,
				Log:         log,
				MetricsSink: ms,
				Netns:       netnsManager,
				Cgroup:      cgroupManager,
				Level:       chaostypes.DisruptionLevelPod,
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

		Context("iptables rules should be created", func() {
			It("should create one chain and two rules", func() {
				iptables.AssertCalled(GinkgoT(), "AddRule", "OUTPUT", "udp", "53", "CHAOS-DNS")
				iptables.AssertCalled(GinkgoT(), "AddRuleWithIP", "CHAOS-DNS", "udp", "53", "DNAT", "10.0.0.2")
				iptables.AssertCalled(GinkgoT(), "CreateChain", "CHAOS-DNS")
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

		Context("iptables cleanup should happen", func() {
			It("should clear the iptables rules", func() {
				iptables.AssertCalled(GinkgoT(), "DeleteRule", "OUTPUT", "udp", "53", "CHAOS-DNS")
				iptables.AssertCalled(GinkgoT(), "ClearAndDeleteChain", "CHAOS-DNS")
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

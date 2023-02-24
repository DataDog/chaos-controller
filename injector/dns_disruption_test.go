// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
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
		inj            Injector
		config         DNSDisruptionInjectorConfig
		spec           v1beta1.DNSDisruptionSpec
		cgroupManager  *cgroup.ManagerMock
		isCgroupV2Call *mock.Call
		netnsManager   *netns.ManagerMock
		iptables       *network.IptablesMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &cgroup.ManagerMock{}
		cgroupManager.On("RelativePath", mock.Anything).Return("/kubepod.slice/foo")
		cgroupManager.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		isCgroupV2Call = cgroupManager.On("IsCgroupV2").Return(false)

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
		iptables.On("Clear").Return(nil)
		iptables.On("RedirectTo", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("Intercept", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
			iptables.AssertCalled(GinkgoT(), "RedirectTo", "udp", "53", "10.0.0.2")
		})

		Context("disruption is node-level", func() {
			It("creates node-level iptable filter rules", func() {
				iptables.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "", "", "10.0.0.2")
			})
		})

		Context("disruption is pod-level", func() {
			BeforeEach(func() {
				config.Level = chaostypes.DisruptionLevelPod
			})

			Context("with cgroups v1", func() {
				It("enables pod-level net_cls packet marking", func() {
					cgroupManager.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", chaostypes.InjectorCgroupClassID)
				})

				It("creates pod-level iptable filter rules", func() {
					iptables.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "", chaostypes.InjectorCgroupClassID, "10.0.0.2")
				})
			})

			Context("with cgroups v2", func() {
				BeforeEach(func() {
					isCgroupV2Call.Return(true)
				})

				It("creates pod-level iptable filter rules", func() {
					iptables.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "/kubepod.slice/foo", "", "10.0.0.2")
				})
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
			iptables.AssertCalled(GinkgoT(), "Clear")
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

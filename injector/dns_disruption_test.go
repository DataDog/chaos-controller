// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"errors"
	"os"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Failure", func() {
	var (
		inj               Injector
		config            DNSDisruptionInjectorConfig
		spec              v1beta1.DNSDisruptionSpec
		cgroupManagerMock *mocks.CGroupManagerMock
		isCgroupV2Call    *mocks.CGroupManagerMock_IsCgroupV2_Call
		netnsManagerMock  *mocks.NetNSManagerMock
		iptablesMock      *network.IPTablesMock
		fileWriterMock    *mocks.FileWriterMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManagerMock = mocks.NewCGroupManagerMock(GinkgoT())
		cgroupManagerMock.EXPECT().RelativePath(mock.Anything).Return("/kubepod.slice/foo").Maybe()
		cgroupManagerMock.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		isCgroupV2Call = cgroupManagerMock.EXPECT().IsCgroupV2().Return(false)
		isCgroupV2Call.Maybe()

		// netns
		netnsManagerMock = mocks.NewNetNSManagerMock(GinkgoT())
		netnsManagerMock.EXPECT().Enter().Return(nil).Maybe()
		netnsManagerMock.EXPECT().Exit().Return(nil).Maybe()

		// container
		ctn := mocks.NewContainerMock(GinkgoT())

		// pythonRunner
		pythonRunner := mocks.NewPythonRunnerMock(GinkgoT())
		pythonRunner.EXPECT().RunPython(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe().Maybe()

		// iptables
		iptablesMock = network.NewIPTablesMock(GinkgoT())
		iptablesMock.EXPECT().Clear().Return(nil).Maybe()
		iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		// fileWriter
		fileWriterMock = mocks.NewFileWriterMock(GinkgoT())
		fileWriterMock.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		// environment variables
		Expect(os.Setenv(env.InjectorChaosPodIP, "10.0.0.2")).To(BeNil())

		// config
		config = DNSDisruptionInjectorConfig{
			Config: Config{
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
				Netns:           netnsManagerMock,
				Cgroup:          cgroupManagerMock,
				Disruption: api.DisruptionArgs{
					Level: chaostypes.DisruptionLevelNode,
				},
			},
			IPTables:     iptablesMock,
			PythonRunner: pythonRunner,
			FileWriter:   fileWriterMock,
		}

		spec = v1beta1.DNSDisruptionSpec{}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewDNSDisruptionInjector(spec, config)
		Expect(err).To(BeNil())
	})

	Describe("inj.Inject", func() {
		var injectError error

		JustBeforeEach(func() {
			var err error
			inj, err = NewDNSDisruptionInjector(spec, config)
			Expect(err).To(BeNil())
			injectError = inj.Inject()
		})

		Context("with missing env variable CHAOS_POD_IP", func() {
			BeforeEach(func() {
				err := os.Unsetenv(env.InjectorChaosPodIP)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should return an error", func() {
				Expect(injectError).Should(HaveOccurred())
				Expect(injectError.Error()).To(Equal("CHAOS_POD_IP environment variable must be set with the chaos pod IP"))
			})
		})

		Context("with an error during the enter of the targeted network namespace", func() {
			BeforeEach(func() {
				managerErrorMock := mocks.NewNetNSManagerMock(GinkgoT())
				managerErrorMock.EXPECT().Enter().Return(errors.New("message")).Maybe()
				config.Netns = managerErrorMock
			})

			It("should return an error", func() {
				Expect(injectError).Should(HaveOccurred())
				Expect(injectError.Error()).To(Equal("unable to enter the given container network namespace: message"))
			})
		})

		Context("with an error during the set up of iptables rules to redirect dns requests to the injector pod", func() {
			BeforeEach(func() {
				iptablesErrorMock := network.NewIPTablesMock(GinkgoT())
				iptablesErrorMock.EXPECT().RedirectTo("udp", "53", mock.Anything).Return(errors.New("message")).Maybe()
				config.IPTables = iptablesErrorMock
			})

			It("should return an error", func() {
				Expect(injectError).Should(HaveOccurred())
				Expect(injectError.Error()).To(Equal("unable to create new iptables rule: message"))
			})
		})

		It("should not return an error", func() {
			Expect(injectError).ShouldNot(HaveOccurred())
		})

		It("should enter and exit the target network namespace", func() {
			netnsManagerMock.AssertCalled(GinkgoT(), "Enter")
			netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Enter", 1)
			netnsManagerMock.AssertCalled(GinkgoT(), "Exit")
			netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Exit", 1)
		})

		It("should create and set the CHAOS-DNS Chain", func() {
			iptablesMock.AssertCalled(GinkgoT(), "RedirectTo", "udp", "53", "10.0.0.2")
			iptablesMock.AssertNumberOfCalls(GinkgoT(), "RedirectTo", 1)
		})

		Context("disruption is node-level", func() {
			It("creates node-level iptable filter rules", func() {
				iptablesMock.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "", "", "10.0.0.2")
				iptablesMock.AssertNumberOfCalls(GinkgoT(), "Intercept", 1)
			})

			Context("with an error during the re-route of all pods under node except for injector pod itself", func() {
				BeforeEach(func() {
					errorIptableMock := network.NewIPTablesMock(GinkgoT())
					errorIptableMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
					errorIptableMock.EXPECT().Intercept("udp", "53", "", "", mock.Anything).Return(errors.New("message")).Maybe()
					config.IPTables = errorIptableMock
				})

				It("should return an error", func() {
					Expect(injectError).Should(HaveOccurred())
					Expect(injectError.Error()).Should(Equal("unable to create new iptables rule: message"))
				})
			})
		})

		Context("disruption is pod-level", func() {
			BeforeEach(func() {
				config.Disruption.Level = chaostypes.DisruptionLevelPod
			})

			Context("on init", func() {
				BeforeEach(func() {
					config.Disruption.OnInit = true
				})

				It("should not call cgroup functions", func() {
					cgroupManagerMock.AssertNotCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", chaostypes.InjectorCgroupClassID)
				})

				It("should not create iptable filter rules", func() {
					iptablesMock.AssertNotCalled(GinkgoT(), "Intercept", "udp", "53", "", chaostypes.InjectorCgroupClassID, "10.0.0.2")
					iptablesMock.AssertNotCalled(GinkgoT(), "Intercept", "udp", "53", "/kubepod.slice/foo", "", "10.0.0.2")
				})

				It("should redirect all dns related traffic in the pod to CHAOS-DNS", func() {
					iptablesMock.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "", "", "")
					iptablesMock.AssertNumberOfCalls(GinkgoT(), "Intercept", 1)
				})

				Context("with an error from iptable", func() {
					BeforeEach(func() {
						iptableErrorMock := network.NewIPTablesMock(GinkgoT())
						iptableErrorMock.EXPECT().RedirectTo("udp", "53", mock.Anything).Return(nil).Maybe()
						iptableErrorMock.EXPECT().Intercept("udp", "53", "", "", "").Return(errors.New("message")).Maybe()
						config.IPTables = iptableErrorMock
					})

					It("should return an error", func() {
						Expect(injectError).Should(HaveOccurred())
						Expect(injectError.Error()).Should(Equal("unable to create new iptables rule: message"))
					})
				})
			})

			Context("with cgroups v1", func() {
				It("enables pod-level net_cls packet marking", func() {
					cgroupManagerMock.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", chaostypes.InjectorCgroupClassID)
					cgroupManagerMock.AssertNumberOfCalls(GinkgoT(), "Write", 1)
				})

				It("creates pod-level iptable filter rules", func() {
					iptablesMock.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "", chaostypes.InjectorCgroupClassID, "10.0.0.2")
					iptablesMock.AssertNumberOfCalls(GinkgoT(), "Intercept", 1)
				})

				Context("with an error from the write function of cgroup", func() {
					BeforeEach(func() {
						cgroupErrorMock := mocks.NewCGroupManagerMock(GinkgoT())
						cgroupErrorMock.EXPECT().IsCgroupV2().Return(false).Maybe()
						cgroupErrorMock.EXPECT().Write("net_cls", "net_cls.classid", mock.Anything).Return(errors.New("message")).Maybe()
						config.Cgroup = cgroupErrorMock
					})

					It("should return an error", func() {
						Expect(injectError).Should(HaveOccurred())
						Expect(injectError.Error()).Should(Equal("unable to write net_cls classid: message"))
					})
				})

				Context("with an error from the intercept function of iptables", func() {
					BeforeEach(func() {
						iptablesErrorMock := network.NewIPTablesMock(GinkgoT())
						iptablesErrorMock.EXPECT().RedirectTo("udp", "53", mock.Anything).Return(nil).Maybe()
						iptablesErrorMock.EXPECT().Intercept("udp", "53", "", chaostypes.InjectorCgroupClassID, "10.0.0.2").Return(errors.New("message")).Maybe()
						config.IPTables = iptablesErrorMock
					})

					It("should return an error", func() {
						Expect(injectError).Should(HaveOccurred())
						Expect(injectError.Error()).Should(Equal("unable to create new iptables rule: message"))
					})
				})
			})

			Context("with cgroups v2", func() {
				BeforeEach(func() {
					isCgroupV2Call.Return(true)
				})

				It("should filter packets on cgroup for cgroup v2", func() {
					cgroupManagerMock.AssertCalled(GinkgoT(), "IsCgroupV2")
					cgroupManagerMock.AssertNumberOfCalls(GinkgoT(), "IsCgroupV2", 1)
				})

				It("creates pod-level iptable filter rules", func() {
					iptablesMock.AssertCalled(GinkgoT(), "Intercept", "udp", "53", "/kubepod.slice/foo", "", "10.0.0.2")
					iptablesMock.AssertNumberOfCalls(GinkgoT(), "Intercept", 1)
				})
			})
		})
	})

	Describe("inj.Clean", func() {
		var cleanError error

		JustBeforeEach(func() {
			cleanError = inj.Clean()
		})

		It("should not return an error", func() {
			Expect(cleanError).To(BeNil())
		})

		It("should enter/exit the target network namespace", func() {
			netnsManagerMock.AssertCalled(GinkgoT(), "Enter")
			netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Enter", 1)
			netnsManagerMock.AssertCalled(GinkgoT(), "Exit")
			netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Exit", 1)
		})

		It("should clear and delete the CHAOS-DNS Chain", func() {
			iptablesMock.AssertCalled(GinkgoT(), "Clear")
			iptablesMock.AssertNumberOfCalls(GinkgoT(), "Clear", 1)
		})

		Context("with an error from the enter netns function", func() {
			BeforeEach(func() {
				netnsErrorMock := mocks.NewNetNSManagerMock(GinkgoT())
				netnsErrorMock.EXPECT().Enter().Return(errors.New("message")).Maybe()
				config.Netns = netnsErrorMock
			})

			It("should return an error", func() {
				Expect(cleanError).Should(HaveOccurred())
				Expect(cleanError.Error()).Should(Equal("unable to enter the given container network namespace: message"))
			})
		})

		Context("with an error from the clear iptables function", func() {
			BeforeEach(func() {
				iptablesErrorMock := network.NewIPTablesMock(GinkgoT())
				iptablesErrorMock.EXPECT().Clear().Return(errors.New("message")).Maybe()
				config.IPTables = iptablesErrorMock
			})

			It("should return an error", func() {
				Expect(cleanError).Should(HaveOccurred())
				Expect(cleanError.Error()).Should(Equal("unable to clean iptables rules and chain: message"))
			})
		})

		Context("with an error from the exit netns function", func() {
			BeforeEach(func() {
				netnsErrorMock := mocks.NewNetNSManagerMock(GinkgoT())
				netnsErrorMock.EXPECT().Enter().Return(nil).Maybe()
				netnsErrorMock.EXPECT().Exit().Return(errors.New("message")).Maybe()
				config.Netns = netnsErrorMock
			})

			It("should return an error", func() {
				Expect(cleanError).Should(HaveOccurred())
				Expect(cleanError.Error()).Should(Equal("unable to exit the given container network namespace: message"))
			})
		})

		Context("with cgroup v1", func() {
			It("should remove the net_cls classid", func() {
				cgroupManagerMock.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0")
				cgroupManagerMock.AssertNumberOfCalls(GinkgoT(), "Write", 1)
			})

			Context("with an error from write cgroup function", func() {
				BeforeEach(func() {
					cgroupErrorMock := mocks.NewCGroupManagerMock(GinkgoT())
					cgroupErrorMock.EXPECT().IsCgroupV2().Return(false).Maybe()
					cgroupErrorMock.EXPECT().Write("net_cls", "net_cls.classid", "0").Return(errors.New("message")).Maybe()
					config.Cgroup = cgroupErrorMock
				})

				It("should return an error", func() {
					Expect(cleanError).Should(HaveOccurred())
				})
			})
		})

		Context("with cgroup v2", func() {
			BeforeEach(func() {
				isCgroupV2Call.Return(true)
			})

			It("should not remove the net_cls classid", func() {
				cgroupManagerMock.AssertNotCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0")
			})
		})

		Context("clean should be idempotent", func() {
			It("should not error even on repeated calls", func() {
				Expect(inj.Clean()).To(BeNil())
				Expect(inj.Clean()).To(BeNil())
				Expect(inj.Clean()).To(BeNil())
				netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Enter", 4)
				netnsManagerMock.AssertNumberOfCalls(GinkgoT(), "Exit", 4)
				iptablesMock.AssertNumberOfCalls(GinkgoT(), "Clear", 4)
			})
		})
	})
})

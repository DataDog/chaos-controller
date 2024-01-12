// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package ebpf_test

import (
	"fmt"

	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sys/unix"
)

var validKernelConfig = `
{
  "system_config": {
	"unprivileged_bpf_disabled": 2,
	"bpf_jit_enable": 1,
	"bpf_jit_harden": 0,
	"bpf_jit_kallsyms": 1,
	"bpf_jit_limit": -201326592,
	"CONFIG_BPF": "y",
	"CONFIG_BPF_SYSCALL": "y",
	"CONFIG_HAVE_EBPF_JIT": "y",
	"CONFIG_BPF_JIT": "y",
	"CONFIG_BPF_JIT_ALWAYS_ON": "y",
	"CONFIG_DEBUG_INFO_BTF": "y",
	"CONFIG_DEBUG_INFO_BTF_MODULES": "y",
	"CONFIG_CGROUPS": "y",
	"CONFIG_CGROUP_BPF": "y",
	"CONFIG_CGROUP_NET_CLASSID": "y",
	"CONFIG_SOCK_CGROUP_DATA": "y",
	"CONFIG_BPF_EVENTS": "y",
	"CONFIG_KPROBE_EVENTS": "y",
	"CONFIG_UPROBE_EVENTS": "y",
	"CONFIG_TRACING": "y",
	"CONFIG_FTRACE_SYSCALLS": "y",
	"CONFIG_FUNCTION_ERROR_INJECTION": "y",
	"CONFIG_BPF_KPROBE_OVERRIDE": "y",
	"CONFIG_NET": "y",
	"CONFIG_XDP_SOCKETS": "y",
	"CONFIG_LWTUNNEL_BPF": "y",
	"CONFIG_NET_ACT_BPF": "m",
	"CONFIG_NET_CLS_BPF": "m",
	"CONFIG_NET_CLS_ACT": "y",
	"CONFIG_NET_SCH_INGRESS": "m",
	"CONFIG_XFRM": "y",
	"CONFIG_IP_ROUTE_CLASSID": "y",
	"CONFIG_IPV6_SEG6_BPF": "y",
	"CONFIG_BPF_LIRC_MODE2": null,
	"CONFIG_BPF_STREAM_PARSER": "y",
	"CONFIG_NETFILTER_XT_MATCH_BPF": "m",
	"CONFIG_BPFILTER": "y",
	"CONFIG_BPFILTER_UMH": "m",
	"CONFIG_TEST_BPF": "m",
	"CONFIG_HZ": "250"
  },
  "map_types": {
	"have_hash_map_type": true,
	"have_array_map_type": true,
	"have_prog_array_map_type": true,
	"have_perf_event_array_map_type": true,
	"have_percpu_hash_map_type": true,
	"have_percpu_array_map_type": true,
	"have_stack_trace_map_type": true,
	"have_cgroup_array_map_type": true,
	"have_lru_hash_map_type": true,
	"have_lru_percpu_hash_map_type": true,
	"have_lpm_trie_map_type": true,
	"have_array_of_maps_map_type": true,
	"have_hash_of_maps_map_type": true,
	"have_devmap_map_type": true,
	"have_sockmap_map_type": true,
	"have_cpumap_map_type": true,
	"have_xskmap_map_type": true,
	"have_sockhash_map_type": true,
	"have_cgroup_storage_map_type": true,
	"have_reuseport_sockarray_map_type": true,
	"have_percpu_cgroup_storage_map_type": false,
	"have_queue_map_type": true,
	"have_stack_map_type": false
  }
}
`

var _ = Describe("ConfigInformer", func() {

	var (
		bpftoolExecutorMock *ebpf.ExecutorMock
		configInformer      ebpf.ConfigInformer
		err                 error
		statFSMock          *mocks.StatFSMock
		unameFuncMock       func() (unix.Utsname, error)
	)

	BeforeEach(func() {
		// Arrange
		bpftoolExecutorMock = ebpf.NewExecutorMock(GinkgoT())
		bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, validKernelConfig, nil).Maybe()

		statFSMock = mocks.NewStatFSMock(GinkgoT())
		statFSMock.EXPECT().Stat(mock.Anything).Return(nil, nil).Maybe()

		unameFuncMock = func() (unix.Utsname, error) {
			return unix.Utsname{}, nil
		}
	})

	JustBeforeEach(func() {
		// Action
		configInformer, _ = ebpf.NewConfigInformer(log, false, bpftoolExecutorMock, statFSMock, unameFuncMock)
	})

	When("GetKernelFeatures method is called", func() {
		var (
			features ebpf.Features
		)

		JustBeforeEach(func() {
			// Action
			features, err = configInformer.GetKernelFeatures()
		})

		DescribeTable("success cases", func(bpftoolOutput string, expectedFeatures ebpf.Features) {
			//	Arrange
			bpftoolExecutorMock := ebpf.NewExecutorMock(GinkgoT())
			bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, bpftoolOutput, nil)

			configInformer, err := ebpf.NewConfigInformer(log, false, bpftoolExecutorMock, nil, nil)
			Expect(err).ShouldNot(HaveOccurred())

			// Action
			features, err := configInformer.GetKernelFeatures()

			// Assert
			Expect(features).To(Equal(expectedFeatures))
			Expect(err).ShouldNot(HaveOccurred())

		},
			Entry("an empty content",
				"{}",
				ebpf.Features{},
			),
			Entry("a system_config",
				`
{
  "system_config": {
	  "unprivileged_bpf_disabled": 2,
	  "bpf_jit_enable": 1,
	  "bpf_jit_harden": 0,
	  "bpf_jit_kallsyms": 1,
	  "bpf_jit_limit": -201326592,
	  "CONFIG_BPF": "y",
	  "CONFIG_BPF_SYSCALL": "y",
	  "CONFIG_HAVE_EBPF_JIT": "y",
	  "CONFIG_BPF_JIT": "y",
	  "CONFIG_BPF_JIT_ALWAYS_ON": "y",
	  "CONFIG_DEBUG_INFO_BTF": "y",
	  "CONFIG_DEBUG_INFO_BTF_MODULES": "y",
	  "CONFIG_CGROUPS": "y",
	  "CONFIG_CGROUP_BPF": "y",
	  "CONFIG_CGROUP_NET_CLASSID": "y",
	  "CONFIG_SOCK_CGROUP_DATA": "y",
	  "CONFIG_BPF_EVENTS": "y",
	  "CONFIG_KPROBE_EVENTS": "y",
	  "CONFIG_UPROBE_EVENTS": "y",
	  "CONFIG_TRACING": "y",
	  "CONFIG_FTRACE_SYSCALLS": "y",
	  "CONFIG_FUNCTION_ERROR_INJECTION": "y",
	  "CONFIG_BPF_KPROBE_OVERRIDE": "y",
	  "CONFIG_NET": "y",
	  "CONFIG_XDP_SOCKETS": "y",
	  "CONFIG_LWTUNNEL_BPF": "y",
	  "CONFIG_NET_ACT_BPF": "m",
	  "CONFIG_NET_CLS_BPF": "m",
	  "CONFIG_NET_CLS_ACT": "y",
	  "CONFIG_NET_SCH_INGRESS": "m",
	  "CONFIG_XFRM": "y",
	  "CONFIG_IP_ROUTE_CLASSID": "y",
	  "CONFIG_IPV6_SEG6_BPF": "y",
	  "CONFIG_BPF_LIRC_MODE2": null,
	  "CONFIG_BPF_STREAM_PARSER": "y",
	  "CONFIG_NETFILTER_XT_MATCH_BPF": "m",
	  "CONFIG_BPFILTER": "y",
	  "CONFIG_BPFILTER_UMH": "m",
	  "CONFIG_TEST_BPF": "m",
	  "CONFIG_HZ": "250"
	}
}
`,
				ebpf.Features{
					SystemConfig: ebpf.SystemConfig{
						UnprivilegedBpfDisabled:      2,
						BpfJitEnable:                 1,
						BpfJitHarden:                 0,
						BpfJitKallsyms:               1,
						BpfJitLimit:                  -201326592,
						ConfigBpf:                    "y",
						ConfigBpfSyscall:             "y",
						ConfigHaveEbpfJit:            "y",
						ConfigBpfJit:                 "y",
						ConfigBpfJitAlwaysOn:         "y",
						ConfigCgroups:                "y",
						ConfigCgroupBpf:              "y",
						ConfigCgroupNetClassID:       "y",
						ConfigSockCgroupData:         "y",
						ConfigBpfEvents:              "y",
						ConfigKprobeEvents:           "y",
						ConfigUprobeEvents:           "y",
						ConfigTracing:                "y",
						ConfigFtraceSyscalls:         "y",
						ConfigFunctionErrorInjection: "y",
						ConfigBpfKprobeOverride:      "y",
						ConfigNet:                    "y",
						ConfigXdpSockets:             "y",
						ConfigLwtunnelBpf:            "y",
						ConfigNetActBpf:              "m",
						ConfigNetClsBpf:              "m",
						ConfigNetClsAct:              "y",
						ConfigNetSchIngress:          "m",
						ConfigXfrm:                   "y",
						ConfigIPRouteClassID:         "y",
						ConfigIPv6Seg6Bpf:            "y",
						ConfigBpfLircMode2:           "",
						ConfigBpfStreamParser:        "y",
						ConfigNetfilterXtMatchBpf:    "m",
						ConfigBpfilter:               "y",
						ConfigBpfilterUmh:            "m",
						ConfigTestBpf:                "m",
						ConfigKernelHz:               "250",
					},
				},
			),
			Entry("a map_types",
				`
{
  "map_types": {
	  "have_hash_map_type": true,
	  "have_array_map_type": true,
	  "have_prog_array_map_type": true,
	  "have_perf_event_array_map_type": true,
	  "have_percpu_hash_map_type": true,
	  "have_percpu_array_map_type": true,
	  "have_stack_trace_map_type": true,
	  "have_cgroup_array_map_type": true,
	  "have_lru_hash_map_type": true,
	  "have_lru_percpu_hash_map_type": true,
	  "have_lpm_trie_map_type": true,
	  "have_array_of_maps_map_type": true,
	  "have_hash_of_maps_map_type": true,
	  "have_devmap_map_type": true,
	  "have_sockmap_map_type": true,
	  "have_cpumap_map_type": true,
	  "have_xskmap_map_type": true,
	  "have_sockhash_map_type": true,
	  "have_cgroup_storage_map_type": true,
	  "have_reuseport_sockarray_map_type": true,
	  "have_percpu_cgroup_storage_map_type": false,
	  "have_queue_map_type": true,
	  "have_stack_map_type": false
	}
}
`,
				ebpf.Features{
					MapTypes: ebpf.MapTypes{
						HaveHashMapType:                true,
						HaveArrayMapType:               true,
						HaveProgArrayMapType:           true,
						HavePerfEventArrayMapType:      true,
						HavePercpuHashMapType:          true,
						HavePercpuArrayMapType:         true,
						HaveStackTraceMapType:          true,
						HaveCgroupArrayMapType:         true,
						HaveLruHashMapType:             true,
						HaveLruPercpuHashMapType:       true,
						HaveLpmTrieMapType:             true,
						HaveArrayOfMapsMapType:         true,
						HaveHashOfMapsMapType:          true,
						HaveDevmapMapType:              true,
						HaveSockmapMapType:             true,
						HaveCpumapMapType:              true,
						HaveXskmapMapType:              true,
						HaveSockhashMapType:            true,
						HaveCgroupStorageMapType:       true,
						HaveReuseportSockarrayMapType:  true,
						HavePercpuCgroupStorageMapType: false,
						HaveQueueMapType:               true,
						HaveStackMapType:               false,
					},
				},
			))

		Describe("error cases", func() {
			When("the bptfool return an error", func() {
				BeforeEach(func() {
					// Arrange
					bpftoolExecutorMock = ebpf.NewExecutorMock(GinkgoT())
					bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, "", fmt.Errorf("an error occured"))
				})

				It("should return the error", func() {
					// Assert
					Expect(features).To(Equal(ebpf.Features{}))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Errorf("an error occured")))
				})
			})

			When("the stdout of the bptfool is not a valid json", func() {
				BeforeEach(func() {
					// Arrange
					bpftoolExecutorMock = ebpf.NewExecutorMock(GinkgoT())
					bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, "", nil)
				})

				It("should return the error", func() {
					// Assert
					Expect(features).To(Equal(ebpf.Features{}))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("unexpected end of JSON input"))
				})
			})
		})
	})

	When("GetRequiredSystemConfig method is called", func() {
		DescribeTable("it should return a map with all required system config",
			func(systemConfig string, expectedResult ebpf.KernelParams) {
				//	Arrange
				bpftoolExecutorMock := ebpf.NewExecutorMock(GinkgoT())
				bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, systemConfig, nil).Once()

				configInformer, err := ebpf.NewConfigInformer(log, false, bpftoolExecutorMock, nil, nil)
				Expect(err).ShouldNot(HaveOccurred())

				// Action && Assert
				Expect(configInformer.GetRequiredSystemConfig()).To(Equal(expectedResult))
			},
			Entry("with all system config enabled",
				validKernelConfig,
				ebpf.KernelParams{
					"CONFIG_BPF": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
					"CONFIG_BPF_SYSCALL": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
					"CONFIG_BPF_JIT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
					"CONFIG_HAVE_EBPF_JIT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
					"CONFIG_BPF_KPROBE_OVERRIDE": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
					"CONFIG_NET_CLS_ACT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     true,
					},
				}),
			Entry("with all system config disabled",
				`
{
  "system_config": {
	"CONFIG_BPF": "n",
	"CONFIG_BPF_SYSCALL": "n",
	"CONFIG_HAVE_EBPF_JIT": "n",
	"CONFIG_BPF_JIT": "n",
	"CONFIG_BPF_KPROBE_OVERRIDE": "n",
	"CONFIG_NET_CLS_ACT": "n"
  }
}
`,
				ebpf.KernelParams{
					"CONFIG_BPF": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
					"CONFIG_BPF_SYSCALL": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
					"CONFIG_BPF_JIT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
					"CONFIG_HAVE_EBPF_JIT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
					"CONFIG_BPF_KPROBE_OVERRIDE": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
					"CONFIG_NET_CLS_ACT": ebpf.KernelOption{
						Description: "Essential eBPF infrastructure",
						Enabled:     false,
					},
				}),
		)
	})

	When("GetMapTypes method is called", func() {
		DescribeTable("it should return a map with all required map types",
			func(mapTypeConfig string, expectedMapTypes ebpf.MapTypes) {
				//	Arrange
				bpftoolExecutorMock := ebpf.NewExecutorMock(GinkgoT())
				bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, mapTypeConfig, nil).Once()

				configInformer, err := ebpf.NewConfigInformer(log, false, bpftoolExecutorMock, nil, nil)
				Expect(err).ShouldNot(HaveOccurred())

				// Action && Assert
				Expect(configInformer.GetMapTypes()).To(Equal(expectedMapTypes))
			},
			Entry("with a valid kernel config",
				validKernelConfig,
				ebpf.MapTypes{
					HaveHashMapType:                true,
					HaveArrayMapType:               true,
					HaveProgArrayMapType:           true,
					HavePerfEventArrayMapType:      true,
					HavePercpuHashMapType:          true,
					HavePercpuArrayMapType:         true,
					HaveStackTraceMapType:          true,
					HaveCgroupArrayMapType:         true,
					HaveLruHashMapType:             true,
					HaveLruPercpuHashMapType:       true,
					HaveLpmTrieMapType:             true,
					HaveArrayOfMapsMapType:         true,
					HaveHashOfMapsMapType:          true,
					HaveDevmapMapType:              true,
					HaveSockmapMapType:             true,
					HaveCpumapMapType:              true,
					HaveXskmapMapType:              true,
					HaveSockhashMapType:            true,
					HaveCgroupStorageMapType:       true,
					HaveReuseportSockarrayMapType:  true,
					HavePercpuCgroupStorageMapType: false,
					HaveQueueMapType:               true,
					HaveStackMapType:               false,
				},
			),
			Entry("with all map types disabled",
				`
{
  "map_types": {
	  "have_hash_map_type": false,
	  "have_array_map_type": false,
	  "have_prog_array_map_type": false,
	  "have_perf_event_array_map_type": false,
	  "have_percpu_hash_map_type": false,
	  "have_percpu_array_map_type": false,
	  "have_stack_trace_map_type": false,
	  "have_cgroup_array_map_type": false,
	  "have_lru_hash_map_type": false,
	  "have_lru_percpu_hash_map_type": false,
	  "have_lpm_trie_map_type": false,
	  "have_array_of_maps_map_type": false,
	  "have_hash_of_maps_map_type": false,
	  "have_devmap_map_type": false,
	  "have_sockmap_map_type": false,
	  "have_cpumap_map_type": false,
	  "have_xskmap_map_type": false,
	  "have_sockhash_map_type": false,
	  "have_cgroup_storage_map_type": false,
	  "have_reuseport_sockarray_map_type": false,
	  "have_percpu_cgroup_storage_map_type": false,
	  "have_queue_map_type": false,
	  "have_stack_map_type": false
	}
}
`,
				ebpf.MapTypes{
					HaveHashMapType:                false,
					HaveArrayMapType:               false,
					HaveProgArrayMapType:           false,
					HavePerfEventArrayMapType:      false,
					HavePercpuHashMapType:          false,
					HavePercpuArrayMapType:         false,
					HaveStackTraceMapType:          false,
					HaveCgroupArrayMapType:         false,
					HaveLruHashMapType:             false,
					HaveLruPercpuHashMapType:       false,
					HaveLpmTrieMapType:             false,
					HaveArrayOfMapsMapType:         false,
					HaveHashOfMapsMapType:          false,
					HaveDevmapMapType:              false,
					HaveSockmapMapType:             false,
					HaveCpumapMapType:              false,
					HaveXskmapMapType:              false,
					HaveSockhashMapType:            false,
					HaveCgroupStorageMapType:       false,
					HaveReuseportSockarrayMapType:  false,
					HavePercpuCgroupStorageMapType: false,
					HaveQueueMapType:               false,
					HaveStackMapType:               false,
				}),
		)
	})

	When("ValidateRequiredSystemConfig method is called", func() {
		JustBeforeEach(func() {
			// Action
			err = configInformer.ValidateRequiredSystemConfig()
		})

		Describe("success cases", func() {
			Context("with a valid system config", func() {
				BeforeEach(func() {
					// Arrange
					bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, validKernelConfig, nil)
				})

				It("should not return an error", func() {
					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Describe("error cases", func() {
			Describe("when the kernel configuration is not available", func() {
				BeforeEach(func() {
					// Arrange
					unameFuncMock = func() (unix.Utsname, error) {
						return unix.Utsname{}, fmt.Errorf("an error happened")
					}
				})

				It("return an error", func() {
					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("kernel config file not found"))
				})
			})

			DescribeTable("invalid system config", func(invalidSystemConfig string, invalidKernelParameters []ebpf.KernelParam) {
				//	Arrange
				unameFuncMock := func() (unix.Utsname, error) {
					return unix.Utsname{}, nil
				}

				statFSMock := mocks.NewStatFSMock(GinkgoT())
				statFSMock.EXPECT().Stat(mock.Anything).Return(nil, nil).Once()

				bpftoolExecutorMock := ebpf.NewExecutorMock(GinkgoT())
				bpftoolExecutorMock.EXPECT().Run([]string{"-j", "feature", "probe"}).Return(0, invalidSystemConfig, nil).Once()

				configInformer, err := ebpf.NewConfigInformer(log, false, bpftoolExecutorMock, statFSMock, unameFuncMock)
				Expect(err).ShouldNot(HaveOccurred())

				// Action
				err = configInformer.ValidateRequiredSystemConfig()

				// Assert
				Expect(err).To(HaveOccurred())
				for _, invalidKernelParam := range invalidKernelParameters {
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s kernel parameter is required (needed for: Essential eBPF infrastructure)", invalidKernelParam)))
				}
			},
				Entry("with a CONFIG_NET_CLS_ACT disabled",
					`
			{
			"system_config": {
				"CONFIG_BPF": "y",
				"CONFIG_BPF_SYSCALL": "y",
				"CONFIG_HAVE_EBPF_JIT": "y",
				"CONFIG_BPF_JIT": "y",
				"CONFIG_BPF_KPROBE_OVERRIDE": "y",
				"CONFIG_NET_CLS_ACT": "n"
			}
			}
			`,
					[]ebpf.KernelParam{
						"CONFIG_NET_CLS_ACT",
					},
				),
				Entry("with all kernel params disabled",
					`
			{
			"system_config": {
				"CONFIG_BPF": "n",
				"CONFIG_BPF_SYSCALL": "n",
				"CONFIG_HAVE_EBPF_JIT": "n",
				"CONFIG_BPF_JIT": "n",
				"CONFIG_BPF_KPROBE_OVERRIDE": "n",
				"CONFIG_NET_CLS_ACT": "n"
			}
			}
			`,
					[]ebpf.KernelParam{
						"CONFIG_BPF",
						"CONFIG_BPF_SYSCALL",
						"CONFIG_HAVE_EBPF_JIT",
						"CONFIG_BPF_JIT",
						"CONFIG_BPF_KPROBE_OVERRIDE",
						"CONFIG_NET_CLS_ACT",
					},
				),
			)

		})
	})

	When("IsKernelConfigAvailable method is called", func() {
		var (
			unameAssertCalledCount  int
			isKernelConfigAvailable bool
			kernelReleaseVersion    = "5.15.0-86-generic"
		)

		BeforeEach(func() {
			// Arrange
			unameAssertCalledCount = 0
			unameFuncMock = func() (unix.Utsname, error) {
				unameAssertCalledCount++
				uname := unix.Utsname{}
				copy(uname.Release[:], kernelReleaseVersion)

				return uname, nil
			}
		})

		JustBeforeEach(func() {
			// Action
			isKernelConfigAvailable = configInformer.IsKernelConfigAvailable()
		})

		Describe("success cases", func() {
			Context("when the /boot/config-x.x.x-x-generic file exists", func() {
				BeforeEach(func() {
					// Arrange
					By("checking if the boot config file is present")
					statFSMock = mocks.NewStatFSMock(GinkgoT())
					statFSMock.EXPECT().Stat(fmt.Sprintf("/boot/config-%s", kernelReleaseVersion)).Return(nil, nil).Once()
				})

				It("should succeed", func() {
					// Assert
					By("return true")
					Expect(isKernelConfigAvailable).To(BeTrue())

					By("getting the unix uname to extract the release version")
					Expect(unameAssertCalledCount).To(Equal(1))
				})
			})

			Context("when the /boot/config-5.15.0-86-generic does not exists but the /proc/config.gz exists", func() {
				BeforeEach(func() {
					// Arrange
					By("checking if both boot config files are present")
					statFSMock = mocks.NewStatFSMock(GinkgoT())
					statFSMock.EXPECT().Stat(fmt.Sprintf("/boot/config-%s", kernelReleaseVersion)).Return(nil, fmt.Errorf("the file does not exists")).Once()
					statFSMock.EXPECT().Stat("/proc/config.gz").Return(nil, nil).Once()
				})

				It("should succeed", func() {
					// Assert
					Expect(isKernelConfigAvailable).To(BeTrue())

					By("getting the unix uname to extract the release version")
					Expect(unameAssertCalledCount).To(Equal(1))
				})
			})
		})

		Describe("error cases", func() {
			Context("when the uname function return an error", func() {
				BeforeEach(func() {
					// Arrange
					unameFuncMock = func() (unix.Utsname, error) {
						unameAssertCalledCount++
						return unix.Utsname{}, fmt.Errorf("could not get the uname")
					}
				})

				It("should not succeed", func() {
					// Assert
					Expect(isKernelConfigAvailable).To(BeFalse())

					By("getting the unix uname to extract the release version")
					Expect(unameAssertCalledCount).To(Equal(1))

					By("not checking the existence of the boot config files")
					statFSMock.AssertNotCalled(GinkgoT(), "Stat", mock.Anything)
				})
			})

			Context("when both boot config files does not exist", func() {
				BeforeEach(func() {
					// Arrange
					By("checking if both boot config files are present")
					statFSMock = mocks.NewStatFSMock(GinkgoT())
					statFSMock.EXPECT().Stat(fmt.Sprintf("/boot/config-%s", kernelReleaseVersion)).Return(nil, fmt.Errorf("the file does not exists")).Once()
					statFSMock.EXPECT().Stat("/proc/config.gz").Return(nil, fmt.Errorf("the file does not exists")).Once()
				})

				It("should succeed", func() {
					// Assert
					Expect(isKernelConfigAvailable).To(BeFalse())

					By("getting the unix uname to extract the release version")
					Expect(unameAssertCalledCount).To(Equal(1))
				})
			})
		})
	})

})

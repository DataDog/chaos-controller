// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"fmt"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNSDisruptionInjector", func() {
	var (
		spec             v1beta1.DNSDisruptionSpec
		config           injector.DNSDisruptionInjectorConfig
		dnsInjector      injector.Injector
		iptablesMock     *network.IPTablesMock
		containerMock    *container.ContainerMock
		cgroupMock       *cgroup.ManagerMock
		netnsMock        *netns.ManagerMock
		dnsResponderMock *network.DNSResponderMock
		logger           *zap.SugaredLogger
	)

	BeforeEach(func() {
		// Reset the global DNS server sync.Once before each test
		injector.ResetDNSServerOnce()

		// Set required environment variable
		GinkgoT().Setenv("CHAOS_POD_IP", "10.244.0.10")

		// Create a test logger
		zapLogger, err := zap.NewDevelopment()
		Expect(err).NotTo(HaveOccurred())
		logger = zapLogger.Sugar()

		// Create mocks
		iptablesMock = network.NewIPTablesMock(GinkgoT())
		containerMock = container.NewContainerMock(GinkgoT())
		cgroupMock = cgroup.NewManagerMock(GinkgoT())
		netnsMock = netns.NewManagerMock(GinkgoT())
		dnsResponderMock = network.NewDNSResponderMock(GinkgoT())

		// Setup DNS responder mock with default expectations
		dnsResponderMock.EXPECT().Start().Return(nil).Maybe()
		dnsResponderMock.EXPECT().Stop().Return(nil).Maybe()

		// Setup container mock
		containerMock.EXPECT().Name().Return("test-container").Maybe()
		containerMock.EXPECT().PID().Return(1234).Maybe()

		// Setup cgroup mock
		cgroupMock.EXPECT().RelativePath("").Return("/kubepods/pod-abc123/container-xyz789").Maybe()

		// Setup netns mock
		netnsMock.EXPECT().Enter().Return(nil).Maybe()
		netnsMock.EXPECT().Exit().Return(nil).Maybe()

		// Default spec with valid records
		spec = v1beta1.DNSDisruptionSpec{
			Records: []v1beta1.DNSRecord{
				{
					Hostname: "example.com",
					Record: v1beta1.DNSRecordConfig{
						Type:  "A",
						Value: "192.168.1.1",
						TTL:   30,
					},
				},
			},
		}

		// Default config
		config = injector.DNSDisruptionInjectorConfig{
			Config: injector.Config{
				Log:             logger,
				TargetContainer: containerMock,
				Cgroup:          cgroupMock,
				Netns:           netnsMock,
				Disruption: chaosapi.DisruptionArgs{
					Level:       chaostypes.DisruptionLevelPod,
					TargetPodIP: "10.0.0.1",
				},
				InjectorCtx: nil, // Use nil context to avoid spec context cancellation issues
			},
			IPTables:  iptablesMock,
			Responder: dnsResponderMock, // Use mock responder for testing
		}
	})

	AfterEach(func() {
		// Reset the injector
		dnsInjector = nil
	})

	Describe("NewDNSDisruptionInjector", func() {
		It("should create a new DNS disruption injector", func() {
			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)
			Expect(dnsInjector).NotTo(BeNil())
		})
	})

	Describe("GetDisruptionKind", func() {
		BeforeEach(func() {
			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)
		})

		It("should return the correct disruption kind", func() {
			kind := dnsInjector.GetDisruptionKind()
			Expect(string(kind)).To(Equal(chaostypes.DisruptionKindDNSDisruption))
		})
	})

	Describe("TargetName", func() {
		BeforeEach(func() {
			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)
		})

		It("should return the container name", func() {
			name := dnsInjector.TargetName()
			Expect(name).To(Equal("test-container"))
		})
	})

	Describe("Inject", func() {
		Context("Success cases", func() {
			It("should inject DNS disruption with default settings", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "").Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "53", "10.244.0.10:5354").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "53", "", "", "").Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with custom port", func() {
				spec.Port = 8053

				// RedirectTo uses responder ports (UDP: 5353, TCP: 5354), Intercept uses the custom DNS port (8053)
				iptablesMock.EXPECT().RedirectTo("udp", "8053", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "8053", "", "", "").Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "8053", "10.244.0.10:5354").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "8053", "", "", "").Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with UDP protocol only", func() {
				spec.Protocol = "udp"

				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "").Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with TCP protocol only", func() {
				spec.Protocol = "tcp"

				iptablesMock.EXPECT().RedirectTo("tcp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "53", "", "", "").Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with multiple records", func() {
				spec.Records = []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "192.168.1.1",
							TTL:   30,
						},
					},
					{
						Hostname: "test.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "192.168.1.2",
							TTL:   30,
						},
					},
					{
						Hostname: "api.example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "192.168.1.3",
							TTL:   30,
						},
					},
				}

				iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()
				iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with DROP special value", func() {
				spec.Records = []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "DROP",
							TTL:   30,
						},
					},
				}

				iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()
				iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with SERVFAIL special value", func() {
				spec.Records = []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "SERVFAIL",
							TTL:   30,
						},
					},
				}

				iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()
				iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with RANDOM special value", func() {
				spec.Records = []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record: v1beta1.DNSRecordConfig{
							Type:  "A",
							Value: "RANDOM",
							TTL:   30,
						},
					},
				}

				iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()
				iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Twice()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Error cases", func() {
			It("should return error when UDP redirect fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(fmt.Errorf("redirect failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for udp redirection"))
			})

			It("should return error when UDP intercept fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "").Return(fmt.Errorf("intercept failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for udp interception"))
			})

			It("should return error when TCP redirect fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "").Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "53", "10.244.0.10:5354").Return(fmt.Errorf("redirect failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for tcp redirection"))
			})

			It("should return error when TCP intercept fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "").Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "53", "10.244.0.10:5354").Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "53", "", "", "").Return(fmt.Errorf("intercept failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for tcp interception"))
			})
		})

		Context("Dry run mode", func() {
			BeforeEach(func() {
				config.Disruption.DryRun = true
			})

			It("should not start DNS responder in dry run mode", func() {
				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
				iptablesMock.AssertNotCalled(GinkgoT(), "RedirectTo")
				iptablesMock.AssertNotCalled(GinkgoT(), "Intercept")
			})

			It("should log what would be disrupted in dry run mode", func() {
				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
			})

			It("should handle multiple records in dry run mode", func() {
				spec.Records = []v1beta1.DNSRecord{
					{
						Hostname: "example.com",
						Record:   v1beta1.DNSRecordConfig{Type: "A", Value: "192.168.1.1", TTL: 30},
					},
					{
						Hostname: "test.com",
						Record:   v1beta1.DNSRecordConfig{Type: "AAAA", Value: "2001:db8::1", TTL: 60},
					},
				}

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
			})

			It("should handle Clean in dry run mode", func() {
				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				err = dnsInjector.Clean()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Stop")
				iptablesMock.AssertNotCalled(GinkgoT(), "Clear")
			})

			It("should work with custom port in dry run mode", func() {
				spec.Port = 8053

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
			})

			It("should work with UDP protocol only in dry run mode", func() {
				spec.Protocol = "udp"

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
			})

			It("should work with TCP protocol only in dry run mode", func() {
				spec.Protocol = "tcp"

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())

				dnsResponderMock.AssertNotCalled(GinkgoT(), "Start")
			})
		})
	})

	Describe("Clean", func() {
		BeforeEach(func() {
			iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

			err := dnsInjector.Inject()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Success cases", func() {
			It("should clean up successfully", func() {
				iptablesMock.EXPECT().Clear().Return(nil).Once()

				err := dnsInjector.Clean()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Error cases", func() {
			It("should return error when IPTables clear fails", func() {
				iptablesMock.EXPECT().Clear().Return(fmt.Errorf("clear failed")).Once()

				err := dnsInjector.Clean()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to clear IPTables rules"))
			})
		})
	})

	Describe("UpdateConfig", func() {
		BeforeEach(func() {
			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)
		})

		It("should update the config", func() {
			newConfig := injector.Config{
				Log: logger,
			}

			Expect(func() {
				dnsInjector.UpdateConfig(newConfig)
			}).NotTo(Panic())
		})
	})
})

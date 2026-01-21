// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"context"
	"fmt"

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
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("DNSDisruptionInjector", func() {
	var (
		spec          v1beta1.DNSDisruptionSpec
		config        injector.DNSDisruptionInjectorConfig
		dnsInjector   injector.Injector
		iptablesMock  *network.IPTablesMock
		containerMock *container.ContainerMock
		cgroupMock    *cgroup.ManagerMock
		netnsMock     *netns.ManagerMock
		logger        *zap.SugaredLogger
		ctx           context.Context
		cancel        context.CancelFunc
	)

	BeforeEach(func() {
		// Create a test logger
		zapLogger, err := zap.NewDevelopment()
		Expect(err).NotTo(HaveOccurred())
		logger = zapLogger.Sugar()

		// Create context
		ctx, cancel = context.WithCancel(context.Background())

		// Create mocks
		iptablesMock = network.NewIPTablesMock(GinkgoT())
		containerMock = container.NewContainerMock(GinkgoT())
		cgroupMock = cgroup.NewManagerMock(GinkgoT())
		netnsMock = netns.NewManagerMock(GinkgoT())

		// Setup container mock
		containerMock.EXPECT().Name().Return("test-container").Maybe()

		// Setup cgroup mock
		cgroupMock.EXPECT().RelativePath("").Return("/kubepods/pod-abc123/container-xyz789").Maybe()

		// Setup netns mock
		netnsMock.EXPECT().Enter().Return(nil).Maybe()
		netnsMock.EXPECT().Exit().Return(nil).Maybe()

		// Default spec
		spec = v1beta1.DNSDisruptionSpec{
			Domains:     []string{"example.com"},
			FailureMode: "nxdomain",
		}

		// Default config
		config = injector.DNSDisruptionInjectorConfig{
			Config: injector.Config{
				Log:             logger,
				TargetContainer: containerMock,
				Cgroup:          cgroupMock,
				Netns:           netnsMock,
				Disruption: chaosapi.DisruptionArgs{
					TargetPodIP: "10.0.0.1",
				},
				InjectorCtx: ctx,
			},
			IPTables: iptablesMock,
		}
	})

	AfterEach(func() {
		cancel()
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
			Expect(kind).To(Equal(chaostypes.DisruptionKindDNSDisruption))
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
			BeforeEach(func() {
				// Mock IPTables calls to succeed (RedirectTo must be called before Intercept)
				iptablesMock.EXPECT().RedirectTo("udp", "5353", "127.0.0.1").Return(nil).Maybe()
				iptablesMock.EXPECT().Intercept("udp", "53", "", "", "10.0.0.1").Return(nil).Maybe()
				iptablesMock.EXPECT().RedirectTo("tcp", "5353", "127.0.0.1").Return(nil).Maybe()
				iptablesMock.EXPECT().Intercept("tcp", "53", "/kubepods/pod-abc123/container-xyz789", "", "10.0.0.1").Return(nil).Maybe()
			})

			It("should inject DNS disruption with default settings", func() {
				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with custom port", func() {
				spec.Port = 8053

				// RedirectTo always uses responder port (5353), Intercept uses the custom DNS port (8053)
				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(nil).Maybe()
				iptablesMock.EXPECT().Intercept("udp", "8053", mock.Anything, "", mock.Anything).Return(nil).Maybe()
				iptablesMock.EXPECT().RedirectTo("tcp", "5353", mock.Anything).Return(nil).Maybe()
				iptablesMock.EXPECT().Intercept("tcp", "8053", mock.Anything, "", mock.Anything).Return(nil).Maybe()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with UDP protocol only", func() {
				spec.Protocol = "udp"

				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", mock.Anything, "", mock.Anything).Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with TCP protocol only", func() {
				spec.Protocol = "tcp"

				iptablesMock.EXPECT().RedirectTo("tcp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "53", mock.Anything, "", mock.Anything).Return(nil).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with multiple domains", func() {
				spec.Domains = []string{"example.com", "test.com", "api.example.com"}

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with drop failure mode", func() {
				spec.FailureMode = "drop"

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with servfail failure mode", func() {
				spec.FailureMode = "servfail"

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should inject DNS disruption with random-ip failure mode", func() {
				spec.FailureMode = "random-ip"

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				// Cancel context immediately so Inject returns
				cancel()

				err := dnsInjector.Inject()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Error cases", func() {
			It("should return error when UDP redirect fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(fmt.Errorf("redirect failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for UDP redirection"))
			})

			It("should return error when UDP intercept fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", mock.Anything, "", mock.Anything).Return(fmt.Errorf("intercept failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for UDP interception"))
			})

			It("should return error when TCP redirect fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", mock.Anything, "", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "5353", mock.Anything).Return(fmt.Errorf("redirect failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for TCP redirection"))
			})

			It("should return error when TCP intercept fails", func() {
				iptablesMock.EXPECT().RedirectTo("udp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("udp", "53", mock.Anything, "", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().RedirectTo("tcp", "5353", mock.Anything).Return(nil).Once()
				iptablesMock.EXPECT().Intercept("tcp", "53", mock.Anything, "", mock.Anything).Return(fmt.Errorf("intercept failed")).Once()

				dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

				err := dnsInjector.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to configure IPTables for TCP interception"))
			})
		})
	})

	Describe("Clean", func() {
		BeforeEach(func() {
			// Mock IPTables calls for injection
			iptablesMock.EXPECT().Intercept(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			iptablesMock.EXPECT().RedirectTo(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

			dnsInjector = injector.NewDNSDisruptionInjector(spec, config)

			// Inject first
			cancel()
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

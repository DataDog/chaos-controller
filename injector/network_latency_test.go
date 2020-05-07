// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
)

var _ = Describe("Tc", func() {
	var (
		ctn                                  fakeContainer
		inj                                  Injector
		config                               NetworkLatencyInjectorConfig
		spec                                 v1beta1.NetworkLatencySpec
		tc                                   fakeTc
		tcIsQdiscClearedCall                 *mock.Call
		nl                                   fakeNetlinkAdapter
		nllink1, nllink2                     *fakeNetlinkLink
		nllink1TxQlenCall, nllink2TxQlenCall *mock.Call
		nlroute1, nlroute2                   *fakeNetlinkRoute
	)

	BeforeEach(func() {
		// container
		ctn = fakeContainer{}
		ctn.On("EnterNetworkNamespace").Return(nil)
		ctn.On("ExitNetworkNamespace").Return(nil)

		// tc
		tc = fakeTc{}
		tc.On("AddDelay", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("ClearQdisc", mock.Anything).Return(nil)
		tcIsQdiscClearedCall = tc.On("IsQdiscCleared", mock.Anything).Return(false, nil)

		// netlink
		nllink1 = &fakeNetlinkLink{}
		nllink1.On("Name").Return("lo")
		nllink1.On("SetTxQLen", mock.Anything).Return(nil)
		nllink1TxQlenCall = nllink1.On("TxQLen").Return(0)
		nllink2 = &fakeNetlinkLink{}
		nllink2.On("Name").Return("eth0")
		nllink2.On("SetTxQLen", mock.Anything).Return(nil)
		nllink2TxQlenCall = nllink2.On("TxQLen").Return(0)

		nlroute1 = &fakeNetlinkRoute{}
		nlroute1.On("Link").Return(nllink1)
		nlroute2 = &fakeNetlinkRoute{}
		nlroute2.On("Link").Return(nllink2)

		nl = fakeNetlinkAdapter{}
		nl.On("LinkList").Return([]network.NetlinkLink{nllink1, nllink2}, nil)
		nl.On("LinkByIndex", 0).Return(nllink1, nil)
		nl.On("LinkByIndex", 1).Return(nllink2, nil)
		nl.On("LinkByName", "lo").Return(nllink1, nil)
		nl.On("LinkByName", "eth0").Return(nllink2, nil)
		nl.On("RoutesForIP", mock.Anything).Return([]network.NetlinkRoute{nlroute1, nlroute2}, nil)

		spec = v1beta1.NetworkLatencySpec{
			Delay: 1000,
		}
		config = NetworkLatencyInjectorConfig{
			TrafficController: &tc,
			NetlinkAdapter:    &nl,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkLatencyInjectorWithConfig("fake", spec, &ctn, log, ms, config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		Context("with no host specified", func() {
			It("should enter and exit the container network namespace", func() {
				Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
				Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
			})
			It("should not set or clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
			It("should add delay to the interfaces root qdisc", func() {
				tc.AssertCalled(GinkgoT(), "AddDelay", "lo", "root", mock.Anything, time.Second)
				tc.AssertCalled(GinkgoT(), "AddDelay", "eth0", "root", mock.Anything, time.Second)
			})
		})

		Context("with multiple hosts specified and interface without qlen", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
				spec.Port = 80
			})

			It("should set and clear the interface qlen", func() {
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})
			It("should create a prio qdisc on both interfaces", func() {
				tc.AssertCalled(GinkgoT(), "AddPrio", "lo", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
			})
			It("should add latency on both interfaces prio qdisc", func() {
				tc.AssertCalled(GinkgoT(), "AddDelay", "lo", "1:4", mock.Anything, time.Second)
				tc.AssertCalled(GinkgoT(), "AddDelay", "eth0", "1:4", mock.Anything, time.Second)
			})
			It("should add a filter to redirect traffic on delayed band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "1.1.1.1/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "2.2.2.2/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "1.1.1.1/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "2.2.2.2/32", 80, "1:4")
			})
		})

		Context("with multiple hosts specified and interfaces with qlen", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
			})
			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		Describe("inj.Clean", func() {
			JustBeforeEach(func() {
				inj.Clean()
			})

			Context("with a non-cleared qdisc", func() {
				It("should enter and exit the container network namespace", func() {
					Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
					Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
				})
				It("should clear the interfaces qdisc", func() {
					tc.AssertCalled(GinkgoT(), "ClearQdisc", "lo")
					tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth0")
				})
			})

			Context("with an already cleared qdisc", func() {
				BeforeEach(func() {
					tcIsQdiscClearedCall.Return(true, nil)
				})
				It("should enter and exit the container network namespace", func() {
					Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
					Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
				})
				It("should not clear the interfaces qdisc", func() {
					tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "lo")
					tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "eth0")
				})
			})
		})
	})
})

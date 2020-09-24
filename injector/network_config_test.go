// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"

	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
)

var _ = Describe("Tc", func() {
	var (
		config                               NetworkDisruptionConfig
		tc                                   network.TcMock
		tcIsQdiscClearedCall                 *mock.Call
		nl                                   network.NetlinkAdapterMock
		nllink1, nllink2                     *network.NetlinkLinkMock
		nllink1TxQlenCall, nllink2TxQlenCall *mock.Call
		nlroute1, nlroute2                   *network.NetlinkRouteMock
		hosts                                []string
		port                                 int
		protocol                             string
		flow                                 string
		delay                                time.Duration
		drop                                 int
		corrupt                              int
		bandwidthLimit                       uint
	)

	BeforeEach(func() {
		// tc
		tc = network.TcMock{}
		tc.On("AddNetem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("ClearQdisc", mock.Anything).Return(nil)
		tcIsQdiscClearedCall = tc.On("IsQdiscCleared", mock.Anything).Return(false, nil)

		// netlink
		nllink1 = &network.NetlinkLinkMock{}
		nllink1.On("Name").Return("lo")
		nllink1.On("SetTxQLen", mock.Anything).Return(nil)
		nllink1TxQlenCall = nllink1.On("TxQLen").Return(0)
		nllink2 = &network.NetlinkLinkMock{}
		nllink2.On("Name").Return("eth0")
		nllink2.On("SetTxQLen", mock.Anything).Return(nil)
		nllink2TxQlenCall = nllink2.On("TxQLen").Return(0)

		nlroute1 = &network.NetlinkRouteMock{}
		nlroute1.On("Link").Return(nllink1)
		nlroute2 = &network.NetlinkRouteMock{}
		nlroute2.On("Link").Return(nllink2)

		nl = network.NetlinkAdapterMock{}
		nl.On("LinkList").Return([]network.NetlinkLink{nllink1, nllink2}, nil)
		nl.On("LinkByIndex", 0).Return(nllink1, nil)
		nl.On("LinkByIndex", 1).Return(nllink2, nil)
		nl.On("LinkByName", "lo").Return(nllink1, nil)
		nl.On("LinkByName", "eth0").Return(nllink2, nil)
		nl.On("RoutesForIP", mock.Anything).Return([]network.NetlinkRoute{nlroute1, nlroute2}, nil)

		// netem parameters
		hosts = []string{}
		port = 80
		protocol = "tcp"
		flow = "egress"
		delay = time.Second
		drop = 5
		corrupt = 10
		bandwidthLimit = 100
	})

	JustBeforeEach(func() {
		config = NewNetworkDisruptionConfig(log, &tc, &nl, nil, hosts, port, protocol, flow)
	})

	Describe("AddNetem", func() {
		JustBeforeEach(func() {
			config.AddNetem(delay, drop, corrupt)
			config.AddOutputLimit(bandwidthLimit)
			config.ApplyOperations()
		})

		Context("with no host, port or protocol specified", func() {
			BeforeEach(func() {
				port = 0
				protocol = ""
			})

			It("should not set or clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})

			It("should apply disruptions to the interfaces root qdisc", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", "lo", "root", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "root", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "lo", "1:", mock.Anything, bandwidthLimit)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "1:", mock.Anything, bandwidthLimit)
			})
		})

		Context("with multiple hosts specified and interface without qlen", func() {
			BeforeEach(func() {
				hosts = []string{"1.1.1.1", "2.2.2.2"}
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

			It("should add a filter to redirect traffic on delayed band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "1.1.1.1/32", port, protocol, "1:4", flow)
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "2.2.2.2/32", port, protocol, "1:4", flow)
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "1.1.1.1/32", port, protocol, "1:4", flow)
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "2.2.2.2/32", port, protocol, "1:4", flow)
			})

			It("should add delay to the interfaces parent qdisc", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", "lo", "1:4", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "1:4", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "lo", "2:", mock.Anything, bandwidthLimit)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "2:", mock.Anything, bandwidthLimit)
			})
		})

		Context("with multiple hosts specified and interfaces with qlen", func() {
			BeforeEach(func() {
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
				hosts = []string{"1.1.1.1", "2.2.2.2"}
			})

			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})
	})

	Describe("ClearAllQdiscs", func() {
		JustBeforeEach(func() {
			config.ClearOperations()
		})

		Context("with a non-cleared qdisc", func() {
			It("should clear the interfaces qdisc", func() {
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "lo")
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth0")
			})
		})

		Context("with an already cleared qdisc", func() {
			BeforeEach(func() {
				tcIsQdiscClearedCall.Return(true, nil)
			})

			It("should not clear the interfaces qdisc", func() {
				tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "lo")
				tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "eth0")
			})
		})
	})
})

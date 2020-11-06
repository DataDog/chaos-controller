// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("Tc", func() {
	var (
		config                                                  NetworkDisruptionConfig
		tc                                                      network.TcMock
		tcIsQdiscClearedCall                                    *mock.Call
		nl                                                      network.NetlinkAdapterMock
		nllink1, nllink2, nllink3                               *network.NetlinkLinkMock
		nllink1TxQlenCall, nllink2TxQlenCall, nllink3TxQlenCall *mock.Call
		nlroute1, nlroute2, nlroute3                            *network.NetlinkRouteMock
		hosts                                                   []string
		port                                                    int
		protocol                                                string
		flow                                                    string
		delay                                                   time.Duration
		drop                                                    int
		corrupt                                                 int
		bandwidthLimit                                          uint
		level                                                   chaostypes.DisruptionLevel
		dns                                                     network.DNSMock
	)

	BeforeEach(func() {
		// tc
		tc = network.TcMock{}
		tc.On("AddNetem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddCgroupFilter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
		nllink3 = &network.NetlinkLinkMock{}
		nllink3.On("Name").Return("eth1")
		nllink3.On("SetTxQLen", mock.Anything).Return(nil)
		nllink3TxQlenCall = nllink3.On("TxQLen").Return(0)

		nlroute1 = &network.NetlinkRouteMock{}
		nlroute1.On("Link").Return(nllink1)
		nlroute1.On("Gateway").Return(net.IP([]byte{}))
		nlroute2 = &network.NetlinkRouteMock{}
		nlroute2.On("Link").Return(nllink2)
		nlroute2.On("Gateway").Return(net.ParseIP("192.168.0.1"))
		nlroute3 = &network.NetlinkRouteMock{}
		nlroute3.On("Link").Return(nllink3)
		nlroute3.On("Gateway").Return(net.ParseIP("192.168.1.1"))

		nl = network.NetlinkAdapterMock{}
		nl.On("LinkList").Return([]network.NetlinkLink{nllink1, nllink2, nllink3}, nil)
		nl.On("LinkByIndex", 0).Return(nllink1, nil)
		nl.On("LinkByIndex", 1).Return(nllink2, nil)
		nl.On("LinkByIndex", 2).Return(nllink3, nil)
		nl.On("LinkByName", "lo").Return(nllink1, nil)
		nl.On("LinkByName", "eth0").Return(nllink2, nil)
		nl.On("LinkByName", "eth1").Return(nllink3, nil)
		nl.On("RoutesForIP", "10.0.0.1/32").Return([]network.NetlinkRoute{nlroute2}, nil)      // node IP route going through eth0
		nl.On("RoutesForIP", "1.1.1.1/32").Return([]network.NetlinkRoute{nlroute2}, nil)       // random external route going through eth0
		nl.On("RoutesForIP", "2.2.2.2/32").Return([]network.NetlinkRoute{nlroute3}, nil)       // random external route going through eth1
		nl.On("RoutesForIP", "192.168.0.254/32").Return([]network.NetlinkRoute{nlroute3}, nil) // apiserver route going through eth1
		nl.On("DefaultRoute").Return(nlroute2, nil)

		// dns
		dns = network.DNSMock{}
		dns.On("Resolve", "kubernetes.default").Return([]net.IP{net.ParseIP("192.168.0.254")}, nil)

		// netem parameters
		hosts = []string{}
		port = 80
		protocol = "tcp"
		flow = "egress"
		delay = time.Second
		drop = 5
		corrupt = 10
		bandwidthLimit = 100
		level = chaostypes.DisruptionLevelPod

		// environment variables
		Expect(os.Setenv(chaostypes.TargetPodHostIPEnv, "10.0.0.1")).To(BeNil())
	})

	JustBeforeEach(func() {
		config = NewNetworkDisruptionConfig(log, &tc, &nl, &dns, level, hosts, port, protocol, flow)
	})

	Describe("Injecting disruptions", func() {
		JustBeforeEach(func() {
			config.AddNetem(delay, drop, corrupt)
			config.AddOutputLimit(bandwidthLimit)
			Expect(config.ApplyOperations()).To(BeNil())
		})

		Context("with no host, port or protocol specified", func() {
			BeforeEach(func() {
				port = 0
				protocol = ""
			})

			It("should set or clear the interface qlen on all interfaces excluding lo", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})

			It("should not apply anything to lo interface", func() {
				tc.AssertNotCalled(GinkgoT(), "AddNetem", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddPrio", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddFilter", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddOutputLimit", "lo", mock.Anything, mock.Anything, mock.Anything)
			})

			It("should create 2 prio qdiscs on main interfaces", func() {
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "1:4", uint32(2), uint32(2), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "1:4", uint32(2), uint32(2), mock.Anything)
			})

			It("should add a filter to redirect all traffic on main interfaces on the disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, port, protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, port, protocol, "1:4")
			})

			It("should add a cgroup filter to classify packets according to their classid", func() {
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth0", "2:0", mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth1", "2:0", mock.Anything)
			})

			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "192.168.0.1/32", 0, 0, "", "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "10.0.0.1/32", 0, 0, "", "1:1")
			})

			It("should apply disruptions to main interfaces 4th band", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "3:", mock.Anything, bandwidthLimit)
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth1", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth1", "3:", mock.Anything, bandwidthLimit)
			})
		})

		Context("with ingress flow", func() {
			BeforeEach(func() {
				flow = "ingress"
			})

			It("should set or clear the interface qlen on all interfaces excluding lo", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})

			It("should not apply anything to lo interface", func() {
				tc.AssertNotCalled(GinkgoT(), "AddNetem", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddPrio", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddFilter", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddOutputLimit", "lo", mock.Anything, mock.Anything, mock.Anything)
			})

			It("should create 2 prio qdiscs on main interfaces", func() {
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "1:4", uint32(2), uint32(2), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "1:4", uint32(2), uint32(2), mock.Anything)
			})

			It("should add a filter to redirect all traffic on main interfaces on the disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "0.0.0.0/0", port, 0, protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "0.0.0.0/0", port, 0, protocol, "1:4")
			})

			It("should add a cgroup filter to classify packets according to their classid", func() {
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth0", "2:0", mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth1", "2:0", mock.Anything)
			})

			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "192.168.0.1/32", 0, 0, "", "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "10.0.0.1/32", 0, 0, "", "1:1")
			})

			It("should apply disruptions to main interfaces 4th band", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "3:", mock.Anything, bandwidthLimit)
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth1", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth1", "3:", mock.Anything, bandwidthLimit)
			})
		})

		Context("with multiple hosts specified and interface without qlen using egress flow", func() {
			BeforeEach(func() {
				hosts = []string{"1.1.1.1", "2.2.2.2"}
			})

			It("should set or clear the interface qlen on all interfaces excluding lo", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})

			It("should not apply anything to lo interface", func() {
				tc.AssertNotCalled(GinkgoT(), "AddNetem", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddPrio", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddFilter", "lo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "AddOutputLimit", "lo", mock.Anything, mock.Anything, mock.Anything)
			})

			It("should create 2 prio qdiscs on main interfaces", func() {
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "1:4", uint32(2), uint32(2), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "1:4", uint32(2), uint32(2), mock.Anything)
			})

			It("should add a filter to redirect targeted traffic on related interfaces on the disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "1.1.1.1/32", 0, port, protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "2.2.2.2/32", 0, port, protocol, "1:4")
			})

			It("should add a cgroup filter to classify packets according to their classid", func() {
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth0", "2:0", mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth1", "2:0", mock.Anything)
			})

			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "192.168.0.1/32", 0, 0, "", "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "10.0.0.1/32", 0, 0, "", "1:1")
			})

			It("should apply disruptions to main interfaces 4th band", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "3:", mock.Anything, bandwidthLimit)
				tc.AssertCalled(GinkgoT(), "AddNetem", "eth1", "2:2", mock.Anything, delay, drop, corrupt)
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth1", "3:", mock.Anything, bandwidthLimit)
			})
		})

		Context("with multiple hosts specified and interfaces with qlen", func() {
			BeforeEach(func() {
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
				nllink3TxQlenCall.Return(1000)
				hosts = []string{"1.1.1.1", "2.2.2.2"}
			})

			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		Context("with node level injection", func() {
			BeforeEach(func() {
				level = chaostypes.DisruptionLevelNode
			})

			It("should create only one prio qdisc on main interfaces", func() {
				tc.AssertNumberOfCalls(GinkgoT(), "AddPrio", 2)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
				tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "root", uint32(1), uint32(4), mock.Anything)
			})

			It("should add safeguard filters allowing SSH, ARP and apiserver communications", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "nil", 22, 0, "tcp", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "nil", 0, 0, "arp", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "192.168.0.254/32", 0, 0, "", "1:1")
			})
		})
	})

	Describe("Clearing all disruptions", func() {
		JustBeforeEach(func() {
			config.ClearOperations()
		})

		Context("with a non-cleared qdisc", func() {
			It("should clear the interfaces qdisc", func() {
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth0")
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth1")
			})
		})

		Context("with an already cleared qdisc", func() {
			BeforeEach(func() {
				tcIsQdiscClearedCall.Return(true, nil)
			})

			It("should not clear the interfaces qdisc", func() {
				tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "lo")
				tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "eth0")
				tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "eth1")
			})
		})
	})
})

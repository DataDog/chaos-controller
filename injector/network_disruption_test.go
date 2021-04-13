// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector_test

import (
	"net"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("Failure", func() {
	var (
		ctn                                                     *container.ContainerMock
		inj                                                     Injector
		config                                                  NetworkDisruptionInjectorConfig
		spec                                                    v1beta1.NetworkDisruptionSpec
		cgroupManager                                           *cgroup.ManagerMock
		cgroupManagerExistsCall                                 *mock.Call
		tc                                                      *network.TcMock
		nl                                                      *network.NetlinkAdapterMock
		nllink1, nllink2, nllink3                               *network.NetlinkLinkMock
		nllink1TxQlenCall, nllink2TxQlenCall, nllink3TxQlenCall *mock.Call
		nlroute1, nlroute2, nlroute3                            *network.NetlinkRouteMock
		dns                                                     *network.DNSMock
		netnsManager                                            *netns.ManagerMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &cgroup.ManagerMock{}
		cgroupManagerExistsCall = cgroupManager.On("Exists", "net_cls").Return(true, nil)
		cgroupManager.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// netns
		netnsManager = &netns.ManagerMock{}
		netnsManager.On("Enter").Return(nil)
		netnsManager.On("Exit").Return(nil)

		// tc
		tc = &network.TcMock{}
		tc.On("AddNetem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddCgroupFilter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("ClearQdisc", mock.Anything).Return(nil)

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

		nl = &network.NetlinkAdapterMock{}
		nl.On("LinkList").Return([]network.NetlinkLink{nllink1, nllink2, nllink3}, nil)
		nl.On("LinkByIndex", 0).Return(nllink1, nil)
		nl.On("LinkByIndex", 1).Return(nllink2, nil)
		nl.On("LinkByIndex", 2).Return(nllink3, nil)
		nl.On("LinkByName", "lo").Return(nllink1, nil)
		nl.On("LinkByName", "eth0").Return(nllink2, nil)
		nl.On("LinkByName", "eth1").Return(nllink3, nil)
		nl.On("DefaultRoute").Return(nlroute2, nil)

		// dns
		dns = &network.DNSMock{}
		dns.On("Resolve", "kubernetes.default").Return([]net.IP{net.ParseIP("192.168.0.254")}, nil)
		dns.On("Resolve", "testhost").Return([]net.IP{net.ParseIP("1.1.1.1")}, nil)

		// container
		ctn = &container.ContainerMock{}

		// environment variables
		Expect(os.Setenv(env.InjectorTargetPodHostIP, "10.0.0.2")).To(BeNil())

		// config
		config = NetworkDisruptionInjectorConfig{
			Config: Config{
				Container:   ctn,
				Log:         log,
				MetricsSink: ms,
				Netns:       netnsManager,
				Cgroup:      cgroupManager,
				Level:       chaostypes.DisruptionLevelPod,
			},
			TrafficController: tc,
			NetlinkAdapter:    nl,
			DNSClient:         dns,
		}

		spec = v1beta1.NetworkDisruptionSpec{
			Hosts:          []string{},
			Port:           80,
			Protocol:       "tcp",
			Drop:           90,
			Duplicate:      80,
			Corrupt:        70,
			Delay:          1000,
			DelayJitter:    100,
			BandwidthLimit: 10000,
			Flow:           "egress",
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkDisruptionInjector(spec, config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		// general tests that should work for all contexts
		It("should enter and exit the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should write the custom classid to the target net_cls cgroup", func() {
			cgroupManager.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0x00020002")
		})

		It("should create 2 prio qdiscs on main interfaces", func() {
			tc.AssertCalled(GinkgoT(), "AddPrio", "lo", "root", uint32(1), uint32(4), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "root", uint32(1), uint32(4), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "root", uint32(1), uint32(4), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", "lo", "1:4", uint32(2), uint32(2), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", "eth0", "1:4", uint32(2), uint32(2), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", "eth1", "1:4", uint32(2), uint32(2), mock.Anything)
		})

		It("should add a cgroup filter to classify packets according to their classid", func() {
			tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth0", "2:0", mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddCgroupFilter", "eth1", "2:0", mock.Anything)
		})

		It("should apply disruptions to main interfaces 2nd band", func() {
			tc.AssertCalled(GinkgoT(), "AddNetem", "eth0", "2:2", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "3:", mock.Anything, uint(spec.BandwidthLimit))
			tc.AssertCalled(GinkgoT(), "AddNetem", "eth1", "2:2", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth1", "3:", mock.Anything, uint(spec.BandwidthLimit))
		})

		// qlen cases
		Context("with interfaces without a qlen value", func() {
			It("should set or clear the interface qlen on all interfaces", func() {
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})
		})

		Context("with interfaces with a qlen value", func() {
			BeforeEach(func() {
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
				nllink3TxQlenCall.Return(1000)
			})

			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		// hosts filtering cases
		Context("with no hosts specified", func() {
			It("should add a filter to redirect all traffic on main interfaces on the disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, spec.Port, spec.Protocol, "1:4")
			})
		})

		Context("with multiple hosts specified", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
			})

			It("should add a filter to redirect targeted traffic on all interfaces on the disrupted band filter on given hosts as destination IP", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "nil", "1.1.1.1/32", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "1.1.1.1/32", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "1.1.1.1/32", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "nil", "2.2.2.2/32", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "2.2.2.2/32", 0, spec.Port, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "2.2.2.2/32", 0, spec.Port, spec.Protocol, "1:4")
			})
		})

		// safeguards
		Context("pod level safeguards", func() {
			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "192.168.0.1/32", 0, 0, "", "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "10.0.0.2/32", 0, 0, "", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "10.0.0.2/32", 0, 0, "", "1:1")
			})
		})

		Context("node level safeguards", func() {
			BeforeEach(func() {
				config.Level = chaostypes.DisruptionLevelNode
			})

			It("should add a filter to redirect SSH traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "nil", 22, 0, "tcp", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "nil", 22, 0, "tcp", "1:1")
			})

			It("should add a filter to redirect ARP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "nil", 0, 0, "arp", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "nil", 0, 0, "arp", "1:1")
			})

			It("should add a filter to apiserver traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "192.168.0.254/32", 0, 0, "", "1:1")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "192.168.0.254/32", 0, 0, "", "1:1")
			})
		})

		Context("with ingress flow", func() {
			BeforeEach(func() {
				spec.Flow = "ingress"
			})

			It("should add a filter to redirect all traffic on main interfaces on the disrupted band with specified port as source port", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "nil", "0.0.0.0/0", spec.Port, 0, spec.Protocol, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth1", "1:0", mock.Anything, "nil", "0.0.0.0/0", spec.Port, 0, spec.Protocol, "1:4")
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

		Context("qdisc cleanup should happen", func() {
			It("should clear the interfaces qdisc", func() {
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "lo")
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth0")
				tc.AssertCalled(GinkgoT(), "ClearQdisc", "eth1")
			})
		})

		Context("with an existing net_cls cgroup", func() {
			It("should erase the classid value", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0x0")
			})
		})

		Context("with a non-existing net_cls cgroup", func() {
			BeforeEach(func() {
				cgroupManagerExistsCall.Return(false, nil)
			})

			It("should not try to erase the classid value", func() {
				cgroupManager.AssertNotCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", mock.Anything)
			})
		})
	})
})

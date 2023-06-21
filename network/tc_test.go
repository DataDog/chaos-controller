// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package network

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Tc", func() {
	var (
		tcRunner          tc
		tcExecuter        *tcExecuterMock
		tcExecuterRunCall *tcExecuterMock_Run_Call
		ifaces            []string
		parent            string
		handle            string
		delay             time.Duration
		delayJitter       time.Duration
		drop              int
		duplicate         int
		corrupt           int
		bands             uint32
		priomap           [16]uint32
		srcIP, dstIP      *net.IPNet
		srcPort, dstPort  int
		protocol          protocol
		connState         connState
		flowid            string
	)

	BeforeEach(func() {
		// fake command executer
		tcExecuter = newTcExecuterMock(GinkgoT())
		tcExecuterRunCall = tcExecuter.EXPECT().Run(mock.Anything).Return(0, "", nil)

		// tc runner
		tcRunner = tc{
			executer:         tcExecuter,
			tcFilterPriority: 1000,
		}

		// injected variables
		ifaces = []string{"lo", "eth0"}
		parent = "root"
		handle = ""
		delay = time.Second
		delayJitter = time.Second
		drop = 5
		duplicate = 5
		corrupt = 1
		bands = 16
		priomap = [16]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
		srcIP = &net.IPNet{
			IP:   net.IPv4(192, 168, 0, 1),
			Mask: net.CIDRMask(32, 32),
		}
		dstIP = &net.IPNet{
			IP:   net.IPv4(10, 0, 0, 1),
			Mask: net.CIDRMask(32, 32),
		}
		srcPort = 12345
		dstPort = 80
		protocol = newProtocol("TCP")
		connState = ConnStateNew
		flowid = "1:2"
	})

	Describe("AddNetem", func() {
		JustBeforeEach(func() {
			Expect(tcRunner.AddNetem(ifaces, parent, handle, delay, delayJitter, drop, corrupt, duplicate)).Should(Succeed())
		})

		Context("add 1s delay and 1s delayJitter to lo interface to the root parent without any handle", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "add", "dev", "lo", "root", "netem", "delay", "1000ms", "1000ms", "distribution", "normal", "loss", "5%", "duplicate", "5%", "corrupt", "1%"})
			})
		})

		Context("add delay and delayJitter with a handle", func() {
			BeforeEach(func() {
				handle = "1:"
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "add", "dev", "lo", "root", "handle", "1:", "netem", "delay", "1000ms", "1000ms", "distribution", "normal", "loss", "5%", "duplicate", "5%", "corrupt", "1%"})
			})
		})

		Context("add delay and delayJitter to the a non-root parent", func() {
			BeforeEach(func() {
				parent = "1:4"
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "add", "dev", "lo", "parent", "1:4", "netem", "delay", "1000ms", "1000ms", "distribution", "normal", "loss", "5%", "duplicate", "5%", "corrupt", "1%"})
			})
		})
	})

	Describe("AddPrio", func() {
		JustBeforeEach(func() {
			Expect(tcRunner.AddPrio(ifaces, parent, handle, bands, priomap)).Should(Succeed())
		})

		Context("add a 16 bands prio with a priomap with one case per band", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "add", "dev", "lo", "root", "prio", "bands", "16", "priomap", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16"})
			})
		})
	})

	Describe("AddFilter", func() {
		JustBeforeEach(func() {
			_, err := tcRunner.AddFilter(ifaces, parent, handle, srcIP, dstIP, srcPort, dstPort, protocol, connState, flowid)
			Expect(err).ShouldNot(HaveOccurred())
		})

		Context("add a filter on packets going to IP 10.0.0.1 and port 80 with flowid 1:4 on egress traffic", func() {
			BeforeEach(func() {
				srcIP = nil
				srcPort = 0
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"filter", "add", "dev", "lo", "protocol", "ip", "priority", "1001", "root", "flower", "ip_proto", "tcp", "dst_ip", "10.0.0.1/32", "dst_port", "80", "ct_state", "+trk+new", "flowid", "1:2"})
			})
		})

		Context("add a filter on packets leaving IP 192.168.0.1 and using port 12345 with flowid 1:4 on egress traffic", func() {
			BeforeEach(func() {
				dstIP = nil
				dstPort = 0
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"filter", "add", "dev", "lo", "protocol", "ip", "priority", "1001", "root", "flower", "ip_proto", "tcp", "src_ip", "192.168.0.1/32", "src_port", "12345", "ct_state", "+trk+new", "flowid", "1:2"})
			})
		})

		Context("add a filter on packets leaving IP 192.168.0.1 port 12345 and going to IP 10.0.0.1 port 80 with flowid 1:4 on egress traffic", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"filter", "add", "dev", "lo", "protocol", "ip", "priority", "1001", "root", "flower", "ip_proto", "tcp", "src_ip", "192.168.0.1/32", "dst_ip", "10.0.0.1/32", "src_port", "12345", "dst_port", "80", "ct_state", "+trk+new", "flowid", "1:2"})
			})
		})
	})

	Describe("AddFwFilter", func() {
		JustBeforeEach(func() {
			Expect(tcRunner.AddFwFilter(ifaces, parent, handle, flowid)).Should(Succeed())
		})

		Context("add a cgroup filter", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"filter", "add", "dev", "lo", "protocol", "ip", "root", "fw", "flowid", "1:2"})
			})
		})
	})

	Describe("AddOutputLimit", func() {
		JustBeforeEach(func() {
			Expect(tcRunner.AddOutputLimit(ifaces, parent, handle, 12345)).Should(Succeed())
		})
		Context("add an output limit on root device of 12345 bytes per second", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "add", "dev", "lo", "root", "tbf", "rate", "12345", "latency", "50ms", "burst", "12345"})
			})
		})
	})

	Describe("ClearQdisc", func() {
		JustBeforeEach(func() {
			Expect(tcRunner.ClearQdisc(ifaces)).Should(Succeed())
		})

		Context("clear qdisc for local interface", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "del", "dev", "lo", "root"})
			})
		})

		Context("clear an already cleared qdisc", func() {
			BeforeEach(func() {
				tcExecuterRunCall.Return(2, "", nil) // return exit code 2
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", []string{"qdisc", "del", "dev", "lo", "root"})
			})
		})
	})
})

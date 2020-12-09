// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.
package network

import (
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type tcExecuterMock struct {
	mock.Mock
}

func (f *tcExecuterMock) Run(args ...string) (string, error) {
	a := f.Called(strings.Join(args, " "))

	return a.String(0), a.Error(1)
}

var _ = Describe("Tc", func() {
	var (
		tcRunner          tc
		tcExecuter        tcExecuterMock
		tcExecuterRunCall *mock.Call
		iface             string
		parent            string
		handle            uint32
		delay             time.Duration
		delayJitter       time.Duration
		drop              int
		duplicate         int
		corrupt           int
		bands             uint32
		priomap           [16]uint32
		srcIP, dstIP      *net.IPNet
		srcPort, dstPort  int
		protocol          string
		flowid            string
	)

	BeforeEach(func() {
		// fake command executer
		tcExecuter = tcExecuterMock{}
		tcExecuterRunCall = tcExecuter.On("Run", mock.Anything).Return("", nil)

		// tc runner
		tcRunner = tc{
			executer: &tcExecuter,
		}

		// injected variables
		iface = "lo"
		parent = "root"
		handle = 0
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
		protocol = "tcp"
		flowid = "1:2"
	})

	Describe("AddNetem", func() {
		JustBeforeEach(func() {
			tcRunner.AddNetem(iface, parent, handle, delay, delayJitter, drop, corrupt, duplicate)
		})

		Context("add 1s delay and 1s delayJitter to lo interface to the root parent without any handle", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo root netem delay 1s 1s distribution normal loss 5% duplicate 5% corrupt 1%")
			})
		})

		Context("add delay and delayJitter with a handle", func() {
			BeforeEach(func() {
				handle = 1
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo root handle 1: netem delay 1s 1s distribution normal loss 5% duplicate 5% corrupt 1%")
			})
		})

		Context("add delay and delayJitter to the a non-root parent", func() {
			BeforeEach(func() {
				parent = "1:4"
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo parent 1:4 netem delay 1s 1s distribution normal loss 5% duplicate 5% corrupt 1%")
			})
		})

		Context("add a 30 minutes delay and 5m delayJitter", func() {
			BeforeEach(func() {
				delay = 30 * time.Minute
				delayJitter = 5 * time.Minute
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo root netem delay 30m0s 5m0s distribution normal loss 5% duplicate 5% corrupt 1%")
			})
		})
	})

	Describe("AddPrio", func() {
		JustBeforeEach(func() {
			tcRunner.AddPrio(iface, parent, handle, bands, priomap)
		})

		Context("add a 16 bands prio with a priomap with one case per band", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo root prio bands 16 priomap 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16")
			})
		})
	})

	Describe("AddFilter", func() {
		JustBeforeEach(func() {
			tcRunner.AddFilter(iface, parent, handle, srcIP, dstIP, srcPort, dstPort, protocol, flowid)
		})

		Context("add a filter on packets going to IP 10.0.0.1 and port 80 with flowid 1:4 on egress traffic", func() {
			BeforeEach(func() {
				srcIP = nil
				srcPort = 0
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "filter add dev lo root u32 match ip dst 10.0.0.1/32 match ip dport 80 0xffff match ip protocol 6 0xff flowid 1:2")
			})
		})

		Context("add a filter on packets leaving IP 192.168.0.1 and using port 12345 with flowid 1:4 on egress traffic", func() {
			BeforeEach(func() {
				dstIP = nil
				dstPort = 0
			})

			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "filter add dev lo root u32 match ip src 192.168.0.1/32 match ip sport 12345 0xffff match ip protocol 6 0xff flowid 1:2")
			})
		})

		Context("add a filter on packets leaving IP 192.168.0.1 port 12345 and going to IP 10.0.0.1 port 80 with flowid 1:4 on egress traffic", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "filter add dev lo root u32 match ip src 192.168.0.1/32 match ip dst 10.0.0.1/32 match ip sport 12345 0xffff match ip dport 80 0xffff match ip protocol 6 0xff flowid 1:2")
			})
		})
	})

	Describe("AddCgroupFilter", func() {
		JustBeforeEach(func() {
			tcRunner.AddCgroupFilter(iface, parent, handle)
		})

		Context("add a cgroup filter", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "filter add dev lo root cgroup")
			})
		})
	})

	Describe("AddOutputLimit", func() {
		JustBeforeEach(func() {
			tcRunner.AddOutputLimit(iface, parent, handle, 12345)
		})
		Context("add an output limit on root device of 12345 bytes per second", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc add dev lo root tbf rate 12345 latency 50ms burst 12345")
			})
		})
	})

	Describe("ClearQdisc", func() {
		JustBeforeEach(func() {
			tcRunner.ClearQdisc(iface)
		})

		Context("clear qdisc for local interface", func() {
			It("should execute", func() {
				tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc del dev lo root")
			})
		})
	})

	Describe("IsQdiscCleared", func() {
		var (
			cleared bool
			err     error
		)

		JustBeforeEach(func() {
			cleared, err = tcRunner.IsQdiscCleared(iface)
			Expect(err).To(BeNil())
			tcExecuter.AssertCalled(GinkgoT(), "Run", "qdisc show dev lo")
		})

		Context("non-cleared qdisc", func() {
			BeforeEach(func() {
				tcExecuterRunCall.Return("qdisc prio 1: root refcnt 2 bands 4 priomap  1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1", nil)
			})
			It("should return false", func() {
				Expect(cleared).To(BeFalse())
			})
		})

		Context("already cleared qdisc", func() {
			BeforeEach(func() {
				tcExecuterRunCall.Return("qdisc noqueue 0: root refcnt 2", nil)
			})
			It("should return false", func() {
				Expect(cleared).To(BeTrue())
			})
		})
	})
})

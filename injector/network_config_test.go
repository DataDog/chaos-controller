// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"

	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
)

// tc
type fakeTc struct {
	mock.Mock
}

func (f *fakeTc) AddDelay(iface string, parent string, handle uint32, delay time.Duration) error {
	args := f.Called(iface, parent, handle, delay)
	return args.Error(0)
}
func (f *fakeTc) AddPrio(iface string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	args := f.Called(iface, parent, handle, bands, priomap)
	return args.Error(0)
}
func (f *fakeTc) AddFilter(iface string, parent string, handle uint32, ip *net.IPNet, port int, flowid string) error {
	args := f.Called(iface, parent, handle, ip.String(), port, flowid)
	return args.Error(0)
}
func (f *fakeTc) AddOutputLimit(iface string, parent string, handle uint32, bytesPerSec uint) error {
	args := f.Called(iface, parent, handle, bytesPerSec)
	return args.Error(0)
}
func (f *fakeTc) ClearQdisc(iface string) error {
	args := f.Called(iface)
	return args.Error(0)
}
func (f *fakeTc) IsQdiscCleared(iface string) (bool, error) {
	args := f.Called(iface)
	return args.Bool(0), args.Error(1)
}

// netlink
type fakeNetlinkAdapter struct {
	mock.Mock
}

func (f *fakeNetlinkAdapter) LinkList() ([]network.NetlinkLink, error) {
	args := f.Called()
	return args.Get(0).([]network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) LinkByIndex(index int) (network.NetlinkLink, error) {
	args := f.Called(index)
	return args.Get(0).(network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) LinkByName(name string) (network.NetlinkLink, error) {
	args := f.Called(name)
	return args.Get(0).(network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) RoutesForIP(ip *net.IPNet) ([]network.NetlinkRoute, error) {
	args := f.Called(ip.String())
	return args.Get(0).([]network.NetlinkRoute), args.Error(1)
}

type fakeNetlinkLink struct {
	mock.Mock
}

func (f *fakeNetlinkLink) Name() string {
	args := f.Called()
	return args.String(0)
}
func (f *fakeNetlinkLink) SetTxQLen(qlen int) error {
	args := f.Called(qlen)
	return args.Error(0)
}
func (f *fakeNetlinkLink) TxQLen() int {
	args := f.Called()
	return args.Int(0)
}

type fakeNetlinkRoute struct {
	mock.Mock
}

func (f *fakeNetlinkRoute) Link() network.NetlinkLink {
	args := f.Called()
	return args.Get(0).(network.NetlinkLink)
}

var _ = Describe("Tc", func() {
	var (
		config                               NetworkDisruptionConfig
		tc                                   fakeTc
		tcIsQdiscClearedCall                 *mock.Call
		nl                                   fakeNetlinkAdapter
		nllink1, nllink2                     *fakeNetlinkLink
		nllink1TxQlenCall, nllink2TxQlenCall *mock.Call
		nlroute1, nlroute2                   *fakeNetlinkRoute
	)

	BeforeEach(func() {
		// tc
		tc = fakeTc{}
		tc.On("AddDelay", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
	})

	JustBeforeEach(func() {
		config = NetworkDisruptionConfigStruct{
			Log:               log,
			TrafficController: &tc,
			NetlinkAdapter:    &nl,
			DNSClient:         nil,
		}
	})

	Describe("addOperation", func() {

		Context("with no host specified", func() {
			JustBeforeEach(func() {
				//
				// NOTE: We're using `AddLatency` here just to make sure the chain is called
				// since addOperation is private, but any would work
				//
				config.AddLatency([]string{}, 0, time.Second)
			})
			It("should not set or clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		Context("with multiple hosts specified and interface without qlen", func() {
			BeforeEach(func() {
				config.AddLatency([]string{"1.1.1.1", "2.2.2.2"}, 80, time.Second)
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
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "1.1.1.1/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "lo", "1:0", mock.Anything, "2.2.2.2/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "1.1.1.1/32", 80, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", "eth0", "1:0", mock.Anything, "2.2.2.2/32", 80, "1:4")
			})
		})

		Context("with multiple hosts specified and interfaces with qlen", func() {
			BeforeEach(func() {
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
				config.AddLatency([]string{"1.1.1.1", "2.2.2.2"}, 0, time.Second)
			})
			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})
	})

	Describe("ClearAllQdiscs", func() {
		JustBeforeEach(func() {
			config.ClearAllQdiscs([]string{})
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

	Describe("AddLatency", func() {
		BeforeEach(func() {
			config.AddLatency([]string{}, 0, time.Second)
		})

		It("should add delay to the interfaces root qdisc", func() {
			tc.AssertCalled(GinkgoT(), "AddDelay", "lo", "root", mock.Anything, time.Second)
			tc.AssertCalled(GinkgoT(), "AddDelay", "eth0", "root", mock.Anything, time.Second)
		})
	})

	Describe("AddOutputLimit", func() {
		BeforeEach(func() {
			config.AddOutputLimit([]string{}, 0, uint(12345))
		})

		It("should add delay to the interfaces root qdisc", func() {
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", "lo", "root", mock.Anything, uint(12345))
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "root", mock.Anything, uint(12345))
		})
	})

})

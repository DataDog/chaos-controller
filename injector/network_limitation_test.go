// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/network"
)

//
// no need to define `fakeTc`, `fakeNetlinkLink`, etc -- defined (same package) in `network_latency_test.go`
//

var _ = Describe("Tc", func() {
	var (
		c                    fakeContainer
		inj                  Injector
		config               NetworkLimitationInjectorConfig
		spec                 v1beta1.NetworkLimitationSpec
		tc                   fakeTc
		tcIsQdiscClearedCall *mock.Call
		nl                   fakeNetlinkAdapter
		nllink1, nllink2     *fakeNetlinkLink
		nlroute1, nlroute2   *fakeNetlinkRoute
	)

	BeforeEach(func() {
		// container
		c = fakeContainer{}
		c.On("EnterNetworkNamespace").Return(nil)
		c.On("ExitNetworkNamespace").Return(nil)

		// tc
		tc = fakeTc{}
		tc.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("ClearQdisc", mock.Anything).Return(nil)
		tcIsQdiscClearedCall = tc.On("IsQdiscCleared", mock.Anything).Return(false, nil)

		// netlink
		nllink1 = &fakeNetlinkLink{}
		nllink1.On("Name").Return("lo")
		nllink1.On("SetTxQLen", mock.Anything).Return(nil)
		nllink2 = &fakeNetlinkLink{}
		nllink2.On("Name").Return("eth0")
		nllink2.On("SetTxQLen", mock.Anything).Return(nil)

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

		spec = v1beta1.NetworkLimitationSpec{
			BytesPerSec: 12345,
		}
		config = NetworkLimitationInjectorConfig{
			TrafficController: &tc,
			NetlinkAdapter:    &nl,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkLimitationInjectorWithConfig("fake", spec, &c, log, ms, config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		Context("with no host specified", func() {
			It("should enter and exit the container network namespace", func() {
				Expect(c.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
				Expect(c.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
			})
			It("should add output limit to the interfaces root qdisc", func() {
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "lo", "root", mock.Anything, uint(12345))
				tc.AssertCalled(GinkgoT(), "AddOutputLimit", "eth0", "root", mock.Anything, uint(12345))
			})
		})

		Describe("inj.Clean", func() {
			JustBeforeEach(func() {
				inj.Clean()
			})

			Context("with a non-cleared qdisc", func() {
				It("should enter and exit the container network namespace", func() {
					Expect(c.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
					Expect(c.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
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
					Expect(c.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
					Expect(c.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
				})
				It("should not clear the interfaces qdisc", func() {
					tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "lo")
					tc.AssertNotCalled(GinkgoT(), "ClearQdisc", "eth0")
				})
			})
		})
	})
})

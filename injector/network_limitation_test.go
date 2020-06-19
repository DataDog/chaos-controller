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
)

// fakeNetworkConfig mock implementation is already defined in `network_latency_test.go`

var _ = Describe("Limitation", func() {
	var (
		ctn      fakeContainer
		inj    Injector
		config fakeNetworkConfig
		spec   v1beta1.NetworkLimitationSpec
	)

	BeforeEach(func() {
		// container
		ctn = fakeContainer{}
		ctn.On("EnterNetworkNamespace").Return(nil)
		ctn.On("ExitNetworkNamespace").Return(nil)

		config = fakeNetworkConfig{}
		config.On("AddLatency", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		config.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		config.On("ClearAllQdiscs", mock.Anything).Return(nil)

		spec = v1beta1.NetworkLimitationSpec{
			Hosts:       []string{"testhost"},
			Port:        22,
			BytesPerSec: 12345,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkLimitationInjectorWithConfig("fake", spec, &ctn, log, ms, &config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should enter and exit the container network namespace", func() {
			Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
			Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
		})

		It("should call AddOutputLimit on its network disruption config", func() {
			Expect(config.AssertCalled(GinkgoT(), "AddOutputLimit", spec.Hosts, spec.Port, spec.BytesPerSec)).To(BeTrue())
		})

		Describe("inj.Clean", func() {
			JustBeforeEach(func() {
				inj.Clean()
			})

			It("should enter and exit the container network namespace", func() {
				Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
				Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
			})

			It("should call ClearAllQdiscs on its network disruption config", func() {
				Expect(config.AssertCalled(GinkgoT(), "ClearAllQdiscs", spec.Hosts)).To(BeTrue())
			})
		})
	})
})

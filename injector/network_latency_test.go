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
)

type fakeNetworkConfig struct {
	mock.Mock
}

func (f *fakeNetworkConfig) AddOutputLimit(hosts []string, port int, bytesPerSec uint) {
	f.Called(hosts, port, bytesPerSec)
	return
}
func (f *fakeNetworkConfig) AddLatency(hosts []string, port int, delay time.Duration) {
	f.Called(hosts, port, delay)
	return
}
func (f *fakeNetworkConfig) ClearAllQdiscs(hosts []string) {
	f.Called(hosts)
	return
}

var _ = Describe("Latency", func() {
	var (
		c      fakeContainer
		inj    Injector
		config fakeNetworkConfig
		spec   v1beta1.NetworkLatencySpec
	)

	BeforeEach(func() {
		// container
		c = fakeContainer{}
		c.On("EnterNetworkNamespace").Return(nil)
		c.On("ExitNetworkNamespace").Return(nil)

		// network disruption conf
		config = fakeNetworkConfig{}
		config.On("AddLatency", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		config.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		config.On("ClearAllQdiscs", mock.Anything).Return(nil)

		spec = v1beta1.NetworkLatencySpec{
			Hosts: []string{"testhost"},
			Port: 22,
			Delay: 1000,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkLatencyInjectorWithConfig("fake", spec, &c, log, ms, &config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should enter and exit the container network namespace", func() {
			Expect(c.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
			Expect(c.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
		})

		It("should call AddLatency on its network disruption config", func() {
			delay_ms := time.Duration(spec.Delay) * time.Millisecond
			Expect(config.AssertCalled(GinkgoT(), "AddLatency", spec.Hosts, spec.Port, delay_ms)).To(BeTrue())
		})

		Describe("inj.Clean", func() {
			JustBeforeEach(func() {
				inj.Clean()
			})

			It("should enter and exit the container network namespace", func() {
				Expect(c.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
				Expect(c.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
			})

			It("should call ClearAllQdiscs on its network disruption config", func() {
				Expect(config.AssertCalled(GinkgoT(), "ClearAllQdiscs", spec.Hosts)).To(BeTrue())
			})
		})
	})
})

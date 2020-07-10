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

func (f *fakeNetworkConfig) AddNetem(delay time.Duration, drop int, corrupt int) {
	f.Called(delay, drop, corrupt)
}
func (f *fakeNetworkConfig) AddOutputLimit(bytesPerSec uint) {
	f.Called(bytesPerSec)
}
func (f *fakeNetworkConfig) ApplyOperations() error {
	args := f.Called()

	return args.Error(0)
}
func (f *fakeNetworkConfig) ClearOperations() error {
	args := f.Called()

	return args.Error(0)
}

var _ = Describe("Latency", func() {
	var (
		ctn    fakeContainer
		inj    Injector
		config fakeNetworkConfig
		spec   v1beta1.NetworkLatencySpec
	)

	BeforeEach(func() {
		// container
		ctn = fakeContainer{}
		ctn.On("EnterNetworkNamespace").Return(nil)
		ctn.On("ExitNetworkNamespace").Return(nil)

		// network disruption conf
		config = fakeNetworkConfig{}
		config.On("AddNetem", mock.Anything, mock.Anything, mock.Anything).Return()
		config.On("AddOutputLimit", mock.Anything).Return()
		config.On("ApplyOperations").Return(nil)
		config.On("ClearOperations").Return(nil)

		spec = v1beta1.NetworkLatencySpec{
			Hosts: []string{"testhost"},
			Port:  22,
			Delay: 1000,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkLatencyInjectorWithConfig("fake", spec, &ctn, log, ms, &config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should enter and exit the container network namespace", func() {
			Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
			Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
		})

		It("should call AddNetem on its network disruption config", func() {
			delay := time.Duration(spec.Delay) * time.Millisecond
			Expect(config.AssertCalled(GinkgoT(), "AddNetem", delay, 0, 0)).To(BeTrue())
		})

		Describe("inj.Clean", func() {
			JustBeforeEach(func() {
				inj.Clean()
			})

			It("should enter and exit the container network namespace", func() {
				Expect(ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")).To(BeTrue())
				Expect(ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")).To(BeTrue())
			})

			It("should call ClearOperations on its network disruption config", func() {
				Expect(config.AssertCalled(GinkgoT(), "ClearOperations")).To(BeTrue())
			})
		})
	})
})

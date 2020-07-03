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

type fakeDiskInformer struct {
	mock.Mock
}

func (f *fakeDiskInformer) Major() int {
	args := f.Called()
	return args.Int(0)
}

func (f *fakeDiskInformer) Source() string {
	args := f.Called()
	return args.String(0)
}

var _ = Describe("Failure", func() {
	var (
		config   DiskPressureInjectorConfig
		cgroup   *fakeCgroup
		ctn      *fakeContainer
		informer *fakeDiskInformer
		inj      Injector
		spec     v1beta1.DiskPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroup = &fakeCgroup{}
		cgroup.On("DiskThrottleRead", mock.Anything, mock.Anything).Return(nil)
		cgroup.On("DiskThrottleWrite", mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = &fakeContainer{}
		ctn.On("Cgroup").Return(cgroup)

		// disk informer
		informer = &fakeDiskInformer{}
		informer.On("Major").Return(8)
		informer.On("Source").Return("/dev/sda1")

		//config
		config = DiskPressureInjectorConfig{
			Informer: informer,
		}

		// spec
		read := 1024
		write := 4096
		spec = v1beta1.DiskPressureSpec{
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				ReadBytesPerSec:  &read,
				WriteBytesPerSec: &write,
			},
		}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewDiskPressureInjectorWithConfig("fake", spec, ctn, log, ms, config)
		Expect(err).To(BeNil())
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should throttle disk from cgroup", func() {
			cgroup.AssertCalled(GinkgoT(), "DiskThrottleRead", 8, *spec.Throttling.ReadBytesPerSec)
			cgroup.AssertCalled(GinkgoT(), "DiskThrottleWrite", 8, *spec.Throttling.WriteBytesPerSec)
		})
	})

	Describe("clean", func() {
		JustBeforeEach(func() {
			inj.Clean()
		})

		It("should remove dks throttle from cgroup", func() {
			cgroup.AssertCalled(GinkgoT(), "DiskThrottleRead", 8, 0)
			cgroup.AssertCalled(GinkgoT(), "DiskThrottleWrite", 8, 0)
		})
	})
})

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
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/disk"
	. "github.com/DataDog/chaos-controller/injector"
)

var _ = Describe("Failure", func() {
	var (
		config   DiskPressureInjectorConfig
		cgroup   *container.CgroupMock
		ctn      *container.ContainerMock
		informer *disk.InformerMock
		inj      Injector
		spec     v1beta1.DiskPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroup = &container.CgroupMock{}
		cgroup.On("DiskThrottleRead", mock.Anything, mock.Anything).Return(nil)
		cgroup.On("DiskThrottleWrite", mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = &container.ContainerMock{}
		ctn.On("Cgroup").Return(cgroup)

		// disk informer
		informer = &disk.InformerMock{}
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

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.
package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
)

var _ = Describe("Failure", func() {
	var (
		config        DiskPressureInjectorConfig
		cgroupManager *cgroup.ManagerMock
		ctn           *container.ContainerMock
		informer      *disk.InformerMock
		inj           Injector
		spec          v1beta1.DiskPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &cgroup.ManagerMock{}
		cgroupManager.On("DiskThrottleRead", mock.Anything, mock.Anything).Return(nil)
		cgroupManager.On("DiskThrottleWrite", mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = &container.ContainerMock{}

		// disk informer
		informer = &disk.InformerMock{}
		informer.On("Major").Return(8)
		informer.On("Source").Return("/dev/sda1")

		// env vars
		os.Setenv(env.InjectorMountHost, "foo")

		//config
		config = DiskPressureInjectorConfig{
			Config: Config{
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
				Cgroup:          cgroupManager,
			},
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

	AfterEach(func() {
		os.Unsetenv(env.InjectorMountHost)
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewDiskPressureInjector(spec, config)
		Expect(err).To(BeNil())
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		It("should throttle disk from cgroup", func() {
			cgroupManager.AssertCalled(GinkgoT(), "DiskThrottleRead", 8, *spec.Throttling.ReadBytesPerSec)
			cgroupManager.AssertCalled(GinkgoT(), "DiskThrottleWrite", 8, *spec.Throttling.WriteBytesPerSec)
		})
	})

	Describe("clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(BeNil())
		})

		It("should remove throttle from cgroup", func() {
			cgroupManager.AssertCalled(GinkgoT(), "DiskThrottleRead", 8, 0)
			cgroupManager.AssertCalled(GinkgoT(), "DiskThrottleWrite", 8, 0)
		})
	})
})

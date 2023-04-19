// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/mocks"
)

var _ = Describe("Failure", func() {
	var (
		config        DiskPressureInjectorConfig
		cgroupManager *mocks.CGroupManagerMock
		ctn           *mocks.ContainerMock
		informer      *mocks.InformerMock
		inj           Injector
		spec          v1beta1.DiskPressureSpec
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = mocks.NewCGroupManagerMock(GinkgoT())
		cgroupManager.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = mocks.NewContainerMock(GinkgoT())

		// disk informer
		informer = mocks.NewInformerMock(GinkgoT())
		informer.EXPECT().Major().Return(8)
		informer.EXPECT().Source().Return("/dev/sda1")

		// env vars
		os.Setenv(env.InjectorMountHost, "foo")

		// config
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

		Context("with cgroups v1", func() {
			BeforeEach(func() {
				cgroupManager.EXPECT().IsCgroupV2().Return(false)
			})

			It("should throttle disk from cgroup", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.read_bps_device", "8:0 1024")
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.write_bps_device", "8:0 4096")
			})
		})

		Context("with cgroups v2", func() {
			BeforeEach(func() {
				cgroupManager.EXPECT().IsCgroupV2().Return(true)
			})

			It("should throttle disk from cgroup", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 rbps=1024")
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 wbps=4096")
			})
		})
	})

	Describe("clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(BeNil())
		})

		Context("with cgroups v1", func() {
			BeforeEach(func() {
				cgroupManager.EXPECT().IsCgroupV2().Return(false)
			})

			It("should remove throttle from cgroup", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.read_bps_device", "8:0 0")
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.write_bps_device", "8:0 0")
			})
		})

		Context("with cgroups v2", func() {
			BeforeEach(func() {
				cgroupManager.EXPECT().IsCgroupV2().Return(true)
			})

			It("should throttle disk from cgroup", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 rbps=max")
				cgroupManager.AssertCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 wbps=max")
			})
		})
	})
})

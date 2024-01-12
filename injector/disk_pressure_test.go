// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
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
		cgroupManager = cgroup.NewManagerMock(GinkgoT())
		cgroupManager.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// container
		ctn = container.NewContainerMock(GinkgoT())

		// disk informer
		informer = disk.NewInformerMock(GinkgoT())
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
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).ToNot(HaveOccurred())
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
			Expect(inj.Clean()).To(Succeed())
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

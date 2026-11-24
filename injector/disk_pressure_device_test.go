// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// deviceInformer supplies the whole-device identity resolved by the disk package.
type deviceInformer struct{ major, minor int }

func (d deviceInformer) Major() int     { return d.major }
func (d deviceInformer) Minor() int     { return d.minor }
func (d deviceInformer) Source() string { return "/dev/test-disk" }

var _ = DescribeTable("disk pressure device targeting",
	func(major, minor int, v2 bool) {
		GinkgoT().Setenv(env.InjectorMountHost, "/mnt/host")
		manager := cgroup.NewManagerMock(GinkgoT())
		manager.EXPECT().IsCgroupV2().Return(v2)
		device := fmt.Sprintf("%d:%d", major, minor)
		read, write, readIOPS, writeIOPS := 1024, 4096, 50, 75
		for _, mode := range []struct {
			file, key string
			limit     int
		}{
			{"blkio.throttle.read_bps_device", "rbps", read},
			{"blkio.throttle.write_bps_device", "wbps", write},
			{"blkio.throttle.read_iops_device", "riops", readIOPS},
			{"blkio.throttle.write_iops_device", "wiops", writeIOPS},
		} {
			file := mode.file
			inject, clean := fmt.Sprintf("%s %d", device, mode.limit), device+" 0"
			if v2 {
				file = "io.max"
				inject = fmt.Sprintf("%s %s=%d", device, mode.key, mode.limit)
				clean = fmt.Sprintf("%s %s=max", device, mode.key)
			}
			manager.EXPECT().Write("blkio", file, inject).Return(nil).Once()
			manager.EXPECT().Write("blkio", file, clean).Return(nil).Once()
		}
		inj, err := NewDiskPressureInjector(v1beta1.DiskPressureSpec{
			Throttling: v1beta1.DiskPressureThrottlingSpec{
				ReadBytesPerSec: &read, WriteBytesPerSec: &write,
				ReadIOPSPerSec: &readIOPS, WriteIOPSPerSec: &writeIOPS,
			},
		}, DiskPressureInjectorConfig{
			Config:   Config{Log: log, MetricsSink: ms, Cgroup: manager},
			Informer: deviceInformer{major, minor},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(inj.Inject()).To(Succeed())
		Expect(inj.Clean()).To(Succeed())
	},
	Entry("minor zero, cgroups v1", 8, 0, false),
	Entry("minor zero, cgroups v2", 8, 0, true),
	Entry("NVMe nonzero minor, cgroups v1", 259, 3, false),
	Entry("NVMe nonzero minor, cgroups v2", 259, 3, true),
	Entry("second SCSI disk, cgroups v1", 8, 16, false),
	Entry("second SCSI disk, cgroups v2", 8, 16, true),
)

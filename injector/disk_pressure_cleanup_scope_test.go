// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = DescribeTable("disk pressure cleanup preserves unconfigured IOPS limits",
	func(v2 bool) {
		GinkgoT().Setenv(env.InjectorMountHost, "")
		manager := cgroup.NewManagerMock(GinkgoT())
		manager.EXPECT().IsCgroupV2().Return(v2)
		manager.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil)

		informer := disk.NewInformerMock(GinkgoT())
		informer.EXPECT().Major().Return(8)
		informer.EXPECT().Minor().Return(0)
		informer.EXPECT().Source().Return("/dev/sda")

		read := 1024
		spec := v1beta1.DiskPressureSpec{
			Path:       "/data",
			Throttling: v1beta1.DiskPressureThrottlingSpec{ReadBytesPerSec: &read},
		}
		config := DiskPressureInjectorConfig{
			Config: Config{
				Log:        log,
				Cgroup:     manager,
				Disruption: api.DisruptionArgs{Level: types.DisruptionLevelNode},
			},
			Informer: informer,
		}
		inj, err := NewDiskPressureInjector(spec, config)
		Expect(err).ToNot(HaveOccurred())
		Expect(inj.Inject()).To(Succeed())
		Expect(inj.Clean()).To(Succeed())

		if v2 {
			manager.AssertNotCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 riops=max")
			manager.AssertNotCalled(GinkgoT(), "Write", "blkio", "io.max", "8:0 wiops=max")
		} else {
			manager.AssertNotCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.read_iops_device", mock.Anything)
			manager.AssertNotCalled(GinkgoT(), "Write", "blkio", "blkio.throttle.write_iops_device", mock.Anything)
		}
	},
	Entry("cgroup v1", false),
	Entry("cgroup v2", true),
)

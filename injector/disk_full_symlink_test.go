// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/types"
)

var _ = Describe("DiskFull pod path resolution", func() {
	var volume string

	BeforeEach(func() {
		volume = GinkgoT().TempDir()
		os.Setenv(env.InjectorMountHost, "")
		DeferCleanup(os.Unsetenv, env.InjectorMountHost)
	})

	newInjector := func(hostPath string) error {
		runtime := container.NewRuntimeMock(GinkgoT())
		runtime.EXPECT().HostPath(mock.Anything, mock.Anything, "/data/logs").Return(hostPath, nil)
		ctr := container.NewContainerMock(GinkgoT())
		ctr.EXPECT().Runtime().Return(runtime)
		ctr.EXPECT().ID().Return("id").Maybe()
		ctr.EXPECT().Name().Return("app").Maybe()

		_, err := NewDiskFullInjector(v1beta1.DiskFullSpec{Path: "/data/logs", Capacity: "95%"}, DiskFullInjectorConfig{Config: Config{
			Log: log, MetricsSink: ms, TargetContainer: ctr,
			Disruption: chaosapi.DisruptionArgs{Level: types.DisruptionLevelPod, DisruptionName: "d", DisruptionNamespace: "ns", TargetName: "t"},
		}})

		return err
	}

	It("accepts a real directory below the volume", func() {
		Expect(os.Mkdir(filepath.Join(volume, "logs"), 0o755)).To(Succeed())
		Expect(newInjector(filepath.Join(volume, "logs"))).To(Succeed())
	})

	It("refuses a path that traverses a symbolic link out of the volume", func() {
		outside := GinkgoT().TempDir()
		Expect(os.Symlink(outside, filepath.Join(volume, "logs"))).To(Succeed())
		Expect(newInjector(filepath.Join(volume, "logs"))).To(MatchError(ContainSubstring("symbolic link")))
	})
})

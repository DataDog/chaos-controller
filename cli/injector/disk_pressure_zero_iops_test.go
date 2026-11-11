// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestDiskPressureCLIExplicitZeroIOPS(t *testing.T) {
	for name, v2 := range map[string]bool{"cgroup-v1": false, "cgroup-v2": true} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			fixtures := map[string]string{
				"df": "#!/bin/sh\nprintf '%s\\n' Filesystem /dev/fake\n",
				"ls": "#!/bin/sh\nprintf '%s\\n' 'brw-rw---- 1 root disk 8, 0 Jan 1 00:00 /dev/fake'\n",
			}
			for executable, body := range fixtures {
				if err := os.WriteFile(filepath.Join(dir, executable), []byte(body), 0755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", dir)
			t.Setenv(env.InjectorMountHost, "")

			zero := 0
			count := intstr.FromInt(1)
			spec := v1beta1.DisruptionSpec{
				Level:    types.DisruptionLevelNode,
				Count:    &count,
				Selector: labels.Set{"app": "target"},
				Duration: "1m",
				DiskPressure: &v1beta1.DiskPressureSpec{
					Path: dir,
					Throttling: v1beta1.DiskPressureThrottlingSpec{
						ReadIOPSPerSec:  &zero,
						WriteIOPSPerSec: &zero,
					},
				},
			}
			// ValidateCreate calls this same full-spec validator after CRD validation.
			if err := spec.Validate(); err != nil {
				t.Fatalf("explicit zero IOPS did not pass admission spec validation: %v", err)
			}

			readFile, writeFile := "blkio.throttle.read_iops_device", "blkio.throttle.write_iops_device"
			readData, writeData := "8:0 0", "8:0 0"
			if v2 {
				readFile, writeFile = "io.max", "io.max"
				readData, writeData = "8:0 riops=max", "8:0 wiops=max"
			}
			manager := cgroup.NewManagerMock(t)
			manager.EXPECT().IsCgroupV2().Return(v2).Maybe()
			manager.EXPECT().Write("blkio", readFile, readData).Return(nil).Once()
			manager.EXPECT().Write("blkio", writeFile, writeData).Return(nil).Once()

			previousConfigs, previousInjectors, previousLog := configs, injectors, log
			t.Cleanup(func() {
				configs, injectors, log = previousConfigs, previousInjectors, previousLog
			})
			log = zap.NewNop().Sugar()
			configs = []injector.Config{{
				Log:        log,
				Cgroup:     manager,
				Disruption: api.DisruptionArgs{Level: types.DisruptionLevelNode},
			}}
			injectors = nil

			cmd := &cobra.Command{}
			cmd.Flags().String("path", "", "")
			for _, flag := range []string{"read-bytes-per-sec", "write-bytes-per-sec", "read-iops-per-sec", "write-iops-per-sec"} {
				cmd.Flags().Int(flag, 0, "")
			}
			args := spec.DiskPressure.GenerateArgs()
			if err := cmd.Flags().Parse(args[1:]); err != nil {
				t.Fatal(err)
			}
			diskPressureCmd.PreRun(cmd, nil)
			if len(injectors) != 1 {
				t.Fatalf("expected one configured injector, got %d", len(injectors))
			}
			if err := injectors[0].Inject(); err != nil {
				t.Fatal(err)
			}
			manager.AssertCalled(t, "Write", "blkio", readFile, readData)
			manager.AssertCalled(t, "Write", "blkio", writeFile, writeData)
		})
	}
}

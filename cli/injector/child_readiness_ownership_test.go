// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func TestChildReadinessOwnership(t *testing.T) {
	const helperEnv = "CHAOS_TEST_READINESS_HELPER"
	if mode := os.Getenv(helperEnv); mode != "" {
		log = zap.NewNop().Sugar()
		ms = noop.New(log)
		readyToInject = true
		injectorCtx = context.Background()
		deadline = time.Now().Add(time.Minute)
		disruptionArgs = chaosapi.DisruptionArgs{Level: chaostypes.DisruptionLevelNode, DryRun: true}
		parentPID = 0
		if mode == "child" {
			parentPID = uint32(os.Getppid())
		}
		signals = make(chan os.Signal, 1)
		inj := injector.NewInjectorMock(t)
		inj.EXPECT().Inject().RunAndReturn(func() error {
			signals <- syscall.SIGTERM
			return nil
		}).Once()
		inj.EXPECT().GetDisruptionKind().Return(chaostypes.DisruptionKindName(chaostypes.DisruptionKindCPUStress)).Once()
		injectors = []targetInjector{{config: 0, Injector: inj}}

		injectAndWait(&cobra.Command{Use: cpuStressCommandName}, nil)

		_, err := os.Stat(readinessProbeFile)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		ready := err == nil
		if ready != (mode == "parent") {
			t.Fatalf("%s process published readiness=%t; want %t", mode, ready, mode == "parent")
		}
		return
	}

	for _, mode := range []string{"child", "parent"} {
		t.Run(mode, func(t *testing.T) {
			// Never overwrite or remove another process's existing readiness file.
			if _, err := os.Stat(readinessProbeFile); err == nil {
				t.Skip("readiness probe already exists; run in an isolated test environment")
			} else if !os.IsNotExist(err) {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Remove(readinessProbeFile); err != nil && !os.IsNotExist(err) {
					t.Error(err)
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestChildReadinessOwnership$", "-test.v")
			cmd.Env = append(os.Environ(), helperEnv+"="+mode)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("readiness helper failed: %v\n%s", err, output)
			}
		})
	}
}

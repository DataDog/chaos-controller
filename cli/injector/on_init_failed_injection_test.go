// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
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

func TestOnInitFailedInjectionKeepsHandlerBlocked(t *testing.T) {
	const helperEnv = "CHAOS_TEST_FAILED_ONINIT_HELPER"
	if os.Getenv(helperEnv) != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOnInitFailedInjectionKeepsHandlerBlocked$", "-test.v")
		cmd.Env = append(os.Environ(), helperEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("on-init helper failed: %v\n%s", err, output)
		}
		return
	}

	// Isolate real SIGUSR1 delivery and the command globals in this subprocess.
	handlerSignals := make(chan os.Signal, 1)
	signal.Notify(handlerSignals, syscall.SIGUSR1)
	defer signal.Stop(handlerSignals)
	log = zap.NewNop().Sugar()
	ms = noop.New(log)
	readyToInject = true
	injectorCtx = context.Background()
	deadline = time.Now().Add(time.Minute)
	parentPID = 0
	handlerPID = uint32(os.Getpid())
	disruptionArgs = chaosapi.DisruptionArgs{Level: chaostypes.DisruptionLevelPod, OnInit: true, DryRun: true}
	signals = make(chan os.Signal, 1)
	inj := injector.NewInjectorMock(t)
	inj.EXPECT().Inject().RunAndReturn(func() error {
		signals <- syscall.SIGTERM
		return errors.New("native regression: injection failed")
	}).Once()
	inj.EXPECT().TargetName().Return("target").Once()
	injectors = []targetInjector{{config: 0, Injector: inj}}

	injectAndWait(&cobra.Command{Use: chaostypes.DisruptionKindNetworkDisruption}, nil)

	select {
	case sig := <-handlerSignals:
		t.Fatalf("failed injection released the on-init handler with %s", sig)
	case <-time.After(time.Second):
	}
}

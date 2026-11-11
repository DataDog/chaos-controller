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
	"strings"
	"testing"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

func TestInjectorCommandIsNotSelectedByMetadata(t *testing.T) {
	const helperArgs = "CHAOS_INJECTOR_COMMAND_METADATA_TEST_ARGS"
	if encoded := os.Getenv(helperArgs); encoded != "" {
		args := strings.Split(encoded, "\n")
		// Exercise the actual generated argv, Cobra dispatch, and initialization.
		// The unsupported container runtime is a safety sentinel: pod-scoped
		// commands must stop there before touching any runtime, namespace, or cgroup.
		// The genuine pod-replacement command skips containers, so stop it at
		// InClusterConfig instead, before any namespace or cgroup operations.
		t.Setenv("KUBERNETES_SERVICE_HOST", "")
		t.Setenv("KUBERNETES_SERVICE_PORT", "")
		os.Args = append([]string{"chaos-injector"}, args...)
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("command failed before the expected initialization sentinel: %v", err)
		}
		t.Fatal("initialization unexpectedly passed both safety sentinels")
	}

	for _, test := range []struct {
		name    string
		command string
		change  func(*chaosapi.DisruptionArgs)
		wantLog string
	}{
		{
			name: "ordinary pod-scoped command", command: "network-disruption",
			wantLog: "can't create container object",
		},
		{
			name: "disruption name", command: "network-disruption",
			change:  func(a *chaosapi.DisruptionArgs) { a.DisruptionName = "pod-replacement" },
			wantLog: "can't create container object",
		},
		{
			name: "disruption namespace", command: "network-disruption",
			change:  func(a *chaosapi.DisruptionArgs) { a.DisruptionNamespace = "pod-replacement" },
			wantLog: "can't create container object",
		},
		{
			name: "target pod name", command: "network-disruption",
			change:  func(a *chaosapi.DisruptionArgs) { a.TargetName = "pod-replacement" },
			wantLog: "can't create container object",
		},
		{
			name: "target node name", command: "network-disruption",
			change:  func(a *chaosapi.DisruptionArgs) { a.TargetNodeName = "pod-replacement" },
			wantLog: "can't create container object",
		},
		{
			name: "chaos namespace", command: "network-disruption",
			change:  func(a *chaosapi.DisruptionArgs) { a.ChaosNamespace = "pod-replacement" },
			wantLog: "can't create container object",
		},
		{
			name: "actual pod replacement", command: "pod-replacement",
			wantLog: "error getting kubernetes client config",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			common := chaosapi.DisruptionArgs{
				Level:               chaostypes.DisruptionLevelPod,
				Kind:                chaostypes.DisruptionKindName(test.command),
				MetricsSink:         "noop",
				TargetContainers:    map[string]string{"app": "unsupported://safety-sentinel"},
				TargetPodIP:         "192.0.2.10",
				DisruptionName:      "test-disruption",
				DisruptionNamespace: "app-namespace",
				TargetName:          "app-pod",
				TargetNodeName:      "worker-node",
				ChaosNamespace:      "chaos-engineering",
			}
			if test.change != nil {
				test.change(&common)
			}
			args := common.CreateCmdArgs([]string{test.command})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestInjectorCommandIsNotSelectedByMetadata$")
			child.Env = append(os.Environ(), helperArgs+"="+strings.Join(args, "\n"))
			output, err := child.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("initialization did not stop at a safety sentinel: %v; output:\n%s", ctx.Err(), output)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("expected fatal initialization sentinel, got %v; output:\n%s", err, output)
			}
			if !strings.Contains(string(output), test.wantLog) {
				t.Fatalf("metadata changed command scope: expected %q; output:\n%s", test.wantLog, output)
			}
		})
	}
}

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
)

func TestHandlerCommandExitStatus(t *testing.T) {
	const helperArgs = "CHAOS_HANDLER_EXIT_STATUS_TEST_ARGS"
	if args := os.Getenv(helperArgs); args != "" {
		rootCmd.SetArgs(strings.Split(args, "\n"))
		main()
		return
	}

	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "invalid timeout", args: []string{"--timeout=not-a-duration"}, code: 1},
		{name: "unknown flag", args: []string{"--not-a-flag"}, code: 1},
		{name: "ordinary timeout fails", args: []string{"--timeout=0s"}, code: 1},
		{name: "explicit successful timeout", args: []string{"--timeout=0s", "--succeed-on-timeout"}, code: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHandlerCommandExitStatus$")
			child.Env = append(os.Environ(), helperArgs+"="+strings.Join(test.args, "\n"))
			output, err := child.CombinedOutput()
			code := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("cannot run helper: %v", err)
				}
				code = exitErr.ExitCode()
			}
			if code != test.code {
				t.Fatalf("chaos-handler %q: exit code %d, want %d; output:\n%s", test.args, code, test.code, output)
			}
		})
	}
}

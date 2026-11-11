// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestChaosliCommandExitStatus(t *testing.T) {
	const helperArgs = "CHAOSLI_EXIT_STATUS_TEST_ARGS"
	if args := os.Getenv(helperArgs); args != "" {
		rootCmd.SetArgs(strings.Split(args, "\n"))
		Execute()
		return
	}

	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "missing required path", args: []string{"validate"}, code: 1},
		{name: "unknown command", args: []string{"not-a-command"}, code: 1},
		{name: "unknown flag", args: []string{"--not-a-flag"}, code: 1},
		{name: "help succeeds", args: []string{"--help"}, code: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestChaosliCommandExitStatus$")
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
				t.Fatalf("chaosli %q: exit code %d, want %d; output:\n%s", test.args, code, test.code, output)
			}
		})
	}
}

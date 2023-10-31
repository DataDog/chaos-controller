// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ebpf

import (
	"bytes"
	"fmt"
	"os/exec"

	"go.uber.org/zap"
)

type Executor interface {
	Run(args []string) (exitCode int, stdout string, stderr error)
}

type defaultBpftoolExecutor struct {
	log    *zap.SugaredLogger
	dryRun bool
}

// NewBpftoolExecutor create a new instance of an Executor responsible of running bpftool command
func NewBpftoolExecutor(log *zap.SugaredLogger, dryRun bool) Executor {
	return defaultBpftoolExecutor{
		log:    log,
		dryRun: dryRun,
	}
}

// Run executes the given args using the bpftool command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e defaultBpftoolExecutor) Run(args []string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(BpftoolBinary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	e.log.Debugf("running bpftool command: %v", cmd.String())

	// early exit if dry-run mode is enabled
	if e.dryRun {
		return 0, "", nil
	}

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	return cmd.ProcessState.ExitCode(), stdout.String(), err
}

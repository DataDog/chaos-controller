// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"bytes"
	"fmt"
	"os/exec"

	"go.uber.org/zap"
)

// PythonRunner is an interface for executing python3 commands
type PythonRunner interface {
	RunPython(args ...string) (int, string, error)
}

type standardPythonRunner struct {
	dryRun bool
	log    *zap.SugaredLogger
}

// RunPython takes a list of arguments to pass to python3, and returns the exit code
// the stdout of the command, and any errors from cmd.Start()
func (p standardPythonRunner) RunPython(args ...string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("/usr/bin/python3", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	p.log.Infof("running python3 command: %v", cmd.String())

	// early exit if dry-run mode is enabled
	if p.dryRun {
		return 0, "", nil
	}

	err := cmd.Start()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	return cmd.ProcessState.ExitCode(), stdout.String(), err
}

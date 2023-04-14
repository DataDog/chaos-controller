// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

//go:generate mockery --name=PythonRunner --testonly --filename=mock_python_runner_test.go
//go:generate mockery --name=command --testonly

// PythonRunner is an interface for executing python3 commands
type PythonRunner interface {
	RunPython(args ...string) error
}

type standardPythonRunner struct {
	dryRun         bool
	log            *zap.SugaredLogger
	newCmd         func(out, err io.Writer, args ...string) command
	maxErrorLines  int
	maxWaitCommand time.Duration
}

func newStandardPythonRunner(dryRun bool, log *zap.SugaredLogger) *standardPythonRunner {
	return &standardPythonRunner{
		dryRun:         dryRun,
		log:            log,
		newCmd:         newCommandFactory,
		maxErrorLines:  100,
		maxWaitCommand: 250 * time.Millisecond,
	}
}

// RunPython takes a list of arguments to pass to python3, and returns the exit code
// the stdout of the command, and any errors from cmd.Start()
func (p *standardPythonRunner) RunPython(args ...string) error {
	// parse args and execute
	stderr := &bytes.Buffer{}
	cmd := p.newCmd(os.Stdout, io.MultiWriter(os.Stderr, stderr), args...)

	// run command
	p.log.Infof("running python3 command: %v", cmd.String())

	// early exit if dry-run mode is enabled
	if p.dryRun {
		return nil
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start command, encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	// we are launching the process in background and Start does not report a failing process for a wrong configuration as an example (wrong flags, ...)
	// the sole purpose of the following, is to gather early exit command feedback to ease troubleshooting in such case
	// and properly report an error to the caller as expected

	chErr := make(chan error)

	go func() {
		chErr <- cmd.Wait()
	}()

	select {
	case <-time.After(p.maxWaitCommand):
	case err := <-chErr:
		if err != nil {
			return fmt.Errorf("unable to wait command, exited early error (%w) using args (%s): %s", err, args, stderr.String())
		}
	}

	// now that the "Start" of the command went successfully (without an early error)
	// we are only interested in late exit
	// we want to be aware of them, mostly to have debugging informations
	// we are not expecting anything else (like stopping the injector)
	go func() {
		err := <-chErr
		if err != nil {
			p.log.With("error", err).Errorf("command late exit with error (%v) using args (%s): %s", err, args, lastLines(stderr.String(), p.maxErrorLines))
		}
	}()

	return nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")

	if len(lines) <= n {
		return s
	}

	return strings.Join(lines[len(lines)-1:], "\n")
}

type command interface {
	Start() error
	Wait() error
	String() string
}

type wrapCmp struct {
	*exec.Cmd
}

func newCommandFactory(out, err io.Writer, args ...string) command {
	cmd := exec.Command("/usr/bin/python3", args...)
	cmd.Stdout = out
	cmd.Stderr = err

	return &wrapCmp{
		Cmd: cmd,
	}
}

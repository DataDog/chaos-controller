// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"go.uber.org/zap"
)

// PythonRunner is an interface for executing python3 commands
type PythonRunner interface {
	RunPython(args ...string) (int, error)
	ReadBufferFromCommand(pid int)
}

type standardPythonRunner struct {
	dryRun bool
	log    *zap.SugaredLogger
}

func (p standardPythonRunner) ReadBufferFromCommand(pid int) {
	stdoutFile, errStdout := os.Open(fmt.Sprintf("/proc/%d/fd/1", pid))

	if errStdout != nil {
		fmt.Println(errStdout)
		return
	}

	defer stdoutFile.Close()

	stdout := make([]byte, 100)
	for {
		stdout = stdout[:cap(stdout)]
		stdoutNb, err := stdoutFile.Read(stdout)
		if err != nil {
			if err == io.EOF {
				stdout = stdout[:stdoutNb]

				p.log.Debugf(string(stdout))
				p.log.Debugf("logs finished")

				break
			}
			p.log.Debugf("couldn't read logs of dns resolver: %s", err.Error())

			return
		}

		stdout = stdout[:stdoutNb]
		p.log.Debugf(string(stdout))
	}
}

// RunPython takes a list of arguments to pass to python3, and returns the exit code
// the stdout of the command, and any errors from cmd.Start()
func (p standardPythonRunner) RunPython(args ...string) (int, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("/usr/bin/python3", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// early exit if dry-run mode is enabled
	if p.dryRun {
		return 0, nil
	}

	err := cmd.Start()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s)", err, args)
	}

	pid := 0
	p.log.Infof("running python3 command: %v.", cmd.String())

	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	return pid, err
}

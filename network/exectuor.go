// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package network

import (
	"bytes"
	"fmt"
	"os/exec"

	"go.uber.org/zap"
)

type executor interface {
	Run(args []string) (exitCode int, stdout string, stderr error)
}

type defaultTcExecutor struct {
	log    *zap.SugaredLogger
	dryRun bool
}

type ebpfTCFilterConfigExecutor struct {
	log    *zap.SugaredLogger
	dryRun bool
}

// NewBPFTCFilterConfigExecutor create a new instance of an executor responsible of configure tc eBPF filter program
func NewBPFTCFilterConfigExecutor(log *zap.SugaredLogger, dryRun bool) executor {
	return ebpfTCFilterConfigExecutor{
		log:    log,
		dryRun: dryRun,
	}
}

// Run executes the given args using the tc command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e defaultTcExecutor) Run(args []string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(tcPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	e.log.Debugf("running tc command: %v", cmd.String())

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

// Run executes the given args using bpf-network-tc-filter program
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e ebpfTCFilterConfigExecutor) Run(args []string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("/usr/local/bin/bpf-network-tc-filter", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	e.log.Debugf("running eBPF config command: %v", cmd.String())

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

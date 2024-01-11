// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/process"
	"go.uber.org/zap"
)

const (
	NotFoundProcessExitCode = -1
)

var (
	cmdBootstrapAllowedDuration = 1 * time.Second
	cmdKeepAliveTickDuration    = 1 * time.Second
)

// Factory defines how we want to create a command (with context and with relevant fields set)
type Factory interface {
	NewCmd(ctx context.Context, name string, args []string) Cmd
}

// Cmd aims to be a convenient wrapper around os/exec.CommandContext to ease testing and move some process methods up (PID/ExitCode)
type Cmd interface {
	Start() error
	String() string
	Wait() error
	PID() int
	ExitCode() int
	DryRun() bool
}

// BackgroundCmd wraps a ContextCmd methods to provide monitorability of a ContextCmd that is launched in background
type BackgroundCmd interface {
	Start() error
	KeepAlive()
	Stop() error
	DryRun() bool
}

type cmd struct {
	*exec.Cmd
	dryRun bool
}

func (c *cmd) PID() int {
	if c == nil || c.Cmd == nil || c.Cmd.Process == nil {
		return process.NotFoundProcessPID
	}

	return c.Cmd.Process.Pid
}

func (c *cmd) ExitCode() int {
	if c == nil || c.Cmd == nil { // ExitCode check if process state is nil
		return NotFoundProcessExitCode
	}

	return c.Cmd.ProcessState.ExitCode()
}

func (c *cmd) DryRun() bool {
	return c.dryRun
}

type backgroundCmd struct {
	Cmd
	sync.Mutex

	log            *zap.SugaredLogger
	processManager process.Manager
	ticker         *time.Ticker
	chErr          chan error
	pid            int
}

type factory struct {
	dryRun bool
}

func NewFactory(dryRun bool) Factory {
	return factory{
		dryRun,
	}
}

func (f factory) NewCmd(ctx context.Context, name string, args []string) Cmd {
	cmdContext := exec.CommandContext(ctx, name, args...)

	cmdContext.Stdout = os.Stdout
	cmdContext.Stderr = os.Stderr

	return &cmd{
		cmdContext,
		f.dryRun,
	}
}

func NewBackgroundCmd(cmd Cmd, log *zap.SugaredLogger, processManager process.Manager) BackgroundCmd {
	return &backgroundCmd{
		cmd,
		sync.Mutex{},
		log,
		processManager,
		nil,
		nil,
		process.NotFoundProcessPID,
	}
}

func (w *backgroundCmd) DryRun() bool {
	return w.Cmd == nil || w.Cmd.DryRun()
}

func (w *backgroundCmd) Start() error {
	if w.DryRun() {
		return nil
	}

	if err := w.Cmd.Start(); err != nil {
		return fmt.Errorf("unable to exec command '%s': %w", w.Cmd.String(), err)
	}

	chErr := make(chan error, 1)

	go func() {
		chErr <- w.Cmd.Wait()
	}()

	// Here we want to provide a small time for command to bootstrap
	// if an error occur during this time, we can halt early
	select {
	case <-time.After(cmdBootstrapAllowedDuration):
	case err := <-chErr:
		if err != nil {
			return fmt.Errorf("an error occurred during startup of exec command: %w", err)
		}
	}

	w.pid = w.Cmd.PID()
	if w.pid == process.NotFoundProcessPID {
		return fmt.Errorf("no process created, processState exit code is %v", w.Cmd.ExitCode())
	}

	w.chErr = chErr
	w.log = w.log.With("pid", w.pid)

	// Monitoring launched process in background to at least give visibility of exit
	go func() {
		w.log.Debug("new process created, monitoring newly created process exit status")

		if err := <-w.chErr; err != nil {
			w.log.Warnw("background command exited with an error", "error", err)
		} else {
			w.log.Info("background command exited successfully")
		}
	}()

	return nil
}

// KeepAlive will create a goroutine to send regular SIGCONT signal to associated command
// a single goroutine will be launched no matter how many calls are done to KeepAlive
func (w *backgroundCmd) KeepAlive() {
	if w.DryRun() {
		return
	}

	w.Lock()
	defer w.Unlock()

	if w.ticker != nil {
		return
	}

	w.ticker = time.NewTicker(cmdKeepAliveTickDuration)

	w.log.Debug("monitoring sending SIGCONT signal to process every 1s")

	go func() {
		for {
			if w.ticker == nil {
				return
			}

			<-w.safeTicker().C

			proc, err := w.processManager.Find(w.pid)
			if err != nil {
				w.log.Errorw("an error occurred when trying to Find process, stopping to monitor background process, ticker removed", "error", err)

				w.resetTicker()

				return
			}

			if err := w.processManager.Signal(proc, syscall.SIGCONT); err != nil {
				if errors.Is(err, os.ErrProcessDone) {
					w.log.Infof("process is already finished, skipping sending SIGCONT from now on")
				} else {
					w.log.Errorw("an error occurred when sending SIGCONT signal to process, stopping to monitor background process, ticker removed", "error", err)
				}

				w.resetTicker()

				return
			}
		}
	}()
}

func (w *backgroundCmd) safeTicker() *time.Ticker {
	w.Lock()
	defer w.Unlock()

	return w.ticker
}

func (w *backgroundCmd) resetTicker() {
	w.Lock()
	defer w.Unlock()

	if w.ticker == nil {
		return
	}

	w.ticker.Stop()
	w.ticker = nil
}

// Stop will send a SIGTERM signal to associated command
// If executed several times it will most likely return an error due to missing process
func (w *backgroundCmd) Stop() error {
	if w.DryRun() {
		return nil
	}

	w.resetTicker()

	w.log.Infow("sending SIGTERM to background process")

	proc, err := w.processManager.Find(w.pid)
	if err != nil {
		return fmt.Errorf("an error occurred while finding signal with pid %d: %w", w.pid, err)
	}

	err = w.processManager.Signal(proc, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("an error occurred while sending SIGTERM signal to process with pid %d: %w", w.pid, err)
	}

	return nil
}

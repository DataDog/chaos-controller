// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"go.uber.org/zap"
)

// BackgroundProcessManager creates processes that are detached from current process
type BackgroundProcessManager interface {
	// Start will create a command, wait for it's successfull start and notify associated process regularly by sending a SIGCONT signal
	Start(targetContainer string, args ...string) (BackgroundProcess, error)
}

// BackgroundProcess enable to perform actions on a background process
type BackgroundProcess interface {
	// Stop halt any keepAlive associate with background process and related command execution
	Stop()
}

type backgroundProcess struct {
	log            *zap.SugaredLogger
	processManager Manager
	pid            int
	ticker         *time.Ticker
	chErr          chan error
}

type backgroundProcessManager struct {
	log            *zap.SugaredLogger
	processManager Manager
	disruptionArgs chaosapi.DisruptionArgs
	deadline       time.Time
}

func NewBackgroundProcessManager(log *zap.SugaredLogger, processManager Manager, disruptionArgs chaosapi.DisruptionArgs, deadline time.Time) BackgroundProcessManager {
	return backgroundProcessManager{
		log:            log,
		processManager: processManager,
		disruptionArgs: disruptionArgs,
		deadline:       deadline,
	}
}

func (b backgroundProcessManager) Start(targetContainer string, args ...string) (BackgroundProcess, error) {
	log := b.log.With("targetContainer", targetContainer)

	targetContainerID, ok := b.disruptionArgs.TargetContainers[targetContainer]
	if !ok {
		return nil, fmt.Errorf("targeted container does not exists: %s", targetContainer)
	}

	b.disruptionArgs.TargetContainers = map[string]string{targetContainer: targetContainerID}

	args = append(args, "--parent-pid", strconv.Itoa(b.processManager.ProcessID()), "--deadline", b.deadline.Format(time.RFC3339))
	args = chaosapi.AppendArgs(args, b.disruptionArgs)

	log.Infow("Starting child command with args", "args", args)
	cmd := exec.Command("/usr/local/bin/chaos-injector", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("unable to start background process: %w", err)
	}

	chErr := make(chan error, 1)
	go func() {
		chErr <- cmd.Wait()
	}()

	// Here we want to provide a small time for command to bootstrap
	// if an error occur during this time, we can halt early
	select {
	case <-time.After(1 * time.Second):
	case err := <-chErr:
		if err != nil {
			return nil, fmt.Errorf("an error occurred during startup of background process: %w", err)
		}
	}

	if cmd.Process == nil {
		return nil, fmt.Errorf("no process created, processState exit code is %v", cmd.ProcessState.ExitCode())
	}

	backgroundProcess := backgroundProcess{
		log:            log.With("pid", cmd.Process.Pid),
		processManager: b.processManager,
		pid:            cmd.Process.Pid,
		ticker:         time.NewTicker(1 * time.Second),
		chErr:          chErr,
	}

	backgroundProcess.keepAlive()

	return &backgroundProcess, nil
}

func (d *backgroundProcess) keepAlive() {
	d.log.Debug("new process created, monitoring newly created process every 1s")

	go func() {
		for {
			select {
			case err := <-d.chErr:
				if err != nil {
					d.log.Errorw("background command exited with an error", "err", err)
				} else {
					d.log.Info("background command exited successfully")
				}
				return
			case <-d.ticker.C:
				process, err := d.processManager.Find(d.pid)
				if err != nil {
					d.log.Errorw("an error occured when trying to Find process, stopping background process", "err", err)
					d.Stop()
					return
				}

				if err := d.processManager.Signal(process, syscall.SIGCONT); err != nil {
					d.log.Errorw("an error occured when sending SIGCONT signal to process, stopping background process", "err", err)
					d.Stop()
					return
				}

				d.log.Debug("SIGCONT signal sent to child process")
			}
		}
	}()
}

func (d *backgroundProcess) Stop() {
	if d.ticker == nil {
		return
	}

	d.ticker.Stop()
	d.ticker = nil

	d.log.Infow("sending SIGTERM to background process")

	process, err := d.processManager.Find(d.pid)
	if err != nil {
		d.log.Warnw("process not found, nothing to clean", "err", err)
		return
	}

	err = d.processManager.Signal(process, syscall.SIGTERM)
	if err != nil {
		d.log.Warnw("an error occured while sending interupt signal to process, cleaning ignored", "err", err)
		return
	}

	d.log.Info("background process has been terminated")
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package stress

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	pscpu "github.com/shirou/gopsutil/cpu"
	"go.uber.org/zap"
)

type cpu struct {
	log      *zap.SugaredLogger
	dryRun   bool
	routines int
	tgid     int
}

// NewCPU creates a CPU stresser
func NewCPU(dryRun bool, log *zap.SugaredLogger) (Stresser, error) {
	cores := 0

	// get the total amount of cores
	cpuInfo, err := pscpu.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting cpu info: %w", err)
	}

	for _, info := range cpuInfo {
		cores += int(info.Cores)
	}

	// get thread group ID
	tgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return nil, fmt.Errorf("error retrieving thread group ID: %w", err)
	}

	return cpu{
		dryRun:   dryRun,
		routines: cores,
		log:      log,
		tgid:     tgid,
	}, nil
}

// Stress starts X goroutines loading CPU
func (c cpu) Stress(exit <-chan struct{}) {
	c.log.Infow("starting stresser routines", "routines", c.routines)

	// set real GOMAXPROCS value
	oldValue := runtime.GOMAXPROCS(c.routines)
	c.log.Infow("updated the GOMAXPROCS value", "newValue", c.routines, "oldValue", oldValue)

	// set current task affinity
	cmd := exec.Command("taskset", "-pac", fmt.Sprintf("0-%d", c.routines-1), strconv.Itoa(c.tgid))

	stdoutStderr, err := cmd.CombinedOutput()
	c.log.Infow("set task affinity", "command", strings.Join(cmd.Args, " "), "output", stdoutStderr, "err", err)

	// early exit if dry-run mode is enabled
	if c.dryRun {
		<-exit

		return
	}

	// routines exit channels
	chs := make([]chan struct{}, c.routines)

	// start cpu load generator goroutines
	for routine := 0; routine < c.routines; routine++ {
		go func(ch chan struct{}) {
			// lock the goroutine on the actual thread to nice it
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			// start eating cpu
			for {
				select {
				case <-ch:
					// exit
					return
				default:
					// eat cpu
					for i := uint64(0); i < 18446744073709551615; i++ {
						// noop
					}
				}

				// useful to let other goroutines be scheduled after a loop
				runtime.Gosched()
			}
		}(chs[routine])
	}

	// handle stresser exit
	<-exit

	// close load generator routines before exiting
	for _, ch := range chs {
		ch <- struct{}{}
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package stress

import "runtime"

type cpu struct {
	dryRun   bool
	routines int
}

// NewCPU creates a CPU stresser
func NewCPU(dryRun bool, routines int) Stresser {
	return cpu{
		dryRun:   dryRun,
		routines: routines,
	}
}

// Stress starts X goroutines loading CPU
func (c cpu) Stress(exit <-chan struct{}) {
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

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package stress

import "runtime"

type cpu struct {
	dryRun bool
}

// NewCPU creates a CPU stresser
func NewCPU(dryRun bool) Stresser {
	return cpu{
		dryRun: dryRun,
	}
}

// Stress starts X goroutines loading CPU
func (c cpu) Stress(exit <-chan struct{}) {
	// early exit if dry-run mode is enabled
	if c.dryRun {
		<-exit

		return
	}

	// start eating cpu
	for {
		select {
		case <-exit:
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
}

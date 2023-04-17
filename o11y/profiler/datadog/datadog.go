// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package datadog

import (
	"github.com/DataDog/chaos-controller/o11y/profiler/types"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

// Sink describes a datadog profiler sink
type Sink struct{}

// New initiated datadog profiler sink
func New(cfg types.SinkConfig) (Sink, error) {
	err := profiler.Start(profiler.WithProfileTypes(
		profiler.CPUProfile,
		profiler.HeapProfile,
		profiler.BlockProfile,
		profiler.MutexProfile,
		profiler.GoroutineProfile,
	))

	return Sink{}, err
}

// Stop returns nil
func (d Sink) Stop() {
	profiler.Stop()
}

// GetSinkName returns the name of the sink
func (d Sink) GetSinkName() string {
	return string(types.SinkDriverDatadog)
}

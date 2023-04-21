// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/profiler/types"
)

// Sink describes a no-op profiler sink
type Sink struct{}

// New NOOP Sink
func New() Sink {
	fmt.Println("NOOP Sink: Profiler Started")
	return Sink{}
}

// Stop profiler
func (n Sink) Stop() {
	fmt.Println("NOOP Sink: Profiler Stopped")
}

// GetSinkName returns the name of the sink
func (n Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

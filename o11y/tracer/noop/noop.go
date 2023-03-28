// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/tracer/types"
)

// Sink describes a no-op sink
type Sink struct{}

// New NOOP Sink
func New() *Sink {
	return &Sink{}
}

// Start tracer
func (n *Sink) Start() {
	fmt.Println("NOOP: Empty Sink Started")
}

// Stop tracer
func (n *Sink) Stop() {
	fmt.Println("NOOP: Empty Sink Stopped")
}

// GetSinkName returns the name of the sink
func (n *Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

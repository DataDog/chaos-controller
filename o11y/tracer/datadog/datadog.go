// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package datadog

import (
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/DataDog/chaos-controller/o11y/tracer/types"
)

// Sink describes a datadog tracer sink
type Sink struct{}

// New initiated datadog tracer sink
func New(cfg types.SinkConfig) (Sink, error) {
	var err error

	tracer.Start(
		tracer.WithSampler(tracer.NewRateSampler(cfg.SampleRate)),
	)

	return Sink{}, err
}

// Stop returns nil
func (d Sink) Stop() {
	tracer.Stop()
}

// GetSinkName returns the name of the sink
func (d Sink) GetSinkName() string {
	return string(types.SinkDriverDatadog)
}

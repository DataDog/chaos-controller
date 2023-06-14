// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"github.com/DataDog/chaos-controller/o11y/profiler/types"
	"go.uber.org/zap"
)

// Sink describes a no-op profiler sink
type Sink struct {
	log *zap.SugaredLogger
}

// New NOOP Sink
func New(log *zap.SugaredLogger) Sink {
	log.Debug("NOOP Sink: Profiler Started")

	return Sink{
		log,
	}
}

// Stop profiler
func (n Sink) Stop() {
	n.log.Debug("NOOP Sink: Profiler Stopped")
}

// GetSinkName returns the name of the sink
func (n Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package noop

import (
	"github.com/DataDog/chaos-controller/o11y/tracer/types"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Sink describes a noop tracer sink
type Sink struct {
	log      *zap.SugaredLogger
	provider trace.TracerProvider
}

// New initiated noop tracer sink
func New(log *zap.SugaredLogger) Sink {
	log.Debug("NOOP Sink: Tracer Started")

	provider := trace.NewNoopTracerProvider()

	return Sink{
		log,
		provider,
	}
}

func (d Sink) GetProvider() trace.TracerProvider {
	return d.provider
}

func (Sink) GetLoggableTraceContext(span trace.Span) []interface{} {
	return nil
}

// Stop returns nil
func (d Sink) Stop() error {
	d.log.Debug("NOOP Sink: Tracer Stopped")
	return nil
}

// GetSinkName returns the name of the sink
func (d Sink) GetSinkName() string {
	return string(types.SinkDriverNoop)
}

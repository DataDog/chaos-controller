// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package tracer

import (
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/tracer/datadog"
	"github.com/DataDog/chaos-controller/o11y/tracer/noop"
	"github.com/DataDog/chaos-controller/o11y/tracer/types"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Sink describes a tracer sink
type Sink interface {
	// GetProvider returns the sink's current tracer provider for otel
	GetProvider() trace.TracerProvider
	// GetLoggableTraceContext returns current span context in a log-friendly way.
	GetLoggableTraceContext(trace.Span) []interface{}
	// GetSinkName returns the current sink name
	GetSinkName() string
	// Stop stops the current tracer sink / provider (to flush, prevent memleaks and such)
	Stop() error
}

// GetSink returns a new initiated tracer sink from the given SinkDriver provider
func GetSink(log *zap.SugaredLogger, driver types.SinkDriver) (Sink, error) {
	switch driver {
	case types.SinkDriverDatadog:
		return datadog.New(), nil
	case types.SinkDriverNoop:
		return noop.New(log), nil
	default:
		return nil, fmt.Errorf("unsupported tracer: %s", driver)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package datadog

import (
	"strconv"

	"github.com/DataDog/chaos-controller/o11y/tracer/types"
	"go.opentelemetry.io/otel/trace"

	ddotel "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"
	ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Sink describes a datadog tracer sink
type Sink struct {
	provider *ddotel.TracerProvider
}

// New initiated datadog tracer sink
func New() Sink {
	provider := ddotel.NewTracerProvider(
		ddtracer.WithProfilerCodeHotspots(true),
		ddtracer.WithLogStartup(false),
	)

	return Sink{provider: provider}
}

// GetLoggableTraceContext returns a Datadog-friendly trace context for logs
// It allows to connect logs with corresponding traces and spans.
func (Sink) GetLoggableTraceContext(span trace.Span) []interface{} {
	stringLogContext := []string{
		"dd.trace_id", convertTraceID(span.SpanContext().TraceID().String()),
		"dd.span_id", convertTraceID(span.SpanContext().SpanID().String()),
	}

	interfaceLogContext := make([]interface{}, len(stringLogContext))
	for i, v := range stringLogContext {
		interfaceLogContext[i] = v
	}

	return interfaceLogContext
}

// GetProvider returns this sink's provider for otel
func (d Sink) GetProvider() trace.TracerProvider {
	return d.provider
}

// Stop shutdowns the datadog provider for otel
func (d Sink) Stop() error {
	return d.provider.Shutdown()
}

// GetSinkName returns the name of the sink (datadog)
func (d Sink) GetSinkName() string {
	return string(types.SinkDriverDatadog)
}

// convertTraceID converts a OTLP Trace/Span ID into a DataDog Trace/Span ID
func convertTraceID(id string) string {
	if len(id) < 16 {
		return ""
	}

	if len(id) > 16 {
		id = id[16:]
	}

	intValue, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return ""
	}

	return strconv.FormatUint(intValue, 10)
}

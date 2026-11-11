// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package datadog

import (
	"context"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestLoggableTraceContextHonorsTraceIDWidth(t *testing.T) {
	const fullID = "1234567890abcdef000000000000000f"
	for _, tc := range []struct {
		name, configured, traceID, want string
	}{
		{"enabled", "true", fullID, fullID},
		{"enabled_numeric", "1", fullID, fullID},
		{"disabled", "false", fullID, "15"},
		{"disabled_numeric", "0", fullID, "15"},
		{"sdk_default", "", fullID, fullID},
		{"invalid_uses_sdk_default", "invalid", fullID, fullID},
		{"64_bit_trace_stays_decimal", "true", "0000000000000000000000000000000f", "15"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DD_TRACE_128_BIT_TRACEID_LOGGING_ENABLED", tc.configured)
			traceID, err := trace.TraceIDFromHex(tc.traceID)
			if err != nil {
				t.Fatal(err)
			}
			spanID, err := trace.SpanIDFromHex("00000000000000a5")
			if err != nil {
				t.Fatal(err)
			}
			spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID})
			span := trace.SpanFromContext(trace.ContextWithSpanContext(context.Background(), spanContext))
			got := (Sink{}).GetLoggableTraceContext(span)
			want := []interface{}{"dd.trace_id", tc.want, "dd.span_id", "165"}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("log correlation fields = %#v; want %#v", got, want)
			}
		})
	}
}

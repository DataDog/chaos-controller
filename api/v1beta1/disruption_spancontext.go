// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	annotationSpanContextKey = "SpanContext"
)

var (
	// ErrNoSpanContext is the error returned when the annotation does not contains expected span context key
	ErrNoSpanContext = fmt.Errorf("span context not found in disruption annotations")
)

// SpanContext extracts this disruption's span context, injects it in the given context, then returns it
func (r *Disruption) SpanContext(ctx context.Context) (context.Context, error) {
	var annotation propagation.MapCarrier

	spanContext, ok := r.Annotations[annotationSpanContextKey]
	if !ok {
		return ctx, ErrNoSpanContext
	}

	err := json.Unmarshal([]byte(spanContext), &annotation)
	if err != nil {
		return ctx, fmt.Errorf("unable to unmarshal span context from annotation: %w", err)
	}

	ctx = otel.GetTextMapPropagator().Extract(ctx, annotation)

	return ctx, nil
}

// SetSpanContext store provided spanContext into expected disruption annotation
func (r *Disruption) SetSpanContext(ctx context.Context) error {
	var annotation = make(propagation.MapCarrier)

	otel.GetTextMapPropagator().Inject(ctx, annotation)

	marshaledAnnotation, err := json.Marshal(annotation)
	if err != nil {
		return fmt.Errorf("unable to marshal span context: %w", err)
	}

	r.Annotations[annotationSpanContextKey] = string(marshaledAnnotation)

	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func endWatcherSpan(s trace.Span, err error) {
	if s == nil {
		return
	}

	if err != nil {
		s.RecordError(err)
		s.SetStatus(codes.Error, err.Error())
	} else {
		s.SetStatus(codes.Ok, "")
	}

	s.End()
}

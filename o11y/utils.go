// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package o11y

import (
	"strings"

	"go.uber.org/zap"
)

// ZapDDLogger wraps a ZapSugaredLogger to fit Datadog's logger interface.
// Required to send the tracer and profiler logs through the SugaredLogger.
type ZapDDLogger struct {
	ZapLogger *zap.SugaredLogger
}

// Log sends an error log through the wrapped zap SugaredLogger
func (ddLogger ZapDDLogger) Log(msg string) {
	switch {
	case strings.Contains(msg, "ERROR"):
		ddLogger.ZapLogger.Error(msg)
	case strings.Contains(msg, "WARN"):
		ddLogger.ZapLogger.Warn(msg)
	case strings.Contains(msg, "INFO"):
		ddLogger.ZapLogger.Info(msg)
	default:
		ddLogger.ZapLogger.Debug(msg)
	}
}

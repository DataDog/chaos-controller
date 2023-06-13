// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package o11y

import "go.uber.org/zap"

// ZapDDLogger wraps a ZapSugaredLogger to fit Datadog's logger interface.
type ZapDDLogger struct {
	ZapLogger *zap.SugaredLogger
}

// Log sends an error log through the wrapped zap SugaredLogger
func (ddLogger ZapDDLogger) Log(msg string) {
	ddLogger.ZapLogger.Error(msg)
}

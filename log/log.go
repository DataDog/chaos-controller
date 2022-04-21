// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLogger returns a zap production sugared logger with pre-configured encoder settings
func NewZapLogger(level zapcore.Level) (*zap.SugaredLogger, error) {
	// configure logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(level)
	loggerConfig.EncoderConfig.MessageKey = "message"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder

	// generate logger
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

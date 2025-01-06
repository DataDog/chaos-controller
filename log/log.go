// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// logger's keys
const (
	// Disruption
	DisruptionPrefixKey    = "disruption"
	DisruptionNameKey      = DisruptionPrefixKey + "Name"
	DisruptionNamespaceKey = DisruptionPrefixKey + "Namespace"

	// DisruptionCron
	DisruptionCronPrefixKey    = "disruptionCron"
	DisruptionCronNameKey      = DisruptionCronPrefixKey + "Name"
	DisruptionCronNamespaceKey = DisruptionCronPrefixKey + "Namespace"
)

// NewZapLogger returns a zap production sugared logger with pre-configured encoder settings
func NewZapLogger() (*zap.SugaredLogger, error) {
	// configure logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(zapcore.DebugLevel)
	loggerConfig.EncoderConfig.MessageKey = "message"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.EpochMillisTimeEncoder

	// optionally override the log level from the default based on the LOG_LEVEL env var
	lvl, exists := os.LookupEnv("LOG_LEVEL")
	if exists {
		// parse string, this is built-in feature of zap
		ll, err := zapcore.ParseLevel(lvl)
		// if the log level can be parsed, set the logger to this level
		if err == nil {
			loggerConfig.Level.SetLevel(ll)
		}
	}

	// generate logger
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

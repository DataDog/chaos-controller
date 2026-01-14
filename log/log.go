// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package log

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is used to store logger in context
type contextKey string

const loggerContextKey contextKey = "chaos-controller-logger"

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

// WithLogger stores a logger in the context
func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// FromContext extracts a logger from the context, creating a default logger if not found
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerContextKey).(*zap.SugaredLogger); ok && logger != nil {
		return logger
	}
	// Create default logger if none found
	defaultLogger, err := NewZapLogger()
	if err != nil {
		// If we can't create a logger, use zap's no-op logger to avoid panics
		return zap.NewNop().Sugar()
	}

	return defaultLogger
}

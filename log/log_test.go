// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package log_test

import (
	"os"

	"github.com/DataDog/chaos-controller/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {
	var (
		testLogger *zap.Logger
	)

	BeforeEach(func() {
		testLogger = zaptest.NewLogger(GinkgoT())
	})

	Describe("NewZapLogger", func() {
		Context("with the LOG_LEVEL env var unset", func() {
			It("should use the default (DEBUG) log level", func() {
				// Arrange
				os.Unsetenv("LOG_LEVEL")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the default DEBUG log level")
				Expect(logger.Level()).Should(Equal(zapcore.DebugLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to DEBUG", func() {
			It("should use the DEBUG log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "DEBUG")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the DEBUG log level")
				Expect(logger.Level()).Should(Equal(zapcore.DebugLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to INFO", func() {
			It("should use the INFO log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "INFO")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the INFO log level")
				Expect(logger.Level()).Should(Equal(zapcore.InfoLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to ERROR", func() {
			It("should use the ERROR log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "ERROR")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the ERROR log level")
				Expect(logger.Level()).Should(Equal(zapcore.ErrorLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to warn (lowercase)", func() {
			It("should use the ERROR log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "warn")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the WARN log level")
				Expect(logger.Level()).Should(Equal(zapcore.WarnLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to an invalid log level", func() {
			It("should use the default (DEBUG) log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "Invalid")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the default DEBUG log level")
				Expect(logger.Level()).Should(Equal(zapcore.DebugLevel))
			})
		})

		Context("with the LOG_LEVEL env var set to an empty string", func() {
			It("should use the INFO log level", func() {
				// Arrange
				os.Setenv("LOG_LEVEL", "")
				logger, err := log.NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the INFO log level")
				Expect(logger.Level()).Should(Equal(zapcore.InfoLevel))
			})
		})

	})

	Describe("WithLogger", func() {
		Context("when adding a logger to context", func() {
			It("should store the logger correctly", func(ctx SpecContext) {
				// Arrange
				logger := testLogger.Sugar()

				// Act
				ctxWithLogger := log.WithLogger(ctx, logger)

				// Assert
				By("storing the logger in the context and being able to retrieve it")
				Expect(ctxWithLogger).ToNot(BeNil())
				retrievedLogger := log.FromContext(ctxWithLogger)
				Expect(retrievedLogger).To(Equal(logger))
			})
		})
	})

	Describe("FromContext", func() {
		Context("when context has a logger", func() {
			It("should return the contextual logger", func(ctx SpecContext) {
				// Arrange
				logger := testLogger.Sugar()
				ctxWithLogger := log.WithLogger(ctx, logger)

				// Act
				retrievedLogger := log.FromContext(ctxWithLogger)

				// Assert
				By("returning the contextual logger from context")
				Expect(retrievedLogger).To(Equal(logger))
			})
		})

		Context("when context has no logger", func() {
			It("should return a default logger", func(ctx SpecContext) {
				// Act
				retrievedLogger := log.FromContext(ctx)

				// Assert
				By("returning a default logger")
				Expect(retrievedLogger).ToNot(BeNil())
			})
		})
	})
})

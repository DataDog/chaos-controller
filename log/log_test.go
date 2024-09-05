// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package log_test

import (
	"os"

	. "github.com/DataDog/chaos-controller/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Zap Logger", func() {
	Describe("NewZapLogger", func() {
		Context("with the LOG_LEVEL env var unset", func() {
			It("should use the default (DEBUG) log level", func() {
				// Arrange
				os.Unsetenv("LOG_LEVEL")
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

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
				logger, err := NewZapLogger()

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("use the INFO log level")
				Expect(logger.Level()).Should(Equal(zapcore.InfoLevel))
			})
		})

	})

})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package services_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	// +kubebuilder:scaffold:imports
)

var logger *zap.SugaredLogger

var _ = BeforeSuite(func() {
	// Arrange
	logger = zaptest.NewLogger(GinkgoT()).Sugar()
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Services Suite")
}

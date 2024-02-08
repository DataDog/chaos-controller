// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package targetselector

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var logger *zap.SugaredLogger

func TestTargetselector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Targetselector Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	logger = zaptest.NewLogger(GinkgoT()).Sugar()
})

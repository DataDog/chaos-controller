// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"

	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
)

func TestV1Beta1(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "v1beta1 Suite")
}

var _ = BeforeSuite(func() {
	By("initializing disruption_webhook global variables")

	baseLogger, err := zap.NewDevelopmentConfig().Build()
	Expect(err).To(BeNil())

	logger = baseLogger.Sugar()
	chaosNamespace = "chaos-engineering"

	metricsSink = noop.New()
})

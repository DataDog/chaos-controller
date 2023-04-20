// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"os"
	"testing"

	"github.com/DataDog/chaos-controller/o11y/metrics"
	metricstypes "github.com/DataDog/chaos-controller/o11y/metrics/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var (
	log *zap.SugaredLogger
	ms  metrics.Sink
)

var _ = BeforeSuite(func() {
	log = zaptest.NewLogger(GinkgoT()).Sugar()
	os.Setenv("STATSD_URL", "localhost:54321")
	ms, _ = metrics.GetSink(metricstypes.SinkConfig{SinkDriver: string(metricstypes.SinkDriverNoop), SinkApp: string(metricstypes.SinkAppInjector)})
})

var _ = AfterSuite(func() {
	os.Unsetenv("STATSD_URL")
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

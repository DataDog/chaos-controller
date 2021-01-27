// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector_test

import (
	"os"
	"testing"

	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger
var ms metrics.Sink

var _ = BeforeSuite(func() {
	z, _ := zap.NewDevelopment()
	log = z.Sugar()
	os.Setenv("STATSD_URL", "localhost:54321")
	ms, _ = metrics.GetSink(types.SinkDriverNoop, types.SinkAppInjector)
})

var _ = AfterSuite(func() {
	os.Unsetenv("STATSD_URL")
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build integration

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

	// ensure testcontainers-go is vendored
	_ "github.com/testcontainers/testcontainers-go"
)

var (
	integrationLog *zap.SugaredLogger
	integrationMS  metrics.Sink
)

func TestIntegration(t *testing.T) {
	if _, ok := os.LookupEnv("CHAOS_INJECTOR_MOUNT_PROC"); !ok {
		t.Skip("CHAOS_INJECTOR_MOUNT_PROC not set — run via `make test-integration`")
	}

	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		t.Skip("/var/run/docker.sock not accessible — run via `make test-integration`")
	}

	integrationLog = zaptest.NewLogger(t).Sugar()
	integrationMS, _ = metrics.GetSink(integrationLog, metricstypes.SinkDriverNoop, metricstypes.SinkAppInjector)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Integration Suite")
}

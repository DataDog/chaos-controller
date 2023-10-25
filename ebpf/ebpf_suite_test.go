// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ebpf_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var log *zap.SugaredLogger

var _ = BeforeSuite(func() {
	log = zaptest.NewLogger(GinkgoT()).Sugar()
})

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "eBPF Suite")
}

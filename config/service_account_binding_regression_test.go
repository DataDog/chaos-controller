// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var _ = Describe("scalar injector service-account configuration", func() {
	BeforeEach(func() { viper.Reset() })
	AfterEach(func() { viper.Reset() })

	DescribeTable("loads defaults and applies flag precedence",
		func(body, flagValue, expected string) {
			path := filepath.Join(GinkgoT().TempDir(), "config.yaml")
			Expect(os.WriteFile(path, []byte(body), 0600)).To(Succeed())
			args := []string{"--config", path}
			if flagValue != "" {
				args = append(args, "--injector-service-account="+flagValue)
			}
			cfg, err := config.New(nil, zap.NewNop().Sugar(), args)
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Injector.ServiceAccount).To(Equal(expected))
		},
		Entry("empty configuration retains the default", "{}\n", "", "chaos-injector"),
		Entry("unrelated configuration retains the default", "controller:\n  metricsBindAddr: ':9090'\n", "", "chaos-injector"),
		Entry("flag works without a file serviceAccount key", "{}\n", "cli-injector", "cli-injector"),
		Entry("file supplies a scalar account", "injector:\n  serviceAccount: file-injector\n", "", "file-injector"),
		Entry("flag overrides the file account", "injector:\n  serviceAccount: file-injector\n", "cli-injector", "cli-injector"),
	)
})

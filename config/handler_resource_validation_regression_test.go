// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var _ = Describe("effective handler resource validation", func() {
	BeforeEach(func() { viper.Reset() })
	AfterEach(func() { viper.Reset() })

	DescribeTable("rejects invalid quantities after loading configuration and flags",
		func(field string, fromFile bool) {
			args := []string{"--handler-enabled=true"}
			if fromFile {
				path := filepath.Join(GinkgoT().TempDir(), "config.yaml")
				body := fmt.Sprintf("injector:\n  serviceAccount: chaos-injector\nhandler:\n  %s: not-a-quantity\n", field)
				Expect(os.WriteFile(path, []byte(body), 0600)).To(Succeed())
				args = append(args, "--config", path)
			} else {
				args = append(args, "--handler-"+field+"=not-a-quantity")
			}
			_, err := config.New(nil, zap.NewNop().Sugar(), args)
			Expect(err).To(HaveOccurred())
		},
		Entry("CPU flag", "cpu", false),
		Entry("memory flag", "memory", false),
		Entry("CPU configuration file", "cpu", true),
		Entry("memory configuration file", "memory", true),
	)

	It("validates flag overrides rather than superseded file quantities", func() {
		path := filepath.Join(GinkgoT().TempDir(), "config.yaml")
		body := "injector:\n  serviceAccount: chaos-injector\nhandler:\n  cpu: not-a-quantity\n  memory: not-a-quantity\n"
		Expect(os.WriteFile(path, []byte(body), 0600)).To(Succeed())
		cfg, err := config.New(nil, zap.NewNop().Sugar(), []string{
			"--config", path,
			"--handler-enabled=true",
			"--handler-cpu=125m",
			"--handler-memory=64Mi",
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(cfg.Handler.CPU).To(Equal("125m"))
		Expect(cfg.Handler.Memory).To(Equal("64Mi"))
	})
})

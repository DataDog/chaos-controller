// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package config_test

import (
	"time"

	"github.com/DataDog/chaos-controller/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var logger *zap.SugaredLogger

var _ = Describe("Config", func() {
	Context("New", func() {
		logger = zaptest.NewLogger(GinkgoT()).Sugar()

		Context("invalid config", func() {
			It("fails with a missing path", func() {
				_, err := config.New(logger, []string{"--config"})
				Expect(err).Should(MatchError("unable to retrieve configuration parse from provided flag: flag needs an argument: --config"))
			})

			It("fails with an invalid path", func() {
				_, err := config.New(logger, []string{"--config", "invalid-path/invalid-file.yaml"})
				Expect(err).Should(MatchError("error loading configuration file: open invalid-path/invalid-file.yaml: no such file or directory"))
			})

			It("fails with an invalid config file", func() {
				_, err := config.New(logger, []string{"--config", "testdata/invalid.yaml"})
				Expect(err).Should(MatchError(ContainSubstring("error loading configuration file: While parsing config: yaml: unmarshal errors:")))
			})

			It("fails with a defaultDuration greater than the maxDuration", func() {
				_, err := config.New(logger, []string{"--config", "testdata/default-duration-too-big.yaml"})
				Expect(err).Should(MatchError("defaultDuration must be less than or equal to maxDuration"))
			})
		})

		Context("without configuration", func() {
			It("succeed with default values", func() {
				v, err := config.New(logger, []string{})
				Expect(err).ToNot(HaveOccurred())

				By("overriding controller values")
				Expect(v.Controller.DeleteOnly).To(BeFalse())
				Expect(v.Controller.DefaultDuration).To(Equal(1 * time.Hour))
				Expect(v.Controller.EnableSafeguards).To(BeTrue())
				Expect(v.Controller.EnableObserver).To(BeTrue())
				Expect(v.Controller.MetricsBindAddr).To(Equal(":8080"))
				Expect(v.Controller.LeaderElection).To(BeFalse())
				Expect(v.Controller.MetricsSink).To(Equal("noop"))
				Expect(v.Controller.ProfilerSink).To(Equal("noop"))
				Expect(v.Controller.UserInfoHook).To(BeTrue())

				By("overriding controller notifier values")
				Expect(v.Controller.Notifiers.Common.ClusterName).To(BeEmpty())
				Expect(v.Controller.Notifiers.Noop.Enabled).To(BeFalse())
				Expect(v.Controller.Notifiers.Slack.Enabled).To(BeFalse())
				Expect(v.Controller.Notifiers.HTTP.Enabled).To(BeFalse())
				Expect(v.Controller.Notifiers.Datadog.Enabled).To(BeFalse())

				By("overriding controller notifier slack values")
				Expect(v.Controller.Notifiers.Slack.MirrorSlackChannelID).To(BeEmpty())
				Expect(v.Controller.Notifiers.Slack.TokenFilepath).To(BeEmpty())

				By("overriding controller notifier http values")
				Expect(v.Controller.Notifiers.HTTP.Headers).To(BeEmpty())
				Expect(v.Controller.Notifiers.HTTP.HeadersFilepath).To(BeEmpty())
				Expect(v.Controller.Notifiers.HTTP.URL).To(BeEmpty())

				By("overriding controller cloudProvider values")
				Expect(v.Controller.CloudProviders.DisableAll).To(BeFalse())
				Expect(v.Controller.CloudProviders.PullInterval).To(Equal(24 * time.Hour))
				Expect(v.Controller.CloudProviders.AWS.Enabled).To(BeTrue())
				Expect(v.Controller.CloudProviders.AWS.IPRangesURL).To(BeEmpty())
				Expect(v.Controller.CloudProviders.GCP.Enabled).To(BeTrue())
				Expect(v.Controller.CloudProviders.GCP.IPRangesURL).To(BeEmpty())
				Expect(v.Controller.CloudProviders.Datadog.Enabled).To(BeTrue())
				Expect(v.Controller.CloudProviders.Datadog.IPRangesURL).To(BeEmpty())

				By("overriding controller safeMode values")
				Expect(v.Controller.SafeMode.Enable).To(BeTrue())
				Expect(v.Controller.SafeMode.ClusterThreshold).To(Equal(66))
				Expect(v.Controller.SafeMode.Environment).To(BeEmpty())
				Expect(v.Controller.SafeMode.NamespaceThreshold).To(Equal(80))

				By("overriding controller webhook values")
				Expect(v.Controller.Webhook.CertDir).To(BeEmpty())
				Expect(v.Controller.Webhook.Host).To(BeEmpty())
				Expect(v.Controller.Webhook.Port).To(Equal(9443))

				By("overriding injector global values")
				Expect(v.Injector.Image).To(Equal("chaos-injector"))
				Expect(v.Injector.ImagePullSecrets).To(BeEmpty())
				Expect(v.Injector.ServiceAccount).To(Equal("chaos-injector"))
				Expect(v.Injector.ChaosNamespace).To(Equal("chaos-engineering"))

				By("overriding handler global values")
				Expect(v.Handler.Enabled).To(BeFalse())
				Expect(v.Handler.Image).To(Equal("chaos-handler"))
				Expect(v.Handler.Timeout).To(Equal(time.Minute))
			})
		})

		Context("with configuration file", func() {
			It("succeed with overriden values", func() {
				v, err := config.New(logger, []string{"--config", "testdata/local.yaml"})
				Expect(err).ToNot(HaveOccurred())

				By("overriding controller values")
				Expect(v.Controller.DeleteOnly).To(BeTrue())
				Expect(v.Controller.DefaultDuration).To(Equal(3 * time.Minute))
				Expect(v.Controller.EnableSafeguards).To(BeFalse())
				Expect(v.Controller.EnableObserver).To(BeFalse())
				Expect(v.Controller.MetricsBindAddr).To(Equal("127.0.0.1:8080"))
				Expect(v.Controller.LeaderElection).To(BeTrue())
				Expect(v.Controller.MetricsSink).To(Equal("datadog"))
				Expect(v.Controller.ProfilerSink).To(Equal("notdatadog"))
				Expect(v.Controller.UserInfoHook).To(BeTrue())

				By("overriding controller notifier values")
				Expect(v.Controller.Notifiers.Common.ClusterName).To(Equal("some-cluster-name"))
				Expect(v.Controller.Notifiers.Noop.Enabled).To(BeTrue())
				Expect(v.Controller.Notifiers.Slack.Enabled).To(BeTrue())
				Expect(v.Controller.Notifiers.HTTP.Enabled).To(BeTrue())
				Expect(v.Controller.Notifiers.Datadog.Enabled).To(BeTrue())

				By("overriding controller notifier slack values")
				Expect(v.Controller.Notifiers.Slack.MirrorSlackChannelID).To(Equal("WOPIEQQET"))
				Expect(v.Controller.Notifiers.Slack.TokenFilepath).To(Equal("/random-file-path"))

				By("overriding controller notifier http values")
				Expect(v.Controller.Notifiers.HTTP.Headers).To(Equal([]string{"a", "b", "c"}))
				Expect(v.Controller.Notifiers.HTTP.HeadersFilepath).To(Equal("/header-file-path/below/me"))
				Expect(v.Controller.Notifiers.HTTP.URL).To(Equal("https://example.com/webhook"))

				By("overriding controller cloudProvider values")
				Expect(v.Controller.CloudProviders.DisableAll).To(BeTrue())
				Expect(v.Controller.CloudProviders.PullInterval).To(Equal(15 * time.Minute))
				Expect(v.Controller.CloudProviders.AWS.Enabled).To(BeFalse())
				Expect(v.Controller.CloudProviders.AWS.IPRangesURL).To(Equal("https://example.com/aws-ip-ranges-url"))
				Expect(v.Controller.CloudProviders.GCP.Enabled).To(BeFalse())
				Expect(v.Controller.CloudProviders.GCP.IPRangesURL).To(Equal("https://example.com/gcp-ip-ranges-url"))
				Expect(v.Controller.CloudProviders.Datadog.Enabled).To(BeFalse())
				Expect(v.Controller.CloudProviders.Datadog.IPRangesURL).To(Equal("https://example.com/datadog-ip-ranges-url"))

				By("overriding controller safeMode values")
				Expect(v.Controller.SafeMode.Enable).To(BeFalse())
				Expect(v.Controller.SafeMode.ClusterThreshold).To(Equal(61))
				Expect(v.Controller.SafeMode.Environment).To(Equal("my-safe-env-value"))
				Expect(v.Controller.SafeMode.NamespaceThreshold).To(Equal(79))

				By("overriding controller webhook values")
				Expect(v.Controller.Webhook.CertDir).To(Equal("/var/data/cert/cert.pem"))
				Expect(v.Controller.Webhook.Host).To(Equal("another-host"))
				Expect(v.Controller.Webhook.Port).To(Equal(7443))

				By("overriding injector global values")
				Expect(v.Injector.Image).To(Equal("datadog.io/chaos-injector:not-latest"))
				Expect(v.Injector.ImagePullSecrets).To(Equal("some-pull-secret"))
				Expect(v.Injector.ServiceAccount).To(Equal("chaos-injector-custom-sa"))
				Expect(v.Injector.ChaosNamespace).To(Equal("chaos-engineering-custom-ns"))

				By("overriding handler global values")
				Expect(v.Handler.Enabled).To(BeTrue())
				Expect(v.Handler.Image).To(Equal("other.io/chaos-handler:not-latest-again"))
				Expect(v.Handler.Timeout).To(Equal(time.Minute + 30*time.Second))
			})
		})

		Context("with configuration file and flag", func() {
			It("succeed with values from flags", func() {
				v, err := config.New(logger, []string{"--config", "testdata/local.yaml", "--notifiers-common-clustername", "provided-by-command-flag-cluster-name"})
				Expect(err).ToNot(HaveOccurred())

				Expect(v.Controller.Notifiers.Common.ClusterName).To(Equal("provided-by-command-flag-cluster-name"))
			})
		})
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package tags_test

import (
	"github.com/DataDog/chaos-controller/o11y/tags"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FormatTag", func() {
	DescribeTable("should format key:value pairs correctly",
		func(key, value, expected string) {
			// Act
			result := tags.FormatTag(key, value)

			// Assert
			Expect(result).To(Equal(expected))
		},
		Entry("basic key-value pair", "name", "test", "name:test"),
		Entry("empty value", "key", "", "key:"),
		Entry("empty key", "", "value", ":value"),
		Entry("both empty", "", "", ":"),
		Entry("numeric value", "count", "42", "count:42"),
		Entry("boolean value", "enabled", "true", "enabled:true"),
		Entry("key with underscore", "disruption_name", "chaos-test", "disruption_name:chaos-test"),
		Entry("key with namespace", "disruptionNamespace", "default", "disruptionNamespace:default"),
		Entry("value with special chars", "message", "test-value_123", "message:test-value_123"),
		Entry("kubernetes resource name", "podName", "my-pod-12345", "podName:my-pod-12345"),
		Entry("namespace value", "namespace", "chaos-engineering", "namespace:chaos-engineering"),
		Entry("status value", "status", "success", "status:success"),
		Entry("kind value", "kind", "pod", "kind:pod"),
		Entry("target value", "target", "deployment/nginx", "target:deployment/nginx"),
		Entry("long values", "description", "this-is-a-very-long-description-for-testing", "description:this-is-a-very-long-description-for-testing"),
	)

	Context("edge cases", func() {
		It("should handle keys with colons", func() {
			// Act
			result := tags.FormatTag("key:with:colons", "value")

			// Assert
			Expect(result).To(Equal("key:with:colons:value"))
		})

		It("should handle values with colons", func() {
			// Act
			result := tags.FormatTag("url", "http://example.com:8080")

			// Assert
			Expect(result).To(Equal("url:http://example.com:8080"))
		})

		It("should handle both key and value with colons", func() {
			// Act
			result := tags.FormatTag("key:colon", "value:colon")

			// Assert
			Expect(result).To(Equal("key:colon:value:colon"))
		})
	})

	Context("observability use cases", func() {
		DescribeTable("should format common observability tags correctly",
			func(key, value, expected string) {
				// Act
				result := tags.FormatTag(key, value)

				// Assert
				Expect(result).To(Equal(expected))
			},
			Entry("disruption name", "disruptionName", "test-disruption", "disruptionName:test-disruption"),
			Entry("disruption namespace", "disruptionNamespace", "chaos-test", "disruptionNamespace:chaos-test"),
			Entry("disruption cron name", "disruptionCronName", "daily-chaos", "disruptionCronName:daily-chaos"),
			Entry("disruption rollout name", "disruptionRolloutName", "rolling-chaos", "disruptionRolloutName:rolling-chaos"),
			Entry("pod name", "podName", "nginx-deployment-abc123", "podName:nginx-deployment-abc123"),
			Entry("node name", "nodeName", "worker-node-01", "nodeName:worker-node-01"),
			Entry("target kind", "targetKind", "deployment", "targetKind:deployment"),
			Entry("event type", "event", "add", "event:add"),
			Entry("watcher name", "watcher", "pod-watcher", "watcher:pod-watcher"),
			Entry("chaos pod name", "chaosPodName", "chaos-injector-xyz789", "chaosPodName:chaos-injector-xyz789"),
			Entry("app tag", "app", "chaos-controller", "app:chaos-controller"),
			Entry("service tag", "service", "web-server", "service:web-server"),
			Entry("team tag", "team", "chaos-engineering", "team:chaos-engineering"),
			Entry("status tag", "status", "success", "status:success"),
			Entry("kind tag", "kind", "networkDisruption", "kind:networkDisruption"),
			Entry("level tag", "level", "pod", "level:pod"),
		)
	})

	Context("metrics integration", func() {
		It("should produce tags suitable for metrics systems", func() {
			// Act: Common patterns used in the codebase
			tags := []string{
				tags.FormatTag("disruptionName", "test-chaos"),
				tags.FormatTag("disruptionNamespace", "default"),
				tags.FormatTag("status", "success"),
				tags.FormatTag("kind", "pod"),
			}

			// Assert
			Expect(tags).To(ConsistOf(
				"disruptionName:test-chaos",
				"disruptionNamespace:default",
				"status:success",
				"kind:pod",
			))
		})

		It("should produce tags suitable for event notifiers", func() {
			// Act: Event notifier patterns
			tags := []string{
				tags.FormatTag("disruption_name", "network-test"),
				tags.FormatTag("target_name", "nginx-pod"),
				tags.FormatTag("team", "platform"),
				tags.FormatTag("service", "web"),
			}

			// Assert
			Expect(tags).To(ConsistOf(
				"disruption_name:network-test",
				"target_name:nginx-pod",
				"team:platform",
				"service:web",
			))
		})
	})
})

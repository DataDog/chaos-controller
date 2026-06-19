// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-controller/types"
)

var _ = Describe("DisruptionInjectionStatus", func() {
	DescribeTable(
		"NotFullyInjected",
		func(s types.DisruptionInjectionStatus, expectCanInject bool) {
			Expect(s.NotFullyInjected()).To(Equal(expectCanInject))
		},
		Entry("not injected returns true", types.DisruptionInjectionStatusNotInjected, true),
		Entry("partially injected returns true", types.DisruptionInjectionStatusPartiallyInjected, true),
		Entry("injected returns false", types.DisruptionInjectionStatusInjected, false),
		Entry("paused partially injected returns true", types.DisruptionInjectionStatusPausedPartiallyInjected, true),
		Entry("paused injected returns true", types.DisruptionInjectionStatusPausedInjected, true),
		Entry("previously not injected returns false", types.DisruptionInjectionStatusPreviouslyNotInjected, false),
		Entry("previously partially injected returns false", types.DisruptionInjectionStatusPreviouslyPartiallyInjected, false),
		Entry("previously injected returns false", types.DisruptionInjectionStatusPreviouslyInjected, false),
	)

	DescribeTable(
		"Previously",
		func(s types.DisruptionInjectionStatus, expect bool) {
			Expect(s.Previously()).To(Equal(expect))
		},
		Entry("initial returns false", types.DisruptionInjectionStatusInitial, false),
		Entry("not injected returns false", types.DisruptionInjectionStatusNotInjected, false),
		Entry("partially injected returns false", types.DisruptionInjectionStatusPartiallyInjected, false),
		Entry("injected returns false", types.DisruptionInjectionStatusInjected, false),
		Entry("previously not injected returns true", types.DisruptionInjectionStatusPreviouslyNotInjected, true),
		Entry("previously partially injected returns true", types.DisruptionInjectionStatusPreviouslyPartiallyInjected, true),
		Entry("previously injected returns true", types.DisruptionInjectionStatusPreviouslyInjected, true),
	)

	DescribeTable(
		"NeverInjected",
		func(s types.DisruptionInjectionStatus, expect bool) {
			Expect(s.NeverInjected()).To(Equal(expect))
		},
		Entry("initial returns true", types.DisruptionInjectionStatusInitial, true),
		Entry("not injected returns true", types.DisruptionInjectionStatusNotInjected, true),
		Entry("partially injected returns false", types.DisruptionInjectionStatusPartiallyInjected, false),
		Entry("injected returns false", types.DisruptionInjectionStatusInjected, false),
		Entry("previously injected returns false", types.DisruptionInjectionStatusPreviouslyInjected, false),
	)
})

var _ = Describe("DisruptionKindName", func() {
	DescribeTable(
		"String",
		func(k types.DisruptionKindName, expected string) {
			Expect(k.String()).To(Equal(expected))
		},
		Entry("network", types.DisruptionKindName(types.DisruptionKindNetworkDisruption), string(types.DisruptionKindNetworkDisruption)),
		Entry("node failure", types.DisruptionKindName(types.DisruptionKindNodeFailure), string(types.DisruptionKindNodeFailure)),
		Entry("cpu pressure", types.DisruptionKindName(types.DisruptionKindCPUPressure), string(types.DisruptionKindCPUPressure)),
	)
})

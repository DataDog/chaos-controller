// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
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
})

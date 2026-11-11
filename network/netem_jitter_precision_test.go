// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("netem microsecond jitter encoding",
	func(delay, jitter time.Duration, expectedDelay, expectedJitter string) {
		executor := newExecutorMock(GinkgoT())
		executor.EXPECT().Run([]string{
			"qdisc", "add", "dev", "test0", "root", "netem",
			"delay", expectedDelay, expectedJitter,
		}).Return(0, "", nil).Once()
		controller := tc{executer: executor}
		Expect(controller.AddNetem([]string{"test0"}, "root", "", delay, jitter, 0, 0, 0)).To(Succeed())
	},
	Entry("default jitter on a one-millisecond delay", time.Millisecond, 100*time.Microsecond, "1000us", "100us"),
	Entry("one percent jitter on a one-millisecond delay", time.Millisecond, 10*time.Microsecond, "1000us", "10us"),
	Entry("fractional-millisecond jitter", 15*time.Millisecond, 1500*time.Microsecond, "15000us", "1500us"),
	Entry("maximum admitted delay and jitter", 60*time.Second, 60*time.Second, "60000000us", "60000000us"),
)

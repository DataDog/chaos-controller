// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	. "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = DescribeTable("disk failure rejects active/dormant pulsing",
	func(level types.DisruptionLevel) {
		count := intstr.FromInt(1)
		spec := DisruptionSpec{
			Level:       level,
			Count:       &count,
			DiskFailure: &DiskFailureSpec{Paths: []string{"/data"}},
			Pulse: &DisruptionPulse{
				ActiveDuration:  "1s",
				DormantDuration: "1s",
			},
		}
		err := spec.ValidateSelectorsOptional(false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pulse is only compatible"))
	},
	Entry("pod level", types.DisruptionLevelPod),
	Entry("node level", types.DisruptionLevelNode),
)

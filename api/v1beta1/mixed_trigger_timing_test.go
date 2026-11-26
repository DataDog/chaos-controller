// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Mixed disruption trigger timing", func() {
	DescribeTable("starts the duration no earlier than pod creation", func(injectDelay, expectedDelay time.Duration) {
		created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		count := intstr.FromInt(1)
		disruption := &Disruption{
			ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(created)},
			Spec: DisruptionSpec{
				Count: &count, Level: "pod", Duration: "10m",
				Selector: labels.Set{"app": "example"},
				Network:  &NetworkDisruptionSpec{Drop: 100},
				Triggers: &DisruptionTriggers{
					CreatePods: DisruptionTrigger{Offset: "1h"},
					Inject:     DisruptionTrigger{NotBefore: metav1.NewTime(created.Add(injectDelay))},
				},
			},
		}
		Expect(disruption.Spec.Validate()).To(Succeed())
		Expect(disruption.TimeToCreatePods()).To(Equal(created.Add(time.Hour)))
		Expect(disruption.TimeToInject()).To(Equal(created.Add(expectedDelay)))
		deadline := disruption.TimeToInject().Add(disruption.Spec.Duration.Duration())
		Expect(deadline.Before(disruption.TimeToCreatePods())).To(BeFalse())
	},
		Entry("absolute injection bound precedes relative pod creation", 5*time.Minute, time.Hour),
		Entry("absolute injection bound follows relative pod creation", 90*time.Minute, 90*time.Minute),
	)
})

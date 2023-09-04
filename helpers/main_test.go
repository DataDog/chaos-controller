// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package helpers

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/builders"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Label Selector Validation", func() {
	Context("validating an empty label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{}
			Expect(ValidateLabelSelector(selector.AsSelector())).ToNot(Succeed())
		})
	})
	Context("validating a good label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating special characters in label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("should succeed", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(ValidateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
})

var _ = Describe("Inject and CreatePods Trigger tests", func() {
	var creationTimestamp time.Time

	BeforeEach(func() {
		creationTimestamp = time.Now()
	})

	Context("TimeToCreatePods", func() {
		It("should return creationTimestamp if triggers is nil", func() {
			var triggers v1beta1.DisruptionTriggers

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return creationTimestamp if triggers.createPods is nil", func() {
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return createPods.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if createPods.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute * 5)
			triggers := v1beta1.DisruptionTriggers{
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "5m",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(offsetTime))
		})
	})

	Context("TimeToInject", func() {
		It("should return creationTimestamp if triggers is nil", func() {
			var triggers v1beta1.DisruptionTriggers

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return triggers.createPods if triggers.inject is nil", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return inject.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "1",
				},
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "2m",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if inject.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "1m",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(offsetTime))
		})
	})
})

var _ = DescribeTable(
	"DisruptionTerminationStatus",
	func(disruption *builders.DisruptionBuilder, pods builders.PodsBuilder, expectTerminationStatus TerminationStatus) {
		Expect(DisruptionTerminationStatus(disruption.Build(), pods.Build())).To(Equal(expectTerminationStatus))
	},
	Entry(
		"not yet created disruption IS NOT terminated",
		builders.NewDisruptionBuilder().Reset(),
		nil,
		TSNotTerminated,
	),
	Entry(
		"1s before deadline, disruption IS NOT terminated",
		builders.NewDisruptionBuilder().WithCreation(time.Minute-time.Second),
		builders.NewPodsBuilder(),
		TSNotTerminated,
	),
	Entry(
		"1s after deadline, disruption IS definitively terminated",
		builders.NewDisruptionBuilder().WithCreation(time.Minute+time.Second),
		builders.NewPodsBuilder(),
		TSDefinitivelyTerminated,
	),
	Entry(
		"half duration disruption IS NOT terminated",
		builders.NewDisruptionBuilder(),
		builders.NewPodsBuilder(),
		TSNotTerminated,
	),
	Entry(
		"at deadline, disruption IS definitively terminated (however even ns before it is not)",
		builders.NewDisruptionBuilder().WithCreation(time.Minute),
		builders.NewPodsBuilder(),
		TSDefinitivelyTerminated,
	),
	Entry(
		"deleted disruption IS definitively terminated",
		builders.NewDisruptionBuilder().WithCreation(time.Minute).WithDeletion(),
		builders.NewPodsBuilder(),
		TSDefinitivelyTerminated,
	),
	Entry(
		"one chaos pod exited out of two IS NOT terminated",
		builders.NewDisruptionBuilder(),
		builders.NewPodsBuilder().One().Terminated().Parent(),
		TSNotTerminated,
	),
	Entry(
		"all chaos pods exited IS temporarily terminated",
		builders.NewDisruptionBuilder(),
		builders.NewPodsBuilder().One().Terminated().Parent().Two().Terminated().Parent(),
		TSTemporarilyTerminated,
	),
	Entry(
		"no pod injected is temporarily terminated",
		builders.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusInjected),
		nil,
		TSTemporarilyTerminated,
	),
	Entry(
		"no pod partially injected is temporarily terminated",
		builders.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusPartiallyInjected),
		nil,
		TSTemporarilyTerminated,
	),
	Entry(
		"no pod NOT injected is not terminated",
		builders.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusNotInjected),
		nil,
		TSNotTerminated,
	),
	Entry(
		"no pod initial injection status is not terminated",
		builders.NewDisruptionBuilder(),
		nil,
		TSNotTerminated,
	),
)

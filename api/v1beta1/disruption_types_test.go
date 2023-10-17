// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1_test

import (
	"sort"
	"time"

	. "github.com/DataDog/chaos-controller/api/v1beta1"
	builderstest "github.com/DataDog/chaos-controller/builderstest"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("TargetInjections", func() {
	Describe("GetTargetNames", func() {
		var targetInjections TargetInjections

		Context("with three targets", func() {
			BeforeEach(func() {
				targetInjections = TargetInjections{"target-1": {}, "target-2": {}, "target-3": {}}
			})

			It("should return the list of targets name", func() {
				targetNames := targetInjections.GetTargetNames()
				sort.Strings(targetNames)
				Expect(targetNames).Should(BeEquivalentTo([]string{"target-1", "target-2", "target-3"}))
			})
		})

		Context("without targets", func() {
			BeforeEach(func() {
				targetInjections = TargetInjections{}
			})

			It("should return the list of targets name", func() {
				Expect(targetInjections.GetTargetNames()).Should(BeEquivalentTo([]string{}))
			})
		})
	})
})

var _ = Describe("AdvancedSelectorsToRequirements", func() {
	Context("valid advanced selectors", func() {
		It("should return valid requirements", func() {
			advancedSelectors := []metav1.LabelSelectorRequirement{
				{
					Key:      "service",
					Operator: "NotIn",
					Values:   []string{"foo", "bar"},
				},
				{
					Key:      "app",
					Operator: "Exists",
					Values:   nil,
				},
			}

			req1, err := labels.NewRequirement("service", selection.NotIn, []string{"foo", "bar"})
			Expect(err).ShouldNot(HaveOccurred())

			req2, err := labels.NewRequirement("app", selection.Exists, nil)
			Expect(err).ShouldNot(HaveOccurred())

			expected := []labels.Requirement{
				*req1,
				*req2,
			}

			req, err := AdvancedSelectorsToRequirements(advancedSelectors)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(req).Should(Equal(expected))
		})
	})

	Context("invalid operator", func() {
		It("should return an error", func() {
			advancedSelectors := []metav1.LabelSelectorRequirement{
				{
					Key:      "app",
					Operator: "CouldBe",
					Values:   []string{"foobaz"},
				},
			}

			_, err := AdvancedSelectorsToRequirements(advancedSelectors)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("error parsing advanced selector operator CouldBe: must be either In, NotIn, Exists or DoesNotExist"))
		})
	})

	Context("invalid values", func() {
		It("should return an error", func() {
			advancedSelectors := []metav1.LabelSelectorRequirement{
				{
					Key:      "app",
					Operator: "In",
					Values:   []string{"*", "{hash}"},
				},
			}

			_, err := AdvancedSelectorsToRequirements(advancedSelectors)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("error parsing given advanced selector to requirements"))
		})
	})
})

var _ = Describe("Check if a target exist into DisruptionStatus targets list", func() {
	var disruptionStatus DisruptionStatus

	BeforeEach(func() {
		disruptionStatus = DisruptionStatus{
			TargetInjections: TargetInjections{"test-1": {}},
		}
	})

	Context("with an empty target", func() {
		It("should return false", func() {
			Expect(disruptionStatus.HasTarget("")).Should(BeFalse())
		})
	})

	Context("with an existing target", func() {
		It("should return true", func() {
			Expect(disruptionStatus.HasTarget("test-1")).Should(BeTrue())
		})
	})

	Context("with an non existing target", func() {
		It("should return false", func() {
			Expect(disruptionStatus.HasTarget("test-2")).Should(BeFalse())
		})
	})
})

var _ = Describe("Disruption", func() {

	var (
		defaultCreationTimestamp = time.Now()
		notBeforeTime            = defaultCreationTimestamp.Add(time.Minute)
	)

	DescribeTable("TimeToInject", func(disruptionBuilder *builderstest.DisruptionBuilder, expectedTime time.Time) {
		// Arrange
		disruption := disruptionBuilder.WithCreationTime(defaultCreationTimestamp).Build()

		// Action && Assert
		Expect(disruption.TimeToInject()).To(Equal(expectedTime))
	},
		Entry(
			"should return creationTimestamp if triggers is nil",
			builderstest.NewDisruptionBuilder(), defaultCreationTimestamp),
		Entry(
			"should return triggers.createPods if triggers.inject is nil",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.NewTime(notBeforeTime),
					Offset:    "",
				},
			}), notBeforeTime),
		Entry(
			"should return inject.notBefore if set",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				Inject: DisruptionTrigger{
					NotBefore: metav1.NewTime(notBeforeTime),
					Offset:    "",
				},
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "2m",
				},
			}), notBeforeTime),
		Entry(
			"should return a time after creationTimestamp if inject.offset is set",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				Inject: DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "1m",
				},
			}), notBeforeTime),
		Entry(
			"should return creationTimestamp if inject.NotBefore is before creationTimestamp",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.NewTime(defaultCreationTimestamp.Add(-time.Minute)),
				},
			}), defaultCreationTimestamp),
		Entry(
			"should return creationTimestamp + 5 minutes if createPods.offset is set",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "5m",
				},
			}), defaultCreationTimestamp.Add(time.Minute*5)),
		Entry(
			"should return creationTimestamp + 5 minutes if createPods.NotBefore is before creationTimestamp",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.NewTime(defaultCreationTimestamp.Add(-time.Minute * 5)),
				},
			}), defaultCreationTimestamp),
	)

	DescribeTable("TimeToCreatePods", func(disruptionBuilder *builderstest.DisruptionBuilder, expectedTime time.Time) {
		// Arrange
		disruption := disruptionBuilder.WithCreationTime(defaultCreationTimestamp).Build()

		// Action && Assert
		Expect(disruption.TimeToCreatePods()).To(Equal(expectedTime))
	},
		Entry(
			"should return creationTimestamp if triggers is nil",
			builderstest.NewDisruptionBuilder(),
			defaultCreationTimestamp),
		Entry(
			"should return creationTimestamp if triggers.createPods is nil",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				Inject: DisruptionTrigger{
					Offset: "15m",
				},
			}),
			defaultCreationTimestamp),
		Entry(
			"should return createPods.notBefore if set",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				Inject: DisruptionTrigger{
					Offset: "15m",
				},
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.NewTime(notBeforeTime),
					Offset:    "",
				},
			}),
			notBeforeTime),
		Entry(
			"should return a time after creationTimestamp if createPods.offset is set",
			builderstest.NewDisruptionBuilder().WithDisruptionTriggers(DisruptionTriggers{
				CreatePods: DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "5m",
				},
			}),
			defaultCreationTimestamp.Add(time.Minute*5)),
	)

	DescribeTable("TerminationStatus", func(disruptionBuilder *builderstest.DisruptionBuilder, pods builderstest.PodsBuilder, expectedTerminationStatus TerminationStatus) {
		// Arrange
		disruption := disruptionBuilder.Build()

		// Action && Assert
		Expect(disruption.TerminationStatus(pods.Build())).To(Equal(expectedTerminationStatus))
	},
		Entry(
			"not yet created disruption IS NOT terminated",
			builderstest.NewDisruptionBuilder().Reset(),
			nil,
			TSNotTerminated),
		Entry(
			"1s before deadline, disruption IS NOT terminated",
			builderstest.NewDisruptionBuilder().WithCreationDuration(time.Minute-time.Second),
			builderstest.NewPodsBuilder(),
			TSNotTerminated),
		Entry(
			"1s after deadline, disruption IS definitively terminated",
			builderstest.NewDisruptionBuilder().WithCreationDuration(time.Minute+time.Second),
			builderstest.NewPodsBuilder(),
			TSDefinitivelyTerminated),
		Entry(
			"half duration disruption IS NOT terminated",
			builderstest.NewDisruptionBuilder(),
			builderstest.NewPodsBuilder(),
			TSNotTerminated),
		Entry(
			"at deadline, disruption IS definitively terminated (however even ns before it is not)",
			builderstest.NewDisruptionBuilder().WithCreationDuration(time.Minute),
			builderstest.NewPodsBuilder(),
			TSDefinitivelyTerminated),
		Entry(
			"deleted disruption IS definitively terminated",
			builderstest.NewDisruptionBuilder().WithCreationDuration(time.Minute).WithDeletion(),
			builderstest.NewPodsBuilder(),
			TSDefinitivelyTerminated),
		Entry(
			"one chaos pod exited out of two IS NOT terminated",
			builderstest.NewDisruptionBuilder(),
			builderstest.NewPodsBuilder().One().Terminated().Parent(),
			TSNotTerminated),
		Entry(
			"all chaos pods exited IS temporarily terminated",
			builderstest.NewDisruptionBuilder(),
			builderstest.NewPodsBuilder().One().Terminated().Parent().Two().Terminated().Parent(),
			TSTemporarilyTerminated),
		Entry(
			"no pod injected is temporarily terminated",
			builderstest.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusInjected),
			nil,
			TSTemporarilyTerminated),
		Entry(
			"no pod partially injected is temporarily terminated",
			builderstest.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusPartiallyInjected),
			nil,
			TSTemporarilyTerminated),
		Entry(
			"no pod NOT injected is not terminated",
			builderstest.NewDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusNotInjected),
			nil,
			TSNotTerminated),
		Entry(
			"no pod initial injection status is not terminated",
			builderstest.NewDisruptionBuilder(),
			nil,
			TSNotTerminated),
	)

	DescribeTable("RemainingDuration", func(disruptionBuilder *builderstest.DisruptionBuilder, expectedRemainingDuration time.Duration) {
		// Arrange
		disruption := disruptionBuilder.Build()

		// Action && Assert
		remainingDuration := disruption.RemainingDuration().Round(time.Second).Truncate(2 * time.Second)
		Expect(remainingDuration).To(Equal(expectedRemainingDuration))
	},
		Entry(
			"should return 30 remaining duration seconds with a disruption created 30 seconds ago with a 1m duration",
			builderstest.NewDisruptionBuilder().WithCreationDuration(30*time.Second).WithDuration("1m"),
			30*time.Second),
		Entry(
			"should return 90 remaining duration seconds with a disruption created 30 seconds ago with a 2m duration",
			builderstest.NewDisruptionBuilder().WithCreationDuration(30*time.Second).WithDuration("2m"),
			90*time.Second),
	)

	Describe("GetTargetsCountAsInt", func() {

		DescribeTable("success cases", func(disruptionBuilder *builderstest.DisruptionBuilder, inputTargetCount int, inputRoundUp bool, expectedTargetCount int) {
			// Arrange
			disruption := disruptionBuilder.Build()

			// Action
			disruptionTargetCount, err := disruption.GetTargetsCountAsInt(inputTargetCount, inputRoundUp)

			// Assert
			Expect(err).ShouldNot(HaveOccurred())
			Expect(disruptionTargetCount).To(Equal(expectedTargetCount))
		},
			Entry(
				"disruption with a count sets at 1 and a single target count with round up at false",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   0,
					IntVal: 1,
					StrVal: "1",
				}),
				1,
				false,
				1,
			),
			Entry(
				"should return 2 targets count with a disruption with a count sets at 2 and a single target count with round up at false",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   0,
					IntVal: 2,
					StrVal: "2",
				}),
				1,
				false,
				2,
			),
			Entry(
				"should return 1 target count with a disruption with a count sets at 100% and a single target count with round up at false",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   1,
					IntVal: 100,
					StrVal: "100%",
				}),
				1,
				false,
				1,
			),
			Entry(
				"should return 50 targets count with a disruption with a count sets at 50% and 100 targets count with round up at false",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   1,
					IntVal: 50,
					StrVal: "50%",
				}),
				100,
				false,
				50,
			),
			Entry(
				"should return 52 targets count with a disruption with a count sets at 51% and 101 targets count with round up at true",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   1,
					IntVal: 51,
					StrVal: "51%",
				}),
				101,
				true,
				52,
			),
			Entry(
				"should return 51 targets count with a disruption with a count sets at 51% and 101 targets count with round up at false",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   1,
					IntVal: 51,
					StrVal: "51%",
				}),
				101,
				false,
				51,
			))

		DescribeTable("error cases", func(disruptionBuilder *builderstest.DisruptionBuilder, inputTargetCount int, inputRoundUp bool, expectedErrorMessage string) {
			disruption := disruptionBuilder.Build()

			// Action
			_, err := disruption.GetTargetsCountAsInt(inputTargetCount, inputRoundUp)

			// Assert
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErrorMessage))
		},
			Entry(
				"should return an error with a disruption without count",
				builderstest.NewDisruptionBuilder(),
				nil,
				false,
				"nil value for IntOrString",
			),
			Entry(
				"should return an error with a disruption with an invalid count",
				builderstest.NewDisruptionBuilder().WithCount(&intstr.IntOrString{
					Type:   2,
					IntVal: 0,
					StrVal: "",
				}),
				nil,
				false,
				"invalid value for IntOrString",
			))
	})

	DescribeTable("IsDeletionExpired", func(disruptionBuilder *builderstest.DisruptionBuilder, timeoutDuration time.Duration, expectedResult bool) {
		// Arrange
		disruption := disruptionBuilder.Build()

		// Action && Assert
		Expect(disruption.IsDeletionExpired(timeoutDuration)).To(Equal(expectedResult))
	},
		Entry("with an none deleted disruption", builderstest.NewDisruptionBuilder(), time.Minute*10, false),
		Entry("with a disruption marked to be deleted not exceeding the timeout limit", builderstest.NewDisruptionBuilder().WithDeletion(), time.Minute*10, false),
		Entry("with a disruption marked to be deleted exceeding the timeout limit", builderstest.NewDisruptionBuilder().WithDeletion(), time.Minute*(-1), true),
	)
})

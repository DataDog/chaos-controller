// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package api_test

import (
	"github.com/DataDog/chaos-controller/types"
	"math/rand"
	"sort"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DisruptionStatus.RemoveDeadTargets Test", func() {
	var matchingTargets []string
	var status *v1beta1.DisruptionStatus

	BeforeEach(func() {
		rand.Seed(time.Now().UnixNano())
		status = &v1beta1.DisruptionStatus{
			TargetInjections: makeValidTargetInjections(),
		}
	})

	When("matchingTargets contains some current targets", func() {
		var saveTargets v1beta1.TargetInjections

		BeforeEach(func() {
			matchingTargets = []string{
				"target-1",
				"target-2",
				"target-3",
				"target-4",
				"target-5",
			}
			saveTargets = status.TargetInjections.DeepCopy()
		})

		When("matchingTargets contains exactly all the current targets", func() {
			It("expects status.TargetInjections to be intact", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.TargetInjections).To(HaveLen(5))
				Expect(status.TargetInjections).To(BeEquivalentTo(saveTargets))
			})
		})

		When("matchingTargets has all current targets and more", func() {
			JustBeforeEach(func() {
				matchingTargets = append(matchingTargets, "target-6")
			})

			It("expects status.TargetInjection to be full", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.TargetInjections).To(HaveLen(5))
			})
		})

		When("matchingTargets contains less targets than status.TargetInjection", func() {
			var randCount int
			BeforeEach(func() {
				rand.Shuffle(len(matchingTargets), func(i, j int) {
					matchingTargets[i], matchingTargets[j] = matchingTargets[j], matchingTargets[i]
				})
				randCount = rand.Intn(4) + 1 // int between 1 and 4
				matchingTargets = matchingTargets[:randCount]
			})

			It("expects status.TargetInjections to be limited to alive targets", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.TargetInjections).To(HaveLen(randCount))
				targetNames := status.TargetInjections.GetTargetNames()
				sort.Strings(targetNames)
				sort.Strings(matchingTargets)
				Expect(targetNames).Should(BeEquivalentTo(matchingTargets))
			})
		})
	})

	When("matchingTarget has no element", func() {
		BeforeEach(func() {
			matchingTargets = []string{}
			Expect(status.TargetInjections).To(HaveLen(5))
		})

		It("expects status.TargetInjection to be empty", func() {
			status.RemoveDeadTargets(matchingTargets)
			Expect(status.TargetInjections).To(HaveLen(0))
		})
	})

	When("matchingTarget has all different elements", func() {
		BeforeEach(func() {
			matchingTargets = []string{
				"target-6",
				"target-7",
				"target-8",
				"target-9",
				"target-10",
			}
		})

		It("expects status.TargetInjection to be empty", func() {
			status.RemoveDeadTargets(matchingTargets)
			Expect(status.TargetInjections).To(HaveLen(0))
		})
	})
})

func makeValidTargetInjections() v1beta1.TargetInjections {
	return v1beta1.TargetInjections{
		"target-1": v1beta1.TargetInjection{
			InjectorPodName: "pod-1",
			InjectionStatus: types.DisruptionInjectionStatusNotInjected,
		},
		"target-2": v1beta1.TargetInjection{
			InjectorPodName: "pod-2",
			InjectionStatus: types.DisruptionInjectionStatusNotInjected,
		},
		"target-3": v1beta1.TargetInjection{
			InjectorPodName: "pod-3",
			InjectionStatus: types.DisruptionInjectionStatusNotInjected,
		},
		"target-4": v1beta1.TargetInjection{
			InjectorPodName: "pod-4",
			InjectionStatus: types.DisruptionInjectionStatusNotInjected,
		},
		"target-5": v1beta1.TargetInjection{
			InjectorPodName: "pod-6",
			InjectionStatus: types.DisruptionInjectionStatusNotInjected,
		},
	}
}

var _ = Describe("DisruptionStatus.AddTargets Test", func() {
	var status, oldStatus *v1beta1.DisruptionStatus
	var newTargetsCount int
	var eligibleTargets v1beta1.TargetInjections

	BeforeEach(func() {
		rand.Seed(time.Now().UnixNano())
		status = &v1beta1.DisruptionStatus{
			TargetInjections: makeValidTargetInjections(),
		}
		oldStatus = status.DeepCopy()
	})

	When("newTargetsCount is between [1;5]", func() {
		BeforeEach(func() {
			newTargetsCount = rand.Intn(5) + 1 // int between [1;5]
		})

		When("eligibleTargets is empty", func() {
			BeforeEach(func() {
				eligibleTargets = v1beta1.TargetInjections{}
			})

			It("expects status.TargetInjection to stay the same", func() {
				status.AddTargets(newTargetsCount, eligibleTargets)
				Expect(status.TargetInjections).To(Equal(oldStatus.TargetInjections))
			})
		})

		When("eligibleTargets 5 new targets", func() {
			BeforeEach(func() {
				eligibleTargets = v1beta1.TargetInjections{
					"target-6": v1beta1.TargetInjection{
						InjectorPodName: "pod-6",
						InjectionStatus: types.DisruptionInjectionStatusNotInjected,
					},
					"target-7": v1beta1.TargetInjection{
						InjectorPodName: "pod-7",
						InjectionStatus: types.DisruptionInjectionStatusNotInjected,
					},
					"target-8": v1beta1.TargetInjection{
						InjectorPodName: "pod-8",
						InjectionStatus: types.DisruptionInjectionStatusNotInjected,
					},
					"target-9": v1beta1.TargetInjection{
						InjectorPodName: "pod-9",
						InjectionStatus: types.DisruptionInjectionStatusNotInjected,
					},
					"target-10": v1beta1.TargetInjection{
						InjectorPodName: "pod-10",
						InjectionStatus: types.DisruptionInjectionStatusNotInjected,
					},
				}
			})

			It("expects newTargets to transfer", func() {
				status.AddTargets(newTargetsCount, eligibleTargets)
				Expect(status.TargetInjections).To(HaveLen(5 + newTargetsCount))
			})
		})
	})

	When("eligibleTargets only has 2 new targets and newTargetsCount is 5+", func() {
		BeforeEach(func() {
			eligibleTargets = v1beta1.TargetInjections{
				"target-6": {},
				"target-7": {},
			} // only 2 new targets
			newTargetsCount = rand.Intn(6) + 5 // int between [5;10]
		})

		It("expects all available targets to transfer", func() {
			oldTargetsCount := len(status.TargetInjections)
			lenEligibleTargets := len(eligibleTargets)
			status.AddTargets(newTargetsCount, eligibleTargets)
			Expect(status.TargetInjections).To(HaveLen(oldTargetsCount + lenEligibleTargets))
		})
	})
})

var _ = Describe("DisruptionStatus.RemoveTargets Test", func() {
	var toRemoveTargetsCount int
	var status *v1beta1.DisruptionStatus

	BeforeEach(func() {
		rand.Seed(time.Now().UnixNano())
		status = &v1beta1.DisruptionStatus{
			TargetInjections: makeValidTargetInjections(),
		}
	})

	Context("asks to remove no targets", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = 0
		})

		It("expects to remove none of the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.TargetInjections).To(HaveLen(5))
		})
	})

	Context("asks to remove less targets than there are in status.TargetInjection", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = rand.Intn(4) + 1
		})

		It("expects to remove part of the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.TargetInjections).To(HaveLen(5 - toRemoveTargetsCount))
		})
	})

	Context("asks to remove exactly the amount of targets in status.TargetInjection", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = len(status.TargetInjections)
		})

		It("expects to remove all the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.TargetInjections).To(BeEmpty())
		})
	})

	Context("asks to remove more than the amount of targets in status.TargetInjection", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = rand.Intn(4) + 1 + len(status.TargetInjections)
		})

		It("expects to remove all the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.TargetInjections).To(BeEmpty())
		})
	})
})

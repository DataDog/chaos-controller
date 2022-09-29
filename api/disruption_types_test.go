// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package api_test

import (
	"math/rand"
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
			Targets: []string{"0.0.0.0", "1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"},
		}
	})

	JustBeforeEach(func() {
		rand.Shuffle(len(matchingTargets), func(i, j int) {
			matchingTargets[i], matchingTargets[j] = matchingTargets[j], matchingTargets[i]
		})
	})

	When("matchingTargets contains some current targets", func() {
		BeforeEach(func() {
			matchingTargets = status.Targets
		})

		When("matchingTargets contains exactly all the current targets", func() {
			It("expects status.Targets to be intact", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.Targets).To(HaveLen(5))
				Expect(status.Targets).To(Equal(matchingTargets))
			})
		})

		When("matchingTargets has all current targets and more", func() {
			JustBeforeEach(func() {
				matchingTargets = append(matchingTargets, "5.5.5.5")
			})

			It("expects status.Targets to be full", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.Targets).To(HaveLen(5))
			})
		})

		When("matchingTargets contains less targets than status.Targets", func() {
			var randCount int
			BeforeEach(func() {
				rand.Shuffle(len(matchingTargets), func(i, j int) {
					matchingTargets[i], matchingTargets[j] = matchingTargets[j], matchingTargets[i]
				})
				randCount = rand.Intn(4) + 1 // int between 1 and 4
				matchingTargets = matchingTargets[:randCount]
			})

			It("expects status.Targets to be limited to alive targets", func() {
				status.RemoveDeadTargets(matchingTargets)
				Expect(status.Targets).To(HaveLen(randCount))
				Expect(status.Targets).To(Equal(matchingTargets))
			})
		})
	})

	When("matchingTarget has no element", func() {
		BeforeEach(func() {
			matchingTargets = []string{}
			Expect(status.Targets).To(HaveLen(5))
		})

		It("expects status.Targets to be empty", func() {
			status.RemoveDeadTargets(matchingTargets)
			Expect(status.Targets).To(HaveLen(0))
		})
	})

	When("matchingTarget has all different elements", func() {
		BeforeEach(func() {
			matchingTargets = []string{"5.5.5.5", "6.6.6.6", "7.7.7.7", "8.8.8.8", "9.9.9.9"}
		})

		It("expects status.Targets to be empty", func() {
			status.RemoveDeadTargets(matchingTargets)
			Expect(status.Targets).To(HaveLen(0))
		})
	})
})

var _ = Describe("DisruptionStatus.AddTargets Test", func() {
	var status, oldStatus *v1beta1.DisruptionStatus
	var newTargetsCount int
	var eligibleTargets []string

	BeforeEach(func() {
		rand.Seed(time.Now().UnixNano())
		status = &v1beta1.DisruptionStatus{
			Targets: []string{"0.0.0.0", "1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"},
		}
		oldStatus = status.DeepCopy()
	})

	When("newTargetsCount is between [1;5]", func() {
		BeforeEach(func() {
			newTargetsCount = rand.Intn(5) + 1 // int between [1;5]
		})

		When("eligibleTargets is empty", func() {
			BeforeEach(func() {
				eligibleTargets = []string{}
			})

			It("expects status.Targets to stay the same", func() {
				status.AddTargets(newTargetsCount, eligibleTargets)
				Expect(status.Targets).To(Equal(oldStatus.Targets))
			})
		})

		When("eligibleTargets 5 new targets", func() {
			BeforeEach(func() {
				eligibleTargets = []string{"5.5.5.5", "6.6.6.6", "7.7.7.7", "8.8.8.8", "9.9.9.9"}
			})

			It("expects newTargets to transfer", func() {
				status.AddTargets(newTargetsCount, eligibleTargets)
				Expect(status.Targets).To(HaveLen(5 + newTargetsCount))
			})
		})

		When("eligibleTargets has exactly all current targets", func() {
			BeforeEach(func() {
				eligibleTargets = status.Targets
			})

			It("expects no filtering, targets will appear double", func() {
				status.AddTargets(newTargetsCount, eligibleTargets)
				Expect(status.Targets).To(HaveLen(5 + newTargetsCount))
			})
		})
	})

	When("eligibleTargets only has 2 new targets and newTargetsCount is 5+", func() {
		BeforeEach(func() {
			eligibleTargets = []string{"5.5.5.5", "6.6.6.6"} // only 2 new targets
			newTargetsCount = rand.Intn(6) + 5               // int between [5;10]
		})

		It("expects all available targets to transfer", func() {
			oldTargetsCount := len(status.Targets)
			status.AddTargets(newTargetsCount, eligibleTargets)
			Expect(status.Targets).To(HaveLen(oldTargetsCount + len(eligibleTargets)))
		})
	})
})

var _ = Describe("DisruptionStatus.RemoveTargets Test", func() {
	var toRemoveTargetsCount int
	var status *v1beta1.DisruptionStatus

	BeforeEach(func() {
		rand.Seed(time.Now().UnixNano())
		status = &v1beta1.DisruptionStatus{
			Targets: []string{"0.0.0.0", "1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"},
		}
	})

	Context("asks to remove no targets", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = 0
		})

		It("expects to remove none of the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.Targets).To(HaveLen(5))
		})
	})

	Context("asks to remove less targets than there are in status.Targets", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = rand.Intn(4) + 1
		})

		It("expects to remove part of the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.Targets).To(HaveLen(5 - toRemoveTargetsCount))
		})
	})

	Context("asks to remove exactly the amount of targets in status.Targets", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = len(status.Targets)
		})

		It("expects to remove all the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.Targets).To(BeEmpty())
		})
	})

	Context("asks to remove more than the amount of targets in status.Targets", func() {
		BeforeEach(func() {
			toRemoveTargetsCount = rand.Intn(4) + 1 + len(status.Targets)
		})

		It("expects to remove all the targets", func() {
			status.RemoveTargets(toRemoveTargetsCount)
			Expect(status.Targets).To(BeEmpty())
		})
	})
})

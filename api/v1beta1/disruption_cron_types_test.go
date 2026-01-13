// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DisruptionCron Types", func() {
	var delay = time.Duration(20 * time.Second)

	Describe("IsReadyToRemoveFinalizer", func() {
		Describe("true cases", func() {
			When("deletedAt is set and we are ready to remove the finalizer", func() {
				It("should return true", func() {
					// Arrange
					readyTime := time.Now().Add(-1 * time.Duration(21*time.Second))

					disruptionCron := DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &metav1.Time{Time: readyTime},
						},
					}

					// Act
					isReady := disruptionCron.IsReadyToRemoveFinalizer(delay)

					// Assert
					Expect(isReady).To(BeTrue())
				})
			})
		})

		Describe("false cases", func() {
			DescribeTable("all false cases", func(disruptionCron DisruptionCron) {
				// Act
				isReady := disruptionCron.IsReadyToRemoveFinalizer(delay)

				// Assert
				Expect(isReady).To(BeFalse())
			},
				Entry("no deletion timestamp set", DisruptionCron{}),
				Entry("deletion timestamp is < than required", DisruptionCron{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				}))
		})
	})

})

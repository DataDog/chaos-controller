// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package network_test

import (
	"fmt"
	"net"

	"github.com/DataDog/chaos-controller/network"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SelectIPsByPercentage", func() {
	var testIPs []*net.IPNet

	BeforeEach(func() {
		// Create a test set of IPs
		testIPs = []*net.IPNet{
			parseCIDR("10.0.0.1/32"),
			parseCIDR("10.0.0.2/32"),
			parseCIDR("10.0.0.3/32"),
			parseCIDR("10.0.0.4/32"),
			parseCIDR("10.0.0.5/32"),
			parseCIDR("10.0.0.6/32"),
			parseCIDR("10.0.0.7/32"),
			parseCIDR("10.0.0.8/32"),
			parseCIDR("10.0.0.9/32"),
			parseCIDR("10.0.0.10/32"),
		}
	})

	Context("Success cases", func() {
		DescribeTable("should select correct percentage of IPs",
			func(percentage int, expectedCount int) {
				// Act
				result := network.SelectIPsByPercentage(testIPs, percentage, "test-seed")

				// Assert
				Expect(result).To(HaveLen(expectedCount))
			},
			Entry("10% of 10 IPs", 10, 1),
			Entry("20% of 10 IPs", 20, 2),
			Entry("25% of 10 IPs", 25, 3),
			Entry("50% of 10 IPs", 50, 5),
			Entry("75% of 10 IPs", 75, 8),
			Entry("90% of 10 IPs", 90, 9),
			Entry("100% of 10 IPs", 100, 10),
		)

		It("should return consistent results with same seed", func() {
			// Act
			seed := "consistent-seed"
			result1 := network.SelectIPsByPercentage(testIPs, 50, seed)
			result2 := network.SelectIPsByPercentage(testIPs, 50, seed)

			// Assert
			Expect(result1).To(HaveLen(len(result2)))
			for i := range result1 {
				Expect(result1[i].String()).To(Equal(result2[i].String()))
			}
		})

		It("should return different results with different seeds", func() {
			// Act
			result1 := network.SelectIPsByPercentage(testIPs, 50, "seed1")
			result2 := network.SelectIPsByPercentage(testIPs, 50, "seed2")

			// Assert
			Expect(result1).To(HaveLen(5))
			Expect(result2).To(HaveLen(5))

			// At least one IP should be different
			result1Map := make(map[string]bool)
			for _, ip := range result1 {
				result1Map[ip.String()] = true
			}

			allSame := true
			for _, ip := range result2 {
				if !result1Map[ip.String()] {
					allSame = false
					break
				}
			}

			Expect(allSame).To(BeFalse(), "Expected different seeds to produce different selections")
		})

		It("should return all IPs when percentage is 100", func() {
			// Act
			result := network.SelectIPsByPercentage(testIPs, 100, "test-seed")

			// Assert
			Expect(result).To(HaveLen(len(testIPs)))
		})

		It("should return all IPs when percentage exceeds number of IPs", func() {
			// Act
			smallSet := []*net.IPNet{
				parseCIDR("10.0.0.1/32"),
			}
			result := network.SelectIPsByPercentage(smallSet, 50, "test-seed")

			// Assert
			Expect(result).To(HaveLen(1))
		})
	})

	Context("Edge cases", func() {
		It("should return empty slice when input is empty", func() {
			// Act
			emptyIPs := []*net.IPNet{}
			result := network.SelectIPsByPercentage(emptyIPs, 50, "test-seed")

			// Assert
			Expect(result).To(BeEmpty())
		})

		DescribeTable("should return all IPs for boundary percentage values",
			func(percentage int) {
				// Act
				result := network.SelectIPsByPercentage(testIPs, percentage, "test-seed")

				// Assert
				Expect(result).To(HaveLen(len(testIPs)))
			},
			Entry("when percentage is 0", 0),
			Entry("when percentage is negative", -10),
			Entry("when percentage is greater than 100", 150),
		)

		It("should ceil the selection count", func() {
			// Act
			result := network.SelectIPsByPercentage(testIPs, 15, "test-seed")

			// Assert
			Expect(result).To(HaveLen(2))
		})
	})

	Context("Deterministic selection", func() {
		It("should select the same IPs across multiple calls with same seed", func() {
			// Arrange
			seed := "stable-seed"

			// Act
			results := make([][]*net.IPNet, 5)
			for i := 0; i < 5; i++ {
				results[i] = network.SelectIPsByPercentage(testIPs, 30, seed)
			}

			// Assert
			// All results should be identical
			for i := 1; i < len(results); i++ {
				Expect(results[i]).To(HaveLen(len(results[0])))
				for j := range results[i] {
					Expect(results[i][j].String()).To(Equal(results[0][j].String()))
				}
			}
		})

		It("should maintain consistency even with IP order changes", func() {
			// Arrange
			seed := "order-test-seed"

			// Act
			// Select from original order
			result1 := network.SelectIPsByPercentage(testIPs, 50, seed)

			// Reverse the IP order
			reversedIPs := make([]*net.IPNet, len(testIPs))
			for i, ip := range testIPs {
				reversedIPs[len(testIPs)-1-i] = ip
			}

			// Select from reversed order
			result2 := network.SelectIPsByPercentage(reversedIPs, 50, seed)

			// Assert
			result1Map := make(map[string]bool)
			for _, ip := range result1 {
				result1Map[ip.String()] = true
			}

			for _, ip := range result2 {
				Expect(result1Map[ip.String()]).To(BeTrue(), fmt.Sprintf("IP %s should be in both selections", ip.String()))
			}
		})
	})
})

func parseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse CIDR %s: %v", cidr, err))
	}
	return ipNet
}

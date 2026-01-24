// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClockSkewSpec", func() {
	When("Call the 'Validate' method", func() {
		DescribeTable("success cases",
			func(clockSkewSpec ClockSkewSpec) {
				// Action && Assert
				Expect(clockSkewSpec.Validate()).Should(Succeed())
			},
			Entry("with a positive offset of 1 hour",
				ClockSkewSpec{
					Offset: DisruptionDuration("1h"),
				},
			),
			Entry("with a positive offset of 24 hours",
				ClockSkewSpec{
					Offset: DisruptionDuration("24h"),
				},
			),
			Entry("with a positive offset of 1 year (8760h)",
				ClockSkewSpec{
					Offset: DisruptionDuration("8760h"),
				},
			),
			Entry("with a negative offset of 1 hour",
				ClockSkewSpec{
					Offset: DisruptionDuration("-1h"),
				},
			),
			Entry("with a negative offset of 30 minutes",
				ClockSkewSpec{
					Offset: DisruptionDuration("-30m"),
				},
			),
			Entry("with a positive offset of 5 years",
				ClockSkewSpec{
					Offset: DisruptionDuration("43800h"), // 5 years
				},
			),
			Entry("with a complex duration",
				ClockSkewSpec{
					Offset: DisruptionDuration("2h30m45s"),
				},
			),
			Entry("with milliseconds",
				ClockSkewSpec{
					Offset: DisruptionDuration("500ms"),
				},
			),
			Entry("with seconds",
				ClockSkewSpec{
					Offset: DisruptionDuration("30s"),
				},
			),
		)

		DescribeTable("error cases",
			func(cs ClockSkewSpec, expectedErrors []string) {
				// Action
				err := cs.Validate()

				// Assert
				Expect(err).To(HaveOccurred())
				for _, expectedError := range expectedErrors {
					Expect(err.Error()).To(ContainSubstring(expectedError))
				}
			},
			Entry("with an empty offset",
				ClockSkewSpec{
					Offset: DisruptionDuration(""),
				},
				[]string{
					"clockSkew.offset must be specified",
				},
			),
			Entry("with a zero offset",
				ClockSkewSpec{
					Offset: DisruptionDuration("0s"),
				},
				[]string{
					"clockSkew.offset must not be zero",
				},
			),
			Entry("with a zero offset as 0h",
				ClockSkewSpec{
					Offset: DisruptionDuration("0h"),
				},
				[]string{
					"clockSkew.offset must not be zero",
				},
			),
			Entry("with an offset greater than 10 years (positive)",
				ClockSkewSpec{
					Offset: DisruptionDuration("100000h"), // ~11.4 years
				},
				[]string{
					"seems unusually large (>10 years)",
				},
			),
			Entry("with an offset greater than 10 years (negative)",
				ClockSkewSpec{
					Offset: DisruptionDuration("-100000h"), // ~11.4 years
				},
				[]string{
					"seems unusually large (>10 years)",
				},
			),
		)
	})

	When("Call the 'GenerateArgs' method", func() {
		DescribeTable("success cases",
			func(clockSkewSpec ClockSkewSpec, expectedArgs []string) {
				// Arrange
				expectedArgs = append([]string{"clock-skew"}, expectedArgs...)

				// Action
				args := clockSkewSpec.GenerateArgs()

				// Assert
				Expect(args).Should(Equal(expectedArgs))
			},
			Entry("with a positive offset of 1 hour",
				ClockSkewSpec{
					Offset: DisruptionDuration("1h"),
				},
				[]string{"--offset", "1h"},
			),
			Entry("with a positive offset of 24 hours",
				ClockSkewSpec{
					Offset: DisruptionDuration("24h"),
				},
				[]string{"--offset", "24h"},
			),
			Entry("with a positive offset of 1 year (8760h)",
				ClockSkewSpec{
					Offset: DisruptionDuration("8760h"),
				},
				[]string{"--offset", "8760h"},
			),
			Entry("with a negative offset of 1 hour",
				ClockSkewSpec{
					Offset: DisruptionDuration("-1h"),
				},
				[]string{"--offset", "-1h"},
			),
			Entry("with a negative offset of 30 minutes",
				ClockSkewSpec{
					Offset: DisruptionDuration("-30m"),
				},
				[]string{"--offset", "-30m"},
			),
			Entry("with a complex duration",
				ClockSkewSpec{
					Offset: DisruptionDuration("2h30m45s"),
				},
				[]string{"--offset", "2h30m45s"},
			),
		)
	})

	When("Call the 'Explain' method", func() {
		Context("with positive offsets", func() {
			It("should explain advancing time forward", func() {
				// Arrange
				clockSkewSpec := ClockSkewSpec{
					Offset: DisruptionDuration("24h"),
				}

				// Action
				explanation := clockSkewSpec.Explain()

				// Assert
				Expect(explanation).ToNot(BeEmpty())
				explanationText := fmt.Sprintf("%v", explanation)
				Expect(explanationText).To(ContainSubstring("advance time forward"))
				Expect(explanationText).To(ContainSubstring("24h"))
				Expect(explanationText).To(ContainSubstring("certificate expiration"))
			})

			It("should mention days when offset is greater than 24 hours", func() {
				// Arrange
				clockSkewSpec := ClockSkewSpec{
					Offset: DisruptionDuration("72h"), // 3 days
				}

				// Action
				explanation := clockSkewSpec.Explain()

				// Assert
				Expect(explanation).ToNot(BeEmpty())
				explanationText := fmt.Sprintf("%v", explanation)
				Expect(explanationText).To(ContainSubstring("3 days"))
				Expect(explanationText).To(ContainSubstring("certificate expiration"))
			})

			It("should include libfaketime information", func() {
				// Arrange
				clockSkewSpec := ClockSkewSpec{
					Offset: DisruptionDuration("1h"),
				}

				// Action
				explanation := clockSkewSpec.Explain()

				// Assert
				Expect(explanation).ToNot(BeEmpty())
				explanationText := fmt.Sprintf("%v", explanation)
				Expect(explanationText).To(ContainSubstring("libfaketime"))
				Expect(explanationText).To(ContainSubstring("LD_PRELOAD"))
			})
		})

		Context("with negative offsets", func() {
			It("should explain moving time backward", func() {
				// Arrange
				clockSkewSpec := ClockSkewSpec{
					Offset: DisruptionDuration("-1h"),
				}

				// Action
				explanation := clockSkewSpec.Explain()

				// Assert
				Expect(explanation).ToNot(BeEmpty())
				explanationText := fmt.Sprintf("%v", explanation)
				Expect(explanationText).To(ContainSubstring("move time backward"))
				Expect(explanationText).To(ContainSubstring("clock drift"))
			})
		})

		Context("with very small offsets", func() {
			It("should not mention days for offsets less than 24 hours", func() {
				// Arrange
				clockSkewSpec := ClockSkewSpec{
					Offset: DisruptionDuration("5h"),
				}

				// Action
				explanation := clockSkewSpec.Explain()

				// Assert
				Expect(explanation).ToNot(BeEmpty())
				explanationText := fmt.Sprintf("%v", explanation)
				Expect(explanationText).ToNot(ContainSubstring("days"))
			})
		})
	})

	Describe("DisruptionDuration integration", func() {
		It("should correctly parse various time formats", func() {
			testCases := []struct {
				input    DisruptionDuration
				expected time.Duration
			}{
				{"1h", time.Hour},
				{"30m", 30 * time.Minute},
				{"1h30m", time.Hour + 30*time.Minute},
				{"24h", 24 * time.Hour},
				{"8760h", 8760 * time.Hour}, // 1 year
				{"-1h", -time.Hour},
				{"-30m", -30 * time.Minute},
				{"500ms", 500 * time.Millisecond},
			}

			for _, tc := range testCases {
				duration := tc.input.Duration()
				Expect(duration).To(Equal(tc.expected), "Failed for input: %s", tc.input)
			}
		})

		It("should handle the 1 year offset correctly", func() {
			// Arrange
			oneYear := DisruptionDuration("8760h")

			// Action
			duration := oneYear.Duration()

			// Assert
			Expect(duration).To(Equal(8760 * time.Hour))
			// Verify it's approximately 365 days
			Expect(duration.Hours()).To(BeNumerically("~", 365*24, 1))
		})
	})
})

// ============================================================================
// Fuzz Tests
// ============================================================================

// FuzzClockSkewValidate fuzzes the ClockSkewSpec Validate method
// to find edge cases that might cause panics or unexpected behavior
func FuzzClockSkewValidate(f *testing.F) {
	// Seed corpus with known valid and invalid inputs
	seeds := []string{
		"1h",
		"24h",
		"8760h",
		"-1h",
		"-30m",
		"0s",
		"0h",
		"",
		"100000h",
		"-100000h",
		"2h30m45s",
		"500ms",
		"1ns",
		"-1ns",
		// Edge cases
		"999999999h",
		"-999999999h",
		"1s",
		"-1s",
		// Invalid formats (should be handled gracefully)
		"1",
		"h",
		"1hh",
		"abc",
		"1d", // days not supported in Go duration
		"1w", // weeks not supported
		"1y", // years not supported
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, offset string) {
		spec := ClockSkewSpec{
			Offset: DisruptionDuration(offset),
		}

		// The Validate method should never panic, regardless of input
		// We don't assert on the result, just that it doesn't crash
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Validate() panicked with offset %q: %v", offset, r)
			}
		}()

		err := spec.Validate()

		// If there's no error, the offset should be parseable and non-zero
		if err == nil {
			duration := spec.Offset.Duration()
			if duration == 0 {
				t.Errorf("Validate() succeeded but duration is zero for offset %q", offset)
			}
		}

		// If the offset is empty, there should always be an error
		if offset == "" && err == nil {
			t.Errorf("Validate() should fail for empty offset")
		}

		// If the offset is "0s" or "0h", there should be an error about zero duration
		if (offset == "0s" || offset == "0h" || offset == "0m") && err == nil {
			t.Errorf("Validate() should fail for zero duration offset %q", offset)
		}
	})
}

// FuzzClockSkewGenerateArgs fuzzes the ClockSkewSpec GenerateArgs method
func FuzzClockSkewGenerateArgs(f *testing.F) {
	// Seed corpus
	seeds := []string{
		"1h",
		"24h",
		"-1h",
		"8760h",
		"2h30m",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, offset string) {
		spec := ClockSkewSpec{
			Offset: DisruptionDuration(offset),
		}

		// GenerateArgs should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GenerateArgs() panicked with offset %q: %v", offset, r)
			}
		}()

		args := spec.GenerateArgs()

		// Args should always have the base command and at least 2 additional args
		if len(args) < 3 {
			t.Errorf("GenerateArgs() returned too few args: %v for offset %q", args, offset)
		}

		// First arg should always be "clock-skew"
		if args[0] != "clock-skew" {
			t.Errorf("First arg should be 'clock-skew', got %q for offset %q", args[0], offset)
		}

		// Second arg should be "--offset"
		if args[1] != "--offset" {
			t.Errorf("Second arg should be '--offset', got %q for offset %q", args[1], offset)
		}

		// Third arg should be the offset value
		if args[2] != offset {
			t.Errorf("Third arg should be %q, got %q", offset, args[2])
		}
	})
}

// FuzzClockSkewExplain fuzzes the ClockSkewSpec Explain method
func FuzzClockSkewExplain(f *testing.F) {
	// Seed corpus
	seeds := []string{
		"1h",
		"24h",
		"-1h",
		"8760h",
		"-30m",
		"72h",
		"1s",
		"-1s",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, offset string) {
		spec := ClockSkewSpec{
			Offset: DisruptionDuration(offset),
		}

		// Explain should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Explain() panicked with offset %q: %v", offset, r)
			}
		}()

		explanation := spec.Explain()

		// Should always return at least one line
		if len(explanation) == 0 {
			t.Errorf("Explain() returned empty explanation for offset %q", offset)
		}

		// Join explanation for easier checking
		fullText := strings.Join(explanation, " ")

		// Should contain key information
		if !strings.Contains(fullText, "clock") && !strings.Contains(fullText, "time") {
			t.Errorf("Explain() should mention 'clock' or 'time', got: %s for offset %q", fullText, offset)
		}

		// If offset parses correctly, check direction
		duration := spec.Offset.Duration()
		if duration > 0 {
			if !strings.Contains(fullText, "forward") && !strings.Contains(fullText, "advance") {
				t.Logf("Explain() should mention 'forward' or 'advance' for positive offset, got: %s for offset %q", fullText, offset)
			}
		} else if duration < 0 {
			if !strings.Contains(fullText, "backward") {
				t.Logf("Explain() should mention 'backward' for negative offset, got: %s for offset %q", fullText, offset)
			}
		}
	})
}

// FuzzClockSkewDurationParsing fuzzes DisruptionDuration parsing
func FuzzClockSkewDurationParsing(f *testing.F) {
	// Seed corpus with various duration formats
	seeds := []string{
		"1h",
		"1m",
		"1s",
		"1ms",
		"1µs",
		"1ns",
		"-1h",
		"1h30m",
		"1h30m45s",
		"300ms",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, durationStr string) {
		dd := DisruptionDuration(durationStr)

		// Duration() method should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Duration() panicked with input %q: %v", durationStr, r)
			}
		}()

		duration := dd.Duration()

		// If time.ParseDuration succeeds, we should get the same result
		expectedDuration, err := time.ParseDuration(durationStr)
		if err == nil {
			if duration != expectedDuration {
				t.Errorf("Duration() returned %v, expected %v for input %q", duration, expectedDuration, durationStr)
			}
		} else {
			// If parsing fails, duration should be 0
			if duration != 0 {
				t.Logf("Duration() returned %v for unparseable input %q, expected 0", duration, durationStr)
			}
		}
	})
}

// FuzzClockSkewOffsetBounds fuzzes the offset boundary conditions
func FuzzClockSkewOffsetBounds(f *testing.F) {
	// Test around the 10 year boundary
	tenYearsInHours := int64(10 * 365 * 24)

	seeds := []int64{
		1,
		24,
		8760,  // 1 year
		17520, // 2 years
		43800, // 5 years
		tenYearsInHours - 1,
		tenYearsInHours,
		tenYearsInHours + 1,
		100000,
		-1,
		-24,
		-8760,
		-tenYearsInHours - 1,
		-tenYearsInHours,
		-tenYearsInHours + 1,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, hours int64) {
		offset := fmt.Sprintf("%dh", hours)
		spec := ClockSkewSpec{
			Offset: DisruptionDuration(offset),
		}

		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Validate() panicked with %d hours: %v", hours, r)
			}
		}()

		err := spec.Validate()

		// Zero should always fail
		if hours == 0 && err == nil {
			t.Errorf("Validate() should fail for 0 hours")
		}

		// Very large values (> 10 years) should warn but might still pass
		absHours := hours
		if absHours < 0 {
			absHours = -absHours
		}

		maxHours := int64(10 * 365 * 24)
		if absHours > maxHours {
			if err == nil {
				t.Logf("Validate() passed for large offset %d hours (>10 years), this is acceptable if it's a warning", hours)
			}
		}
	})
}

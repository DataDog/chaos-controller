// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getCommonLicenses returns the most frequently used SPDX license identifiers
// Returns a new slice to prevent external modification
func getCommonLicenses() []string {
	return []string{
		"Apache-2.0",
		"MIT",
		"BSD-3-Clause",
		"BSD-2-Clause",
		"GPL-3.0",
		"LGPL-3.0",
		"MPL-2.0",
		"ISC",
	}
}

// PromptForLicenseWithDefault displays an interactive prompt for license selection with a default value
// If defaultLicense is provided, user can press Enter to keep it
// Returns: selected license string, or error if user aborts
func PromptForLicenseWithDefault(moduleName, defaultLicense string) (string, error) {
	commonLicenses := getCommonLicenses()

	fmt.Printf("\n⚠️  Could not determine license for module: %s\n", moduleName)

	if defaultLicense != "" {
		fmt.Printf("\nCached license found: %s\n", defaultLicense)
		fmt.Println("  <Enter>) Keep cached license")
	}

	fmt.Println("\nCommon licenses:")

	for i, license := range commonLicenses {
		fmt.Printf("  %d) %s\n", i+1, license)
	}

	fmt.Println("  m) Enter license manually")
	fmt.Println("  s) Skip this module (will fail build)")

	if defaultLicense != "" {
		fmt.Print("\nSelect option (or press Enter to keep cached): ")
	} else {
		fmt.Print("\nSelect option: ")
	}

	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Handle empty input - keep default if provided
	if input == "" && defaultLicense != "" {
		fmt.Printf("✓ Keeping cached license: %s\n", defaultLicense)
		return defaultLicense, nil
	}

	// Handle manual entry
	if strings.ToLower(input) == "m" {
		fmt.Print("Enter license name (SPDX identifier recommended): ")

		manual, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}

		return strings.TrimSpace(manual), nil
	}

	// Handle skip/abort
	if strings.ToLower(input) == "s" {
		return "", fmt.Errorf("user skipped license selection")
	}

	// Handle numbered selection
	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(commonLicenses) {
		return "", fmt.Errorf("invalid selection: %s", input)
	}

	return commonLicenses[choice-1], nil
}

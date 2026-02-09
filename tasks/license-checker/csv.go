// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

// LicenseEntry represents a row in the LICENSE-3rdparty.csv file
type LicenseEntry struct {
	From    string // Parent module
	Package string // Package name
	License string // License type (SPDX ID)
}

// ReadCSV reads the existing LICENSE-3rdparty.csv file
func ReadCSV(path string) ([]LicenseEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LicenseEntry{}, nil // File doesn't exist yet, return empty
		}

		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return []LicenseEntry{}, nil
	}

	// Skip header row
	var entries []LicenseEntry

	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) >= 3 {
			entries = append(entries, LicenseEntry{
				From:    record[0],
				Package: record[1],
				License: record[2],
			})
		}
	}

	return entries, nil
}

// WriteCSV writes the LICENSE-3rdparty.csv file
func WriteCSV(path string, entries []LicenseEntry) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"From", "Package", "License"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write entries
	for _, entry := range entries {
		if err := writer.Write([]string{entry.From, entry.Package, entry.License}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

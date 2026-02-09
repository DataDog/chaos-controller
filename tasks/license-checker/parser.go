// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Module represents a Go module with its packages
type Module struct {
	Parent   string   // e.g., "github.com/owner/repo"
	Version  string   // e.g., "v1.0.0"
	Packages []string // Child packages under this module
}

// ParseVendorModules parses the vendor/modules.txt file
// Format:
// # github.com/owner/repo v1.0.0
// ## explicit; go 1.21
// github.com/owner/repo/subpackage
// github.com/owner/repo/other
func ParseVendorModules(path string) ([]Module, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}

	defer func() {
		_ = file.Close()
	}()

	var modules []Module

	var currentModule *Module

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip ## lines (metadata)
		if strings.HasPrefix(line, "##") {
			continue
		}

		// New module declaration: # module@version
		if strings.HasPrefix(line, "#") {
			// Save previous module if exists
			if currentModule != nil {
				modules = append(modules, *currentModule)
			}

			// Parse new module
			parts := strings.Fields(line[1:]) // Remove # and split
			if len(parts) >= 1 {
				currentModule = &Module{
					Parent:   strings.TrimSpace(parts[0]),
					Version:  "",
					Packages: []string{},
				}
				if len(parts) >= 2 {
					currentModule.Version = strings.TrimSpace(parts[1])
				}
			}
		} else if currentModule != nil {
			// Package line - add to current module
			pkg := strings.TrimSpace(line)
			if pkg != "" {
				currentModule.Packages = append(currentModule.Packages, pkg)
			}
		}
	}

	// Don't forget the last module
	if currentModule != nil {
		modules = append(modules, *currentModule)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return modules, nil
}

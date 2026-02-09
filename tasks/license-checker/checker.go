// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	classifier "github.com/google/licenseclassifier/v2"
	"github.com/google/licenseclassifier/v2/assets"
)

// LicenseChecker handles license detection and CSV generation
type LicenseChecker struct {
	csvPath     string
	modulesPath string
	classifier  *classifier.Classifier
	cache       map[string]string // module -> license cache from existing CSV
	noPrompt    bool              // disable interactive prompts for CI
}

// LicenseResolutionStrategy encapsulates decision logic for handling license detection
type LicenseResolutionStrategy interface {
	Resolve(lc *LicenseChecker, module string, detectionError error, detected string, cached string, hasCached bool) (string, error)
}

// CILicenseResolver handles CI mode (fail-fast, use cache or fail)
type CILicenseResolver struct{}

func (r *CILicenseResolver) Resolve(lc *LicenseChecker, module string, detectionError error, detected string, cached string, hasCached bool) (string, error) {
	if detectionError != nil {
		// CI mode: use cache if available, otherwise fail
		if hasCached {
			fmt.Printf("warning: license type for package %s cannot be determined but has already been specified: %s\n",
				module, cached)

			return cached, nil
		}

		// CI mode: fail immediately without prompting
		fmt.Fprintf(os.Stderr, "error: %v\n", detectionError)

		return "", fmt.Errorf("license detection failed for %s (use interactive mode or add to cache)", module)
	}

	// Detection succeeded - check if license changed
	if hasCached {
		lc.checkLicenseChange(module, cached, detected)
	}

	return detected, nil
}

// InteractiveLicenseResolver handles interactive mode (prompt user)
type InteractiveLicenseResolver struct{}

func (r *InteractiveLicenseResolver) Resolve(lc *LicenseChecker, module string, detectionError error, detected string, cached string, hasCached bool) (string, error) {
	if detectionError != nil {
		// Interactive mode: prompt user (with default if cached)
		selectedLicense, err := PromptForLicenseWithDefault(module, cached)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return "", fmt.Errorf("license detection failed for %s", module)
		}

		fmt.Printf("✓ User selected license: %s\n", selectedLicense)

		return selectedLicense, nil
	}

	// Detection succeeded - check if license changed
	if hasCached {
		lc.checkLicenseChange(module, cached, detected)
	}

	return detected, nil
}

// NewLicenseChecker creates a new LicenseChecker
func NewLicenseChecker(csvPath, modulesPath string, noPrompt bool) *LicenseChecker {
	// Use DefaultClassifier which loads embedded license data
	lc, err := assets.DefaultClassifier()
	if err != nil {
		// Fallback to empty classifier if assets fail to load
		fmt.Fprintf(os.Stderr, "Warning: failed to load license assets: %v\n", err)

		lc = classifier.NewClassifier(0.5)
	}

	return &LicenseChecker{
		csvPath:     csvPath,
		modulesPath: modulesPath,
		classifier:  lc,
		cache:       make(map[string]string),
		noPrompt:    noPrompt,
	}
}

// loadCache reads the existing CSV and builds a cache of known licenses
func (lc *LicenseChecker) loadCache() error {
	entries, err := ReadCSV(lc.csvPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Cache the license for each parent module
		lc.cache[entry.From] = entry.License
	}

	return nil
}

// findLicenseFile finds the LICENSE file for a given module path
// Returns the path to the license file, or an error if not found or multiple found
func (lc *LicenseChecker) findLicenseFile(modulePath string) (string, error) {
	basePath := filepath.Join("vendor", modulePath)

	// If the directory doesn't exist, try walking up to parent directories
	// (some packages share a LICENSE file with their parent module)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		// Try parent directories and their subdirectories (siblings)
		parentPath := filepath.Dir(modulePath)
		for parentPath != "." && parentPath != "/" {
			parentBasePath := filepath.Join("vendor", parentPath)
			if _, err := os.Stat(parentBasePath); err == nil {
				// First try the parent directory itself
				license, err := lc.findLicenseInDir(parentBasePath, parentPath)
				if err == nil {
					return license, nil
				}

				// Then try sibling directories (other packages in the same module)
				// This handles cases like github.com/moby/sys/mountinfo having a LICENSE
				// that should be used for github.com/moby/sys/atomicwriter
				siblings, err := os.ReadDir(parentBasePath)
				if err == nil {
					for _, sibling := range siblings {
						if !sibling.IsDir() {
							continue
						}

						siblingPath := filepath.Join(parentBasePath, sibling.Name())

						license, err := lc.findLicenseInDir(siblingPath, filepath.Join(parentPath, sibling.Name()))
						if err == nil {
							return license, nil
						}
					}
				}
			}

			parentPath = filepath.Dir(parentPath)
		}

		return "", fmt.Errorf("could not find directory or license for package %s", modulePath)
	}

	return lc.findLicenseInDir(basePath, modulePath)
}

// findLicenseInDir searches for a license file in the specified directory
func (lc *LicenseChecker) findLicenseInDir(basePath, modulePath string) (string, error) {
	// Read directory and look for license files
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return "", fmt.Errorf("could not read directory for package %s: %w", modulePath, err)
	}

	var licenseFiles []string

	var primaryLicense string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		lowerName := strings.ToLower(name)

		// Skip LICENSE-3rdparty.csv files (they're for dependencies, not the package itself)
		if strings.Contains(lowerName, "3rdparty") || strings.Contains(lowerName, "third-party") {
			continue
		}

		// Check if filename starts with "license" or "copying"
		if strings.HasPrefix(lowerName, "license") || strings.HasPrefix(lowerName, "copying") {
			fullPath := filepath.Join(basePath, name)
			licenseFiles = append(licenseFiles, fullPath)

			// Prefer exact "LICENSE" or "COPYING" file (no extensions, no suffixes)
			// Examples: LICENSE (yes), LICENSE.md (no), LICENSE.docs (no), LICENSE-APACHE (no)
			if lowerName == "license" || lowerName == "copying" {
				primaryLicense = fullPath
			}
		}
	}

	if len(licenseFiles) == 0 {
		return "", fmt.Errorf("could not find license file for package %s", modulePath)
	}

	// If we found a primary LICENSE file, use it (even if there are others like LICENSE-APACHE)
	if primaryLicense != "" {
		return primaryLicense, nil
	}

	// Otherwise, if there's only one license file, use it
	if len(licenseFiles) == 1 {
		return licenseFiles[0], nil
	}

	// Multiple license files and no clear primary - this is ambiguous
	return "", fmt.Errorf("multiple license files for package %s: %v", modulePath, licenseFiles)
}

// detectLicenseType uses the license classifier to identify the license type
func (lc *LicenseChecker) detectLicenseType(licenseFile string) (string, error) {
	content, err := os.ReadFile(licenseFile)
	if err != nil {
		return "", fmt.Errorf("failed to read license file: %w", err)
	}

	if lc.classifier == nil {
		return "", fmt.Errorf("license classifier not initialized")
	}

	results := lc.classifier.Match(content)
	if len(results.Matches) == 0 {
		return "", fmt.Errorf("could not determine license type")
	}

	// Filter out generic "Copyright" matches - these are headers, not actual licenses
	// Prefer actual license types like "MIT", "Apache-2.0", etc.
	for _, match := range results.Matches {
		if match.Name != "Copyright" {
			return match.Name, nil
		}
	}

	// If only "Copyright" was found, that's not sufficient - treat as unknown
	if results.Matches[0].Name == "Copyright" {
		return "", fmt.Errorf("only found copyright notice, no actual license detected")
	}

	// Fallback to first match
	return results.Matches[0].Name, nil
}

// getCachedLicense retrieves a cached license for a module
func (lc *LicenseChecker) getCachedLicense(module string) (string, bool) {
	license, ok := lc.cache[module]
	return license, ok
}

// checkLicenseChange compares old and new licenses and warns if different
func (lc *LicenseChecker) checkLicenseChange(module, oldLicense, newLicense string) {
	if oldLicense != newLicense {
		fmt.Printf("warning: license may be outdated for package %s (actual: %s, detected: %s)\n",
			module, oldLicense, newLicense)
	}
}

// DetectModuleLicense encapsulates detection logic (find file + detect type)
func (lc *LicenseChecker) DetectModuleLicense(module Module) (string, error) {
	// Find LICENSE file
	licenseFile, err := lc.findLicenseFile(module.Parent)
	if err != nil {
		return "", err
	}

	// Detect license type
	detected, err := lc.detectLicenseType(licenseFile)
	if err != nil {
		return "", err
	}

	return detected, nil
}

// ResolveLicenseWithStrategy uses strategy to resolve license (detection + caching + validation)
func (lc *LicenseChecker) ResolveLicenseWithStrategy(module Module, resolver LicenseResolutionStrategy) (string, error) {
	// Try to detect license
	detected, detectionError := lc.DetectModuleLicense(module)

	// Get cached value
	cached, hasCached := lc.getCachedLicense(module.Parent)

	// Use strategy to resolve
	return resolver.Resolve(lc, module.Parent, detectionError, detected, cached, hasCached)
}

// CreateLicenseEntries generates entries for all packages in module
func (lc *LicenseChecker) CreateLicenseEntries(module Module, licenseType string) []LicenseEntry {
	var entries []LicenseEntry

	if len(module.Packages) == 0 {
		// Module has no packages - create entry for the module itself
		entries = append(entries, LicenseEntry{
			From:    module.Parent,
			Package: module.Parent,
			License: licenseType,
		})
	} else {
		// Module has packages - create entry for each package
		for _, pkg := range module.Packages {
			entries = append(entries, LicenseEntry{
				From:    module.Parent,
				Package: pkg,
				License: licenseType,
			})
		}
	}

	return entries
}

// Run executes the license checker
func (lc *LicenseChecker) Run() error {
	// 1. Load existing cache
	if err := lc.loadCache(); err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	// 2. Parse vendor/modules.txt
	modules, err := ParseVendorModules(lc.modulesPath)
	if err != nil {
		return fmt.Errorf("failed to parse modules: %w", err)
	}

	// 3. Choose resolution strategy (CI vs Interactive)
	var resolver LicenseResolutionStrategy
	if lc.noPrompt {
		resolver = &CILicenseResolver{}
	} else {
		resolver = &InteractiveLicenseResolver{}
	}

	// 4. Process each module
	var allEntries []LicenseEntry

	for _, module := range modules {
		licenseType, err := lc.ResolveLicenseWithStrategy(module, resolver)
		if err != nil {
			return err
		}

		entries := lc.CreateLicenseEntries(module, licenseType)
		allEntries = append(allEntries, entries...)
	}

	// 5. Write CSV
	if err := WriteCSV(lc.csvPath, allEntries); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	return nil
}

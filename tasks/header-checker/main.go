// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"fmt"
	"os"
)

func main() {
	checker := NewHeaderChecker()

	modified, err := checker.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1) // Exit 1 on actual errors
	}

	if modified {
		// Files were modified, but this is not an error - just informational
		// Exit 0 because the operation succeeded
		fmt.Println("\n✓ Headers updated successfully")
	} else {
		fmt.Println("\n✓ Headers are up to date")
	}

	os.Exit(0) // Success (whether files were modified or already valid)
}

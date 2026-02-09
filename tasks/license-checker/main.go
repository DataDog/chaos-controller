// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"
)

func main() {
	noPrompt := flag.Bool("no-prompt", false, "Disable interactive prompts (fail instead)")

	flag.Parse()

	// Auto-detect if stdin is not a terminal (non-interactive environment)
	// This ensures the tool doesn't hang in CI or when piped
	interactive := !*noPrompt && term.IsTerminal(int(os.Stdin.Fd()))

	checker := NewLicenseChecker("LICENSE-3rdparty.csv", "vendor/modules.txt", !interactive)
	if err := checker.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

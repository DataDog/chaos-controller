// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package api

import (
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DisruptionArgsGenerator generates args for the given disruption
type DisruptionArgsGenerator interface {
	GenerateArgs(level chaostypes.DisruptionLevel, containerIDs []string, sink string, dryRun bool) []string
}

// helper: generate common args
func AppendCommonArgs(args []string, level chaostypes.DisruptionLevel, containerIDs []string, sink string, dryRun bool) []string {
	args = append(args,
		"--metrics-sink",
		sink,
		"--level",
		string(level),
		"--containers-id",
		strings.Join(containerIDs, ","),
	)

	// enable dry-run mode
	if dryRun {
		args = append(args,
			"--dry-run",
		)
	}

	return args
}

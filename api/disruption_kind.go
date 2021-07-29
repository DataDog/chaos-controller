// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package api

import (
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DisruptionKind contains all methods required for a disruption sub-specification (Network, DNS, CPUPressure, etc.)
type DisruptionKind interface {
	// generates CLI args for the given disruption sub-specification
	GenerateArgs() []string

	// validates schema for the given disruption sub-specification
	Validate() error
}

// AppendArgs is a helper function generating common and global args and appending them to the given args array
func AppendArgs(args []string, level chaostypes.DisruptionLevel, kind chaostypes.DisruptionKindName, containerIDs []string, podIP string, sink string, dryRun bool,
	disruptionName string, disruptionNamespace string, targetName string, onInit bool, allowedHosts []string) []string {
	args = append(args,
		// basic args
		"--metrics-sink", sink,
		"--level", string(level),
		"--containers-id", strings.Join(containerIDs, ","),
		"--pod-ip", podIP,

		// log context args
		"--log-context-disruption-name", disruptionName,
		"--log-context-disruption-namespace", disruptionNamespace,
		"--log-context-target-name", targetName,
	)

	// enable dry-run mode
	if dryRun {
		args = append(args, "--dry-run")
	}

	// enable chaos handler init container notification
	if onInit {
		args = append(args, "--on-init")
	}

	// append allowed hosts for network disruptions
	if kind == chaostypes.DisruptionKindNetworkDisruption {
		for _, host := range allowedHosts {
			args = append(args, "--allowed-hosts", host)
		}
	}

	return args
}

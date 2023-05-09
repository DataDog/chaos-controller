// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package api

import (
	"fmt"
	"strings"
	"time"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DisruptionKind contains all methods required for a disruption sub-specification (Network, DNS, CPUPressure, etc.)
type DisruptionKind interface {
	// generates CLI args for the given disruption sub-specification
	GenerateArgs() []string

	// validates schema for the given disruption sub-specification
	Validate() error
}

type DisruptionArgs struct {
	AllowedHosts         []string
	TargetContainers     map[string]string
	Level                chaostypes.DisruptionLevel
	Kind                 chaostypes.DisruptionKindName
	TargetPodIP          string
	MetricsSink          string
	DisruptionName       string
	DisruptionNamespace  string
	TargetName           string
	TargetNodeName       string
	DNSServer            string
	KubeDNS              string
	ChaosNamespace       string
	DryRun               bool
	OnInit               bool
	PulseInitialDelay    time.Duration
	PulseActiveDuration  time.Duration
	PulseDormantDuration time.Duration
}

// AppendArgs is a helper function generating common and global args and appending them to the given args array
func AppendArgs(args []string, xargs DisruptionArgs) []string {
	formattedTargetContainers := []string{}

	for name, id := range xargs.TargetContainers {
		f := fmt.Sprintf("%s;%s", name, id)
		formattedTargetContainers = append(formattedTargetContainers, f)
	}

	args = append(args,
		// basic args
		"--metrics-sink", xargs.MetricsSink,
		"--level", string(xargs.Level),
		"--target-containers", strings.Join(formattedTargetContainers, ","),
		"--target-pod-ip", xargs.TargetPodIP,
		"--chaos-namespace", xargs.ChaosNamespace,

		// log context args
		"--log-context-disruption-name", xargs.DisruptionName,
		"--log-context-disruption-namespace", xargs.DisruptionNamespace,
		"--log-context-target-name", xargs.TargetName,
		"--log-context-target-node-name", xargs.TargetNodeName,
	)

	// enable dry-run mode
	if xargs.DryRun {
		args = append(args, "--dry-run")
	}

	// enable chaos handler init container notification
	if xargs.OnInit {
		args = append(args, "--on-init")
	}

	if xargs.PulseActiveDuration > 0 && xargs.PulseDormantDuration > 0 {
		args = append(args, "--pulse-active-duration", xargs.PulseActiveDuration.String())
		args = append(args, "--pulse-dormant-duration", xargs.PulseDormantDuration.String())
	}

	if xargs.PulseInitialDelay > 0 {
		args = append(args, "--pulse-initial-delay", xargs.PulseInitialDelay.String())
	}

	// DNS disruption configs
	if xargs.Kind == chaostypes.DisruptionKindDNSDisruption {
		args = append(args, "--dns-server", xargs.DNSServer)
		args = append(args, "--kube-dns", xargs.KubeDNS)
	}

	// append allowed hosts for network disruptions
	if xargs.Kind == chaostypes.DisruptionKindNetworkDisruption {
		for _, host := range xargs.AllowedHosts {
			args = append(args, "--allowed-hosts", host)
		}
	}

	return args
}

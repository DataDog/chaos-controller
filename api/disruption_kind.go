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
	HostResolveInterval  time.Duration
	NotInjectedBefore    time.Time
}

// CreateCmdArgs is a helper function generating common and global args and appending them to the given args array
func (d DisruptionArgs) CreateCmdArgs(args []string) []string {
	formattedTargetContainers := []string{}

	for name, id := range d.TargetContainers {
		f := fmt.Sprintf("%s;%s", name, id)
		formattedTargetContainers = append(formattedTargetContainers, f)
	}

	args = append(args,
		// basic args
		"--metrics-sink", d.MetricsSink,
		"--level", string(d.Level),
		"--target-containers", strings.Join(formattedTargetContainers, ","),
		"--target-pod-ip", d.TargetPodIP,
		"--chaos-namespace", d.ChaosNamespace,

		// log context args
		"--log-context-disruption-name", d.DisruptionName,
		"--log-context-disruption-namespace", d.DisruptionNamespace,
		"--log-context-target-name", d.TargetName,
		"--log-context-target-node-name", d.TargetNodeName,
	)

	// enable dry-run mode
	if d.DryRun {
		args = append(args, "--dry-run")
	}

	// enable chaos handler init container notification
	if d.OnInit {
		args = append(args, "--on-init")
	}

	if d.PulseActiveDuration > 0 && d.PulseDormantDuration > 0 {
		args = append(args, "--pulse-active-duration", d.PulseActiveDuration.String())
		args = append(args, "--pulse-dormant-duration", d.PulseDormantDuration.String())
	}

	if d.PulseInitialDelay > 0 {
		args = append(args, "--pulse-initial-delay", d.PulseInitialDelay.String())
	}

	if !d.NotInjectedBefore.IsZero() {
		args = append(args, "--not-injected-before", d.NotInjectedBefore.Format(time.RFC3339))
	}

	// DNS disruption configs
	if d.Kind == chaostypes.DisruptionKindDNSDisruption {
		args = append(args, "--dns-server", d.DNSServer)
		args = append(args, "--kube-dns", d.KubeDNS)
	}

	// append allowed hosts for network disruptions
	if d.Kind == chaostypes.DisruptionKindNetworkDisruption {
		for _, host := range d.AllowedHosts {
			args = append(args, "--allowed-hosts", host)
		}
		if d.HostResolveInterval > 0 {
			args = append(args, "--host-resolve-interval", d.HostResolveInterval.String())
		}
	}

	return args
}

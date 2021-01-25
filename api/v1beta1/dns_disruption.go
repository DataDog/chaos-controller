// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strings"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DNSDisruptionSpec represents a dns disruption
type DNSDisruptionSpec []HostRecordPair

// HostRecordPair represents a hostname and a dns record override
type HostRecordPair struct {
	Hostname string    `json:"hostname"`
	Record   DNSRecord `json:"record"`
}

// DNSRecord represents a type of DNS Record, such as A or CNAME, and the value of that record
type DNSRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Validate validates args for the given disruption
func (s DNSDisruptionSpec) Validate() error {
	for _, pair := range s {
		if pair.Hostname == "" {
			return errors.New("no hostname specified in dns disruption")
		}

		if pair.Record.Type != "A" && pair.Record.Type != "CNAME" {
			return fmt.Errorf("invalid record type specified in dns disruption, must be A or CNAME but found: %s", pair.Record.Type)
		}

		if pair.Record.Value == "" {
			return errors.New("no value specified for dns record in dns disruption")
		}
	}

	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s DNSDisruptionSpec) GenerateArgs(level chaostypes.DisruptionLevel, containerID, sink string, dryRun bool) []string {
	args := []string{
		"dns-disruption",
		"--metrics-sink",
		sink,
		"--level",
		string(level),
		"--container-id",
		containerID,
	}

	// enable dry-run mode
	if dryRun {
		args = append(args, "--dry-run")
	}

	hostRecordPairArgs := []string{}

	for _, pair := range s {
		arg := fmt.Sprintf("%s;%s;%s", pair.Hostname, pair.Record.Type, pair.Record.Value)
		hostRecordPairArgs = append(hostRecordPairArgs, arg)
	}

	args = append(args, "--host-record-pairs")

	// Each value passed to --host-record-pairs should be of the form `hostname;type;value`, e.g.
	// `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
	args = append(args, strings.Split(strings.Join(hostRecordPairArgs, " --host-record-pairs "), " ")...)

	return args
}

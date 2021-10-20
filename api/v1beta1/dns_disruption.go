// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// DNSDisruptionSpec represents a dns disruption
type DNSDisruptionSpec []HostRecordPair

// HostRecordPair represents a hostname and a corresponding dns record override
type HostRecordPair struct {
	Hostname string    `json:"hostname"`
	Record   DNSRecord `json:"record"`
}

// DNSRecord represents a type of DNS Record, such as A or CNAME, and the value of that record
type DNSRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Validate validates that there are no missing hostnames or records for the given dns disruption spec
func (s DNSDisruptionSpec) Validate() error {
	var retErr error = nil
	for _, pair := range s {
		if pair.Hostname == "" {
			retErr = multierror.Append(retErr, errors.New("no hostname specified in dns disruption"))
		}

		if pair.Record.Type != "A" && pair.Record.Type != "CNAME" {
			retErr = multierror.Append(retErr, fmt.Errorf("invalid record type specified in dns disruption, must be A or CNAME but found: %s", pair.Record.Type))
		}

		if pair.Record.Value == "" {
			retErr = multierror.Append(retErr, errors.New("no value specified for dns record in dns disruption"))
		}
	}

	return retErr
}

// GenerateArgs generates injection pod arguments for the given spec
func (s DNSDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"dns-disruption",
	}

	hostRecordPairArgs := []string{}

	for _, pair := range s {
		whiteSpaceCleanedIPList := strings.ReplaceAll(pair.Record.Value, " ", "")
		arg := fmt.Sprintf("%s;%s;%s", pair.Hostname, pair.Record.Type, whiteSpaceCleanedIPList)
		hostRecordPairArgs = append(hostRecordPairArgs, arg)
	}

	args = append(args, "--host-record-pairs")

	// Each value passed to --host-record-pairs should be of the form `hostname;type;value`, e.g.
	// `foo.bar.svc.cluster.local;A;10.0.0.0,10.0.0.13`
	args = append(args, strings.Split(strings.Join(hostRecordPairArgs, " --host-record-pairs "), " ")...)

	return args
}

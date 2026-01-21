// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// DNSDisruptionSpec represents a DNS resolution failure injection
type DNSDisruptionSpec struct {
	// Domains is the list of domain names to disrupt DNS resolution for
	Domains []string `json:"domains" chaos_validate:"required,min=1"`
	// FailureMode determines how DNS queries should fail
	// Supported values: nxdomain, drop, servfail, random-ip
	FailureMode string `json:"failureMode" chaos_validate:"required,oneofci=nxdomain drop servfail random-ip"`
	// Port is the DNS port to intercept (optional, defaults to 53)
	Port int `json:"port,omitempty" chaos_validate:"omitempty,gte=1,lte=65535"`
	// Protocol determines which DNS protocols to disrupt (optional, defaults to both)
	// Supported values: udp, tcp, both
	Protocol string `json:"protocol,omitempty" chaos_validate:"omitempty,oneofci=udp tcp both"`
}

// Validate validates args for the given disruption
func (s *DNSDisruptionSpec) Validate() (retErr error) {
	// Validate domains list is not empty
	if len(s.Domains) == 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("spec.dnsDisruption.domains must contain at least one domain"))
	}

	// Validate each domain is not empty
	for i, domain := range s.Domains {
		if strings.TrimSpace(domain) == "" {
			retErr = multierror.Append(retErr, fmt.Errorf("spec.dnsDisruption.domains[%d] cannot be empty", i))
		}
	}

	// Validate failureMode
	validFailureModes := []string{"nxdomain", "drop", "servfail", "random-ip"}
	isValidMode := false
	normalizedMode := strings.ToLower(s.FailureMode)

	for _, mode := range validFailureModes {
		if normalizedMode == mode {
			isValidMode = true
			break
		}
	}

	if !isValidMode {
		retErr = multierror.Append(retErr,
			fmt.Errorf("spec.dnsDisruption.failureMode must be one of: %s", strings.Join(validFailureModes, ", ")))
	}

	// Validate port if specified
	if s.Port != 0 && (s.Port < 1 || s.Port > 65535) {
		retErr = multierror.Append(retErr, fmt.Errorf("spec.dnsDisruption.port must be between 1 and 65535"))
	}

	// Validate protocol if specified
	if s.Protocol != "" {
		validProtocols := []string{"udp", "tcp", "both"}
		isValidProtocol := false
		normalizedProtocol := strings.ToLower(s.Protocol)

		for _, proto := range validProtocols {
			if normalizedProtocol == proto {
				isValidProtocol = true
				break
			}
		}

		if !isValidProtocol {
			retErr = multierror.Append(retErr,
				fmt.Errorf("spec.dnsDisruption.protocol must be one of: %s", strings.Join(validProtocols, ", ")))
		}
	}

	return retErr
}

// GenerateArgs generates injection pod arguments for the given spec
func (s *DNSDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"dns-disruption",
	}

	// Add domains (comma-separated)
	if len(s.Domains) > 0 {
		args = append(args, "--domains", strings.Join(s.Domains, ","))
	}

	// Add failure mode
	args = append(args, "--failure-mode", strings.ToLower(s.FailureMode))

	// Add port if specified, otherwise use default 53
	port := s.Port
	if port == 0 {
		port = 53
	}
	args = append(args, "--port", fmt.Sprintf("%d", port))

	// Add protocol if specified, otherwise use default "both"
	protocol := s.Protocol
	if protocol == "" {
		protocol = "both"
	}
	args = append(args, "--protocol", strings.ToLower(protocol))

	return args
}

// Explain provides a human-readable explanation of what the disruption will do
func (s *DNSDisruptionSpec) Explain() []string {
	var explanation strings.Builder

	explanation.WriteString("spec.dnsDisruption will cause DNS resolution failures for the following domain(s): ")
	explanation.WriteString(strings.Join(s.Domains, ", "))
	explanation.WriteString(".\n\n")

	switch strings.ToLower(s.FailureMode) {
	case "nxdomain":
		explanation.WriteString("DNS queries will receive an NXDOMAIN response (domain does not exist), ")
		explanation.WriteString("which typically causes applications to fail immediately with a 'domain not found' error.")
	case "drop":
		explanation.WriteString("DNS queries will be dropped without any response, ")
		explanation.WriteString("which causes timeout errors after the application's DNS timeout period expires.")
	case "servfail":
		explanation.WriteString("DNS queries will receive a SERVFAIL response (server failure), ")
		explanation.WriteString("which indicates the DNS server encountered an error processing the query.")
	case "random-ip":
		explanation.WriteString("DNS queries will receive a response with a random/invalid IP address, ")
		explanation.WriteString("which causes applications to attempt connections to unreachable addresses.")
	}

	explanation.WriteString("\n\n")

	port := s.Port
	if port == 0 {
		port = 53
	}
	protocol := s.Protocol
	if protocol == "" {
		protocol = "both"
	}

	explanation.WriteString(fmt.Sprintf("The disruption will intercept DNS traffic on port %d using %s protocol(s).",
		port, protocol))

	return []string{"", explanation.String()}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// DNSDisruptionSpec represents a DNS resolution disruption with flexible record-based configuration
type DNSDisruptionSpec struct {
	// Records defines the list of DNS records to fake/disrupt
	// +kubebuilder:validation:MinItems=1
	Records []DNSRecord `json:"records" chaos_validate:"required,min=1,dive"`

	// Port is the DNS port to intercept (optional, defaults to 53)
	// +optional
	Port int `json:"port,omitempty" chaos_validate:"omitempty,gte=1,lte=65535"`

	// Protocol determines which DNS protocols to disrupt (optional, defaults to both)
	// Supported values: udp, tcp, both
	// +optional
	Protocol string `json:"protocol,omitempty" chaos_validate:"omitempty,oneofci=udp tcp both"`
}

// GetPortWithDefault returns the port to use, applying the default value of 53 if not specified
func (s *DNSDisruptionSpec) GetPortWithDefault() int {
	if s.Port == 0 {
		return 53
	}

	return s.Port
}

// GetProtocolWithDefault returns the protocol to use, applying the default value of "both" if not specified
func (s *DNSDisruptionSpec) GetProtocolWithDefault() string {
	if s.Protocol == "" {
		return "both"
	}

	return strings.ToLower(s.Protocol)
}

// DNSRecord represents a single DNS record to fake
type DNSRecord struct {
	// Hostname is the domain name to intercept (e.g., "app.datadoghq.com")
	Hostname string `json:"hostname" chaos_validate:"required"`
	// Record defines the DNS record configuration to return
	Record DNSRecordConfig `json:"record" chaos_validate:"required"`
}

// DNSRecordConfig defines the DNS record type and value to return
type DNSRecordConfig struct {
	// Type is the DNS record type
	// Supported values: A, AAAA, CNAME, MX, TXT, SRV
	Type string `json:"type" chaos_validate:"required,oneofci=A AAAA CNAME MX TXT SRV"`
	// Value is the record value(s)
	// For A: comma-separated IPv4 addresses or special values (NXDOMAIN, DROP, SERVFAIL, RANDOM)
	// For AAAA: comma-separated IPv6 addresses or special values (NXDOMAIN, DROP, SERVFAIL, RANDOM)
	// For CNAME: target hostname
	// For MX: comma-separated "priority hostname" pairs (e.g., "10 mail.example.com,20 mail2.example.com")
	// For TXT: text content
	// For SRV: comma-separated "priority weight port target" tuples (e.g., "10 60 5060 sipserver.example.com")
	Value string `json:"value" chaos_validate:"required"`
	// TTL is the time-to-live for the record in seconds (optional, defaults to 0)
	// +optional
	TTL uint32 `json:"ttl,omitempty"`
}

// Validate validates args for the given disruption
func (s *DNSDisruptionSpec) Validate() (retErr error) {
	// Validate records
	if err := s.validateRecords(); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	// Validate port if specified
	if s.Port != 0 && (s.Port < 1 || s.Port > 65535) {
		retErr = multierror.Append(retErr, fmt.Errorf("spec.dns.port must be between 1 and 65535"))
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
				fmt.Errorf("spec.dns.protocol must be one of: %s", strings.Join(validProtocols, ", ")))
		}
	}

	return retErr
}

// validateRecords validates the new record-based format
func (s *DNSDisruptionSpec) validateRecords() (retErr error) {
	if len(s.Records) == 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("spec.dns.records must contain at least one record"))
		return retErr
	}

	// Track seen hostname+type combinations to detect duplicates
	// Allow multiple record types for the same hostname (e.g., A and AAAA for dual-stack)
	seenRecords := make(map[string]int)

	for i, record := range s.Records {
		// Validate hostname
		if strings.TrimSpace(record.Hostname) == "" {
			retErr = multierror.Append(retErr, fmt.Errorf("spec.dns.records[%d].hostname cannot be empty", i))
		}

		// Check for duplicate hostname+type combination (normalize the same way as the DNS responder)
		normalizedHostname := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(record.Hostname), "."))
		normalizedType := strings.ToUpper(strings.TrimSpace(record.Record.Type))
		recordKey := normalizedHostname + "|" + normalizedType

		if firstIndex, exists := seenRecords[recordKey]; exists {
			retErr = multierror.Append(retErr,
				fmt.Errorf("spec.dns.records[%d].hostname '%s' with type '%s' is duplicated (first seen at index %d)", i, record.Hostname, record.Record.Type, firstIndex))
		} else {
			seenRecords[recordKey] = i
		}

		// Validate record type (normalizedType already computed above)
		validTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "SRV"}
		isValidType := false

		for _, t := range validTypes {
			if normalizedType == t {
				isValidType = true

				break
			}
		}

		if !isValidType {
			retErr = multierror.Append(retErr,
				fmt.Errorf("spec.dns.records[%d].record.type must be one of: %s", i, strings.Join(validTypes, ", ")))

			continue
		}

		// Validate value based on type
		if err := s.validateRecordValue(i, record.Record.Type, record.Record.Value); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	return retErr
}

// validateRecordValue validates the value based on record type
func (s *DNSDisruptionSpec) validateRecordValue(index int, recordType, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("spec.dns.records[%d].record.value cannot be empty", index)
	}

	normalizedType := strings.ToUpper(recordType)
	upperValue := strings.ToUpper(strings.TrimSpace(value))

	// Check for special values (valid for A and AAAA types)
	specialValues := []string{"NXDOMAIN", "DROP", "SERVFAIL", "RANDOM"}
	isSpecialValue := false

	for _, special := range specialValues {
		if upperValue == special {
			isSpecialValue = true
			break
		}
	}

	switch normalizedType {
	case "A":
		if isSpecialValue {
			return nil
		}
		// Validate comma-separated IPv4 addresses
		ips := strings.Split(value, ",")

		for _, ipStr := range ips {
			ipStr = strings.TrimSpace(ipStr)
			ip := net.ParseIP(ipStr)

			if ip == nil || ip.To4() == nil {
				return fmt.Errorf("spec.dns.records[%d].record.value: '%s' is not a valid IPv4 address or special value", index, ipStr)
			}
		}

	case "AAAA":
		if isSpecialValue {
			return nil
		}
		// Validate comma-separated IPv6 addresses
		ips := strings.Split(value, ",")

		for _, ipStr := range ips {
			ipStr = strings.TrimSpace(ipStr)
			ip := net.ParseIP(ipStr)

			if ip == nil || ip.To4() != nil {
				return fmt.Errorf("spec.dns.records[%d].record.value: '%s' is not a valid IPv6 address or special value", index, ipStr)
			}
		}

	case "CNAME":
		// Just check it's not empty and not a special value
		if isSpecialValue {
			return fmt.Errorf("spec.dns.records[%d].record.value: special values not allowed for CNAME records", index)
		}

	case "MX":
		if isSpecialValue {
			return fmt.Errorf("spec.dns.records[%d].record.value: special values not allowed for MX records", index)
		}
		// Validate format: "priority hostname" pairs separated by commas
		entries := strings.Split(value, ",")
		for _, entry := range entries {
			parts := strings.Fields(strings.TrimSpace(entry))
			if len(parts) != 2 {
				return fmt.Errorf("spec.dns.records[%d].record.value: MX record must be in format 'priority hostname' (e.g., '10 mail.example.com')", index)
			}

			if _, err := strconv.Atoi(parts[0]); err != nil {
				return fmt.Errorf("spec.dns.records[%d].record.value: MX priority must be a number", index)
			}
		}

	case "TXT":
		if isSpecialValue {
			return fmt.Errorf("spec.dns.records[%d].record.value: special values not allowed for TXT records", index)
		}
		// Any string is valid for TXT

	case "SRV":
		if isSpecialValue {
			return fmt.Errorf("spec.dns.records[%d].record.value: special values not allowed for SRV records", index)
		}
		// Validate format: "priority weight port target" tuples separated by commas
		entries := strings.Split(value, ",")
		for _, entry := range entries {
			parts := strings.Fields(strings.TrimSpace(entry))
			if len(parts) != 4 {
				return fmt.Errorf("spec.dns.records[%d].record.value: SRV record must be in format 'priority weight port target' (e.g., '10 60 5060 sipserver.example.com')", index)
			}
			// Validate priority, weight, and port are numbers
			for i, field := range []string{"priority", "weight", "port"} {
				if _, err := strconv.Atoi(parts[i]); err != nil {
					return fmt.Errorf("spec.dns.records[%d].record.value: SRV %s must be a number", index, field)
				}
			}
		}
	}

	return nil
}

// GenerateArgs generates injection pod arguments for the given spec
func (s *DNSDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"dns-disruption",
	}

	// Add records
	if len(s.Records) > 0 {
		// Format: hostname1:type1:value1:ttl1,hostname2:type2:value2:ttl2,...
		var recordArgs []string

		for _, record := range s.Records {
			recordStr := fmt.Sprintf("%s:%s:%s:%d",
				record.Hostname,
				strings.ToUpper(record.Record.Type),
				record.Record.Value,
				record.Record.TTL,
			)
			recordArgs = append(recordArgs, recordStr)
		}

		args = append(args, "--records", strings.Join(recordArgs, ";"))
	}

	// Add port (defaults to 53 if not specified)
	args = append(args, "--port", fmt.Sprintf("%d", s.GetPortWithDefault()))

	// Add protocol (defaults to "both" if not specified)
	args = append(args, "--protocol", s.GetProtocolWithDefault())

	return args
}

// Explain provides a human-readable explanation of what the disruption will do
func (s *DNSDisruptionSpec) Explain() []string {
	var explanation strings.Builder

	if len(s.Records) == 0 {
		return []string{"", "No DNS records configured for disruption."}
	}

	explanation.WriteString("spec.dns will manipulate DNS responses for the following records:\n\n")

	for _, record := range s.Records {
		explanation.WriteString(fmt.Sprintf("• %s (%s record):\n", record.Hostname, strings.ToUpper(record.Record.Type)))

		upperValue := strings.ToUpper(strings.TrimSpace(record.Record.Value))

		switch upperValue {
		case "NXDOMAIN":
			explanation.WriteString("  Returns NXDOMAIN (domain does not exist), causing immediate 'domain not found' errors.\n")
		case "DROP":
			explanation.WriteString("  Drops queries without response, causing timeout errors.\n")
		case "SERVFAIL":
			explanation.WriteString("  Returns SERVFAIL (server failure), indicating DNS server errors.\n")
		case "RANDOM":
			explanation.WriteString("  Returns a random/invalid IP address, causing connections to unreachable addresses.\n")
		default:
			switch strings.ToUpper(record.Record.Type) {
			case "A":
				ips := strings.Split(record.Record.Value, ",")
				if len(ips) > 1 {
					explanation.WriteString(fmt.Sprintf("  Returns IPv4 addresses in round-robin: %s\n", record.Record.Value))
				} else {
					explanation.WriteString(fmt.Sprintf("  Returns IPv4 address: %s\n", record.Record.Value))
				}
			case "AAAA":
				ips := strings.Split(record.Record.Value, ",")
				if len(ips) > 1 {
					explanation.WriteString(fmt.Sprintf("  Returns IPv6 addresses in round-robin: %s\n", record.Record.Value))
				} else {
					explanation.WriteString(fmt.Sprintf("  Returns IPv6 address: %s\n", record.Record.Value))
				}
			case "CNAME":
				explanation.WriteString(fmt.Sprintf("  Returns CNAME pointing to: %s\n", record.Record.Value))
			case "MX":
				explanation.WriteString(fmt.Sprintf("  Returns MX records: %s\n", record.Record.Value))
			case "TXT":
				explanation.WriteString(fmt.Sprintf("  Returns TXT record: %s\n", record.Record.Value))
			case "SRV":
				explanation.WriteString(fmt.Sprintf("  Returns SRV records: %s\n", record.Record.Value))
			}
		}

		if record.Record.TTL > 0 {
			explanation.WriteString(fmt.Sprintf("  TTL: %d seconds\n", record.Record.TTL))
		}

		explanation.WriteString("\n")
	}

	port := s.GetPortWithDefault()
	protocol := s.GetProtocolWithDefault()

	explanation.WriteString(fmt.Sprintf("The disruption will intercept DNS traffic on port %d using %s protocol(s).",
		port, protocol))

	return []string{"", explanation.String()}
}

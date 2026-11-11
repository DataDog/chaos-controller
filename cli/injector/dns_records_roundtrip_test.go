// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"reflect"
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
)

func TestDNSRecordsRoundTrip(t *testing.T) {
	for _, value := range []string{
		"v=DKIM1; p=abc",
		"text:with,delimiters;and;more",
		"quoted \"value\", slash \\, newline\nUnicode: café",
	} {
		t.Run(value, func(t *testing.T) {
			spec := v1beta1.DNSDisruptionSpec{Records: []v1beta1.DNSRecord{
				{Hostname: "example.com", Record: v1beta1.DNSRecordConfig{Type: "TXT", Value: value, TTL: 60}},
				{Hostname: "example.com", Record: v1beta1.DNSRecordConfig{Type: "AAAA", Value: "2001:db8::1,2001:db8::2", TTL: 30}},
			}}
			if err := spec.Validate(); err != nil {
				t.Fatalf("valid DNS record rejected: %v", err)
			}
			args := spec.GenerateArgs()
			records, err := parseRecords(args[2])
			if err != nil {
				t.Fatalf("generated records cannot be parsed: %v", err)
			}
			if !reflect.DeepEqual(records, spec.Records) {
				t.Fatalf("records changed during serialization: got %#v, want %#v", records, spec.Records)
			}
		})
	}
}

func TestParseRecordsRejectsInvalidJSON(t *testing.T) {
	for _, input := range []string{"", "null", "[]", "{}", "not JSON"} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseRecords(input); err == nil {
				t.Fatalf("accepted invalid records %q", input)
			}
		})
	}
}

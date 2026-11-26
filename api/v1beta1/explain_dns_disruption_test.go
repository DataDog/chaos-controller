// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Explain DNS disruption", func() {
	It("includes the DNS records and interception settings in the full spec explanation", func() {
		count := intstr.FromInt(1)
		spec := DisruptionSpec{
			Count: &count, Level: "pod", Duration: "1m",
			Selector: labels.Set{"app": "example"},
			DNS: &DNSDisruptionSpec{
				Port: 5353, Protocol: "udp",
				Records: []DNSRecord{{
					Hostname: "api.example.com",
					Record:   DNSRecordConfig{Type: "A", Value: "192.0.2.7", TTL: 30},
				}},
			},
		}
		Expect(spec.Validate()).To(Succeed())
		explanation := strings.Join(spec.Explain(), "\n")
		for _, detail := range spec.DNS.Explain() {
			if detail != "" {
				Expect(explanation).To(ContainSubstring(detail))
			}
		}
	})
})

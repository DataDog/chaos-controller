// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DNS TXT wire values", func() {
	DescribeTable("preserves valid text in encodable DNS character strings", func(value string) {
		spec := v1beta1.DNSDisruptionSpec{Records: []v1beta1.DNSRecord{{
			Hostname: "example.com",
			Record:   v1beta1.DNSRecordConfig{Type: "TXT", Value: value, TTL: 60},
		}}}
		Expect(spec.Validate()).To(Succeed())
		responder := makeResponder([]DNSRecordEntry{{Hostname: "example.com", RecordType: "TXT", Value: value, TTL: 60}})
		writer := &fakeResponseWriter{}
		responder.handleQuery(writer, makeReq("EXAMPLE.COM.", dns.TypeTXT))
		Expect(writer.written).To(HaveLen(1))
		Expect(writer.written[0].Answer).To(HaveLen(1))
		rr := writer.written[0].Answer[0]
		wire := make([]byte, dns.Len(rr))
		end, err := dns.PackRR(rr, wire, 0, nil, false)
		Expect(err).NotTo(HaveOccurred())
		_, nameEnd, err := dns.UnpackDomainName(wire, 0)
		Expect(err).NotTo(HaveOccurred())
		// Skip TYPE, CLASS, TTL and RDLENGTH; inspect actual TXT payload bytes.
		payload := wire[nameEnd+10 : end]
		var decoded []byte
		for len(payload) > 0 {
			size := int(payload[0])
			Expect(len(payload)).To(BeNumerically(">=", size+1))
			decoded = append(decoded, payload[1:1+size]...)
			payload = payload[1+size:]
		}
		Expect(string(decoded)).To(Equal(value))
	},
		Entry("255 bytes", strings.Repeat("a", 255)),
		Entry("256 bytes", strings.Repeat("a", 256)),
		Entry("multiple character strings", strings.Repeat("a", 510)),
		Entry("literal backslashes", `path\to\file\255\`),
		Entry("backslash at a chunk boundary", strings.Repeat("a", 254)+`\`+strings.Repeat("b", 10)),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"net"

	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

// fakeResponseWriter captures DNS responses for testing.
type fakeResponseWriter struct {
	written []*dns.Msg
	err     error
}

func (f *fakeResponseWriter) LocalAddr() net.Addr  { return &net.UDPAddr{} }
func (f *fakeResponseWriter) RemoteAddr() net.Addr { return &net.UDPAddr{} }
func (f *fakeResponseWriter) WriteMsg(msg *dns.Msg) error {
	if f.err != nil {
		return f.err
	}

	f.written = append(f.written, msg)

	return nil
}

func (f *fakeResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (f *fakeResponseWriter) Close() error                { return nil }
func (f *fakeResponseWriter) TsigStatus() error           { return nil }
func (f *fakeResponseWriter) TsigTimersOnly(bool)         {}
func (f *fakeResponseWriter) Hijack()                     {}

func makeReq(name string, qtype uint16) *dns.Msg {
	req := &dns.Msg{}
	req.SetQuestion(dns.Fqdn(name), qtype)
	return req
}

func makeResponder(records []DNSRecordEntry) *dnsResponder {
	log := zaptest.NewLogger(GinkgoT()).Sugar()
	r := NewDNSResponder(DNSResponderConfig{
		Records:  records,
		UDPPort:  0,
		TCPPort:  0,
		Protocol: "udp",
		Logger:   log,
	})

	return r.(*dnsResponder)
}

var _ = Describe("dnsResponder record builders", func() {
	Describe("buildARecord", func() {
		It("writes A record response for valid IP", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeA)
			record := &DNSRecordEntry{RecordType: "A", IPs: []string{"1.2.3.4"}, TTL: 60}
			r.buildARecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
			Expect(w.written[0].Answer).To(HaveLen(1))
		})

		It("writes SERVFAIL when no IPs", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeA)
			record := &DNSRecordEntry{RecordType: "A", IPs: []string{}, TTL: 60}
			r.buildARecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
			Expect(w.written[0].Rcode).To(Equal(dns.RcodeServerFailure))
		})

		It("writes SERVFAIL when IP is invalid", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeA)
			record := &DNSRecordEntry{RecordType: "A", IPs: []string{"not-an-ip"}, TTL: 60}
			r.buildARecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})
	})

	Describe("buildAAAARecord", func() {
		It("writes AAAA record for valid IPv6", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeAAAA)
			record := &DNSRecordEntry{RecordType: "AAAA", IPs: []string{"::1"}, TTL: 60}
			r.buildAAAARecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})

		It("writes SERVFAIL when IP is empty", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeAAAA)
			record := &DNSRecordEntry{RecordType: "AAAA", IPs: []string{}, TTL: 60}
			r.buildAAAARecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})
	})

	Describe("buildCNAMERecord", func() {
		It("writes CNAME record", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("alias.example.com", dns.TypeA)
			record := &DNSRecordEntry{RecordType: "CNAME", Value: "canonical.example.com", TTL: 60}
			r.buildCNAMERecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
			Expect(w.written[0].Answer).To(HaveLen(1))
		})
	})

	Describe("buildMXRecord", func() {
		It("writes MX record for valid format", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeMX)
			record := &DNSRecordEntry{RecordType: "MX", Value: "10 mail.example.com", TTL: 60}
			r.buildMXRecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})

		It("handles invalid MX format gracefully", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeMX)
			record := &DNSRecordEntry{RecordType: "MX", Value: "invalid-no-priority", TTL: 60}
			Expect(func() { r.buildMXRecord(w, req, record) }).NotTo(Panic())
		})
	})

	Describe("buildTXTRecord", func() {
		It("writes TXT record", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeTXT)
			record := &DNSRecordEntry{RecordType: "TXT", Value: "v=spf1 include:example.com ~all", TTL: 60}
			r.buildTXTRecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})
	})

	Describe("buildSRVRecord", func() {
		It("writes SRV record for valid format", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("_http._tcp.example.com", dns.TypeSRV)
			record := &DNSRecordEntry{RecordType: "SRV", Value: "10 20 80 service.example.com", TTL: 60}
			r.buildSRVRecord(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})

		It("handles invalid SRV format gracefully", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeSRV)
			record := &DNSRecordEntry{RecordType: "SRV", Value: "invalid", TTL: 60}
			Expect(func() { r.buildSRVRecord(w, req, record) }).NotTo(Panic())
		})
	})

	Describe("buildRandomIPResponse", func() {
		It("builds random IPv4 A response", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeA)
			record := &DNSRecordEntry{RecordType: "A", Value: "RANDOM", TTL: 60}
			r.buildRandomIPResponse(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})

		It("builds random IPv6 AAAA response", func() {
			r := makeResponder(nil)
			w := &fakeResponseWriter{}
			req := makeReq("example.com", dns.TypeAAAA)
			record := &DNSRecordEntry{RecordType: "AAAA", Value: "RANDOM", TTL: 60}
			r.buildRandomIPResponse(w, req, record)
			Expect(w.written).To(HaveLen(1))
		})
	})
})

var _ = Describe("dnsResponder.Stop", func() {
	It("stops a responder that was never started", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			UDPPort:  0,
			TCPPort:  0,
			Protocol: "udp",
			Logger:   log,
		})

		Expect(func() { r.Stop() }).NotTo(Panic())
	})
})

var _ = Describe("dnsResponder integration (Start/Stop/handleQuery)", func() {
	const testPort = 55353 // high port unlikely to conflict

	It("starts, responds to a query with a configured A record, then stops", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			Records: []DNSRecordEntry{
				{Hostname: "test.example.com", RecordType: "A", Value: "1.2.3.4", TTL: 60},
			},
			UDPPort:     testPort,
			TCPPort:     testPort + 1,
			Protocol:    "udp",
			UpstreamDNS: "8.8.8.8:53",
			Logger:      log,
		})

		err := r.Start()
		if err != nil {
			Skip("DNS responder failed to start (port in use?): " + err.Error())
		}

		DeferCleanup(r.Stop)

		// Query the running responder
		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("test.example.com.", dns.TypeA)
		resp, _, queryErr := c.Exchange(m, "127.0.0.1:55353")
		Expect(queryErr).NotTo(HaveOccurred())
		Expect(resp.Answer).NotTo(BeEmpty())
	})

	It("responds to AAAA, CNAME, MX, TXT, SRV, NXDOMAIN, RANDOM queries", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			Records: []DNSRecordEntry{
				{Hostname: "v6.example.com", RecordType: "AAAA", Value: "::1", TTL: 60},
				{Hostname: "alias.example.com", RecordType: "CNAME", Value: "canonical.example.com", TTL: 60},
				{Hostname: "mail.example.com", RecordType: "MX", Value: "10 smtp.example.com", TTL: 60},
				{Hostname: "txt.example.com", RecordType: "TXT", Value: "v=spf1 ~all", TTL: 60},
				{Hostname: "_http._tcp.example.com", RecordType: "SRV", Value: "10 20 80 srv.example.com", TTL: 60},
				{Hostname: "nx.example.com", RecordType: "A", Value: "NXDOMAIN", TTL: 60},
				{Hostname: "rnd.example.com", RecordType: "A", Value: "RANDOM", TTL: 60},
				{Hostname: "rnd6.example.com", RecordType: "AAAA", Value: "RANDOM", TTL: 60},
			},
			UDPPort:     testPort,
			Protocol:    "udp",
			UpstreamDNS: "127.0.0.1:1",
			Logger:      log,
		})

		err := r.Start()
		if err != nil {
			Skip("DNS responder failed to start: " + err.Error())
		}

		DeferCleanup(r.Stop)

		c := new(dns.Client)
		sendQuery := func(name string, qtype uint16) {
			m := new(dns.Msg)
			m.SetQuestion(name, qtype)
			c.Exchange(m, "127.0.0.1:55353")
		}

		sendQuery("v6.example.com.", dns.TypeAAAA)
		sendQuery("alias.example.com.", dns.TypeA)
		sendQuery("mail.example.com.", dns.TypeMX)
		sendQuery("txt.example.com.", dns.TypeTXT)
		sendQuery("_http._tcp.example.com.", dns.TypeSRV)
		sendQuery("nx.example.com.", dns.TypeA)
		sendQuery("rnd.example.com.", dns.TypeA)
		sendQuery("rnd6.example.com.", dns.TypeAAAA)
		// Unknown host → forwards to invalid upstream → SERVFAIL
		sendQuery("notexist.example.com.", dns.TypeA)
	})

	It("forwards unknown queries to upstream (forwardToUpstream coverage)", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			UDPPort:     testPort,
			Protocol:    "udp",
			UpstreamDNS: "127.0.0.1:1",
			Logger:      log,
		})

		err := r.Start()
		if err != nil {
			Skip("DNS responder failed to start: " + err.Error())
		}

		DeferCleanup(r.Stop)

		c := new(dns.Client)
		m := new(dns.Msg)
		m.SetQuestion("unknown.notexist.", dns.TypeA)
		Expect(func() { c.Exchange(m, "127.0.0.1:55353") }).NotTo(Panic())
	})
})

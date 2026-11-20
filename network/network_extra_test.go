// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("NewBPFTCFilterConfigExecutor", func() {
	It("creates executor with dry-run=false", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := NewBPFTCFilterConfigExecutor(log, false)
		Expect(exec).NotTo(BeNil())
	})

	It("dry-run mode returns 0 without running binary", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := NewBPFTCFilterConfigExecutor(log, true)
		code, stdout, err := exec.Run([]string{"some-arg"})
		Expect(err).NotTo(HaveOccurred())
		Expect(code).To(Equal(0))
		Expect(stdout).To(BeEmpty())
	})

	It("non-dry-run returns error when binary not found", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := NewBPFTCFilterConfigExecutor(log, false)
		_, _, err := exec.Run([]string{"some-arg"})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("defaultTcExecutor dry-run", func() {
	It("dry-run returns 0 without running tc", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		exec := defaultTcExecutor{log: log, dryRun: true}
		code, out, err := exec.Run([]string{"qdisc", "show"})
		Expect(err).NotTo(HaveOccurred())
		Expect(code).To(Equal(0))
		Expect(out).To(BeEmpty())
	})
})

var _ = Describe("NewDNSClient", func() {
	It("creates client with defaults when config is empty", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		client := NewDNSClient(DNSClientConfig{Logger: log})
		Expect(client).NotTo(BeNil())
	})

	It("creates client with custom paths", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		client := NewDNSClient(DNSClientConfig{
			PodResolvConfPath:  "/tmp/pod-resolv.conf",
			NodeResolvConfPath: "/tmp/node-resolv.conf",
			Logger:             log,
		})
		Expect(client).NotTo(BeNil())
	})
})

var _ = Describe("dnsClient.getResolversAndNames", func() {
	var (
		log      = zaptest.NewLogger(GinkgoT()).Sugar()
		tmpDir   string
		podConf  string
		nodeConf string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "dns-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)

		podConf = filepath.Join(tmpDir, "pod-resolv.conf")
		nodeConf = filepath.Join(tmpDir, "node-resolv.conf")
		Expect(os.WriteFile(podConf, []byte("nameserver 1.1.1.1\n"), 0o644)).To(Succeed())
		Expect(os.WriteFile(nodeConf, []byte("nameserver 8.8.8.8\n"), 0o644)).To(Succeed())
	})

	newClient := func() dnsClient {
		return dnsClient{
			podResolvConfPath:  podConf,
			nodeResolvConfPath: nodeConf,
			log:                log,
		}
	}

	It("strategy=pod uses pod resolvers", func() {
		c := newClient()
		resolvers, names, err := c.getResolversAndNames("example.com", DNSResolverPod)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolvers).To(ContainElement("1.1.1.1"))
		Expect(names).NotTo(BeEmpty())
	})

	It("strategy=node uses node resolvers", func() {
		c := newClient()
		resolvers, names, err := c.getResolversAndNames("example.com", DNSResolverNode)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolvers).To(ContainElement("8.8.8.8"))
		Expect(names).NotTo(BeEmpty())
	})

	It("strategy=pod-fallback-node combines both", func() {
		c := newClient()
		resolvers, _, err := c.getResolversAndNames("example.com", DNSResolverPodFallbackNode)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolvers).To(ContainElement("1.1.1.1"))
		Expect(resolvers).To(ContainElement("8.8.8.8"))
	})

	It("strategy=node-fallback-pod puts node first", func() {
		c := newClient()
		resolvers, _, err := c.getResolversAndNames("example.com", DNSResolverNodeFallbackPod)
		Expect(err).NotTo(HaveOccurred())
		Expect(resolvers[0]).To(Equal("8.8.8.8"))
	})

	It("empty strategy defaults to pod-fallback-node", func() {
		c := newClient()
		resolvers, _, err := c.getResolversAndNames("example.com", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolvers).NotTo(BeEmpty())
	})

	It("unknown strategy returns error", func() {
		c := newClient()
		_, _, err := c.getResolversAndNames("example.com", "unknown")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown DNS resolver strategy"))
	})

	It("returns error when pod resolv.conf missing for pod strategy", func() {
		c := newClient()
		c.podResolvConfPath = "/nonexistent/resolv.conf"
		_, _, err := c.getResolversAndNames("example.com", DNSResolverPod)
		Expect(err).To(HaveOccurred())
	})

	It("returns error when node resolv.conf missing for node strategy", func() {
		c := newClient()
		c.nodeResolvConfPath = "/nonexistent/resolv.conf"
		_, _, err := c.getResolversAndNames("example.com", DNSResolverNode)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("dnsClient.resolve", func() {
	It("returns error for unknown protocol", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		c := dnsClient{log: log}
		_, err := c.resolve("example.com.", "ftp", []string{"8.8.8.8"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown protocol"))
	})
})

var _ = Describe("NewDNSResponder", func() {
	It("creates responder with defaults", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			UDPPort:  5353,
			TCPPort:  5354,
			Protocol: "both",
			Logger:   log,
		})
		Expect(r).NotTo(BeNil())
	})

	It("creates responder with A records", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			Records: []DNSRecordEntry{
				{Hostname: "example.com", RecordType: "A", Value: "1.2.3.4,5.6.7.8", TTL: 60},
				{Hostname: "v6.example.com", RecordType: "AAAA", Value: "::1", TTL: 60},
				{Hostname: "alias.example.com", RecordType: "CNAME", Value: "example.com", TTL: 60},
			},
			UDPPort:  5353,
			TCPPort:  5354,
			Protocol: "udp",
			Logger:   log,
		})
		Expect(r).NotTo(BeNil())
	})

	It("creates responder with custom upstream", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r := NewDNSResponder(DNSResponderConfig{
			UpstreamDNS: "1.1.1.1:53,8.8.8.8:53",
			UDPPort:     5353,
			TCPPort:     5354,
			Protocol:    "tcp",
			Logger:      log,
		})
		Expect(r).NotTo(BeNil())
	})
})

var _ = Describe("dnsResponder.findRecord", func() {
	var r *dnsResponder

	BeforeEach(func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		r = &dnsResponder{
			logger: log,
			records: map[string]*DNSRecordEntry{
				"example.com|A":     {Hostname: "example.com", RecordType: "A", Value: "1.2.3.4"},
				"example.com|AAAA":  {Hostname: "example.com", RecordType: "AAAA", Value: "::1"},
				"example.com|CNAME": {Hostname: "example.com", RecordType: "CNAME", Value: "canonical.com"},
				"*.wildcard.com|A":  {Hostname: "*.wildcard.com", RecordType: "A", Value: "5.6.7.8"},
				"api.example.com|A": {Hostname: "api.example.com", RecordType: "A", Value: "9.10.11.12"},
			},
		}
	})

	It("exact match returns record", func() {
		rec := r.findRecord("example.com", "A")
		Expect(rec).NotTo(BeNil())
		Expect(rec.Value).To(Equal("1.2.3.4"))
	})

	It("exact AAAA match returns record", func() {
		rec := r.findRecord("example.com", "AAAA")
		Expect(rec).NotTo(BeNil())
		Expect(rec.Value).To(Equal("::1"))
	})

	It("no match returns nil", func() {
		rec := r.findRecord("unknown.example.org", "A")
		Expect(rec).To(BeNil())
	})

	It("CNAME fallback for A query", func() {
		rec := r.findRecord("example.com", "A")
		Expect(rec).NotTo(BeNil())
	})

	It("subdomain match with longer pattern wins", func() {
		rec := r.findRecord("api.example.com", "A")
		Expect(rec).NotTo(BeNil())
		Expect(rec.Value).To(Equal("9.10.11.12"))
	})

	It("trailing dot normalized", func() {
		rec := r.findRecord("example.com.", "A")
		Expect(rec).NotTo(BeNil())
		Expect(rec.Value).To(Equal("1.2.3.4"))
	})
})

var _ = Describe("dnsResponder.getNextIP", func() {
	var r *dnsResponder

	BeforeEach(func() {
		r = &dnsResponder{}
	})

	It("returns empty string for no IPs", func() {
		record := &DNSRecordEntry{IPs: []string{}}
		Expect(r.getNextIP(record)).To(BeEmpty())
	})

	It("returns single IP without round-robin", func() {
		record := &DNSRecordEntry{IPs: []string{"1.2.3.4"}}
		Expect(r.getNextIP(record)).To(Equal("1.2.3.4"))
	})

	It("round-robins multiple IPs", func() {
		record := &DNSRecordEntry{IPs: []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}, IPIndex: 0}
		first := r.getNextIP(record)
		second := r.getNextIP(record)
		third := r.getNextIP(record)
		Expect(first).NotTo(Equal(second))
		Expect(second).NotTo(Equal(third))
		// Wraps around
		fourth := r.getNextIP(record)
		Expect(fourth).To(Equal(first))
	})
})

var _ = Describe("NewConnState", func() {
	DescribeTable("converts string to connState",
		func(input string, expected connState) {
			Expect(NewConnState(input)).To(Equal(expected))
		},
		Entry("new", "new", ConnStateNew),
		Entry("est", "est", ConnStateEstablished),
		Entry("empty string → undefined", "", ConnStateUndefined),
		Entry("unknown → undefined", "unknown", ConnStateUndefined),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

// DNSResponder interface for testing
type DNSResponder interface {
	Start() error
	Stop() error
}

// DNSRecordEntry represents a single DNS record configuration for the responder
type DNSRecordEntry struct {
	Hostname   string
	RecordType string // A, AAAA, CNAME, MX, TXT, SRV
	Value      string // Raw value from config
	TTL        uint32

	// Internal fields for round-robin (A and AAAA records)
	IPs     []string // Parsed IPs for round-robin
	IPIndex int      // Current index for round-robin
	ipMutex sync.Mutex
}

// dnsResponder is a lightweight DNS server that can manipulate DNS responses
type dnsResponder struct {
	records            map[string]*DNSRecordEntry // Map of "hostname|type" -> record (allows multiple types per hostname)
	udpPort            int                        // UDP server port
	tcpPort            int                        // TCP server port
	protocol           string
	upstreamDNSServers []string // Multiple upstream DNS servers for redundancy
	logger             *zap.SugaredLogger
	udpServer          *dns.Server
	tcpServer          *dns.Server
}

// DNSResponderConfig holds the configuration for creating a DNSResponder
type DNSResponderConfig struct {
	Records     []DNSRecordEntry
	UDPPort     int
	TCPPort     int
	Protocol    string
	UpstreamDNS string
	Logger      *zap.SugaredLogger
}

// NewDNSResponder creates a new DNS responder with the given configuration
func NewDNSResponder(config DNSResponderConfig) DNSResponder {
	upstreamDNS := config.UpstreamDNS

	if upstreamDNS == "" {
		// Default to Google's public DNS
		upstreamDNS = "8.8.8.8:53"
	}

	// Parse comma-separated list of upstream DNS servers
	upstreamServers := []string{}

	for _, server := range strings.Split(upstreamDNS, ",") {
		server = strings.TrimSpace(server)
		if server != "" {
			upstreamServers = append(upstreamServers, server)
		}
	}

	// Build record map keyed by "hostname|type" to support multiple record types per hostname
	recordMap := make(map[string]*DNSRecordEntry, len(config.Records))

	for i := range config.Records {
		record := &config.Records[i]
		// Normalize hostname (trim whitespace, lowercase, remove trailing dot)
		hostname := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(record.Hostname), "."))
		recordType := strings.ToUpper(strings.TrimSpace(record.RecordType))

		// Parse IPs for A and AAAA records with multiple values
		if record.RecordType == "A" || record.RecordType == "AAAA" {
			upperValue := strings.ToUpper(strings.TrimSpace(record.Value))

			// Only parse IPs if not a special value
			if upperValue != "NXDOMAIN" && upperValue != "DROP" && upperValue != "SERVFAIL" && upperValue != "RANDOM" {
				record.IPs = strings.Split(record.Value, ",")

				for j := range record.IPs {
					record.IPs[j] = strings.TrimSpace(record.IPs[j])
				}

				record.IPIndex = 0
			}
		}

		// Key by hostname|type to allow multiple record types per hostname (e.g., A and AAAA)
		recordKey := hostname + "|" + recordType
		recordMap[recordKey] = record
	}

	return &dnsResponder{
		records:            recordMap,
		udpPort:            config.UDPPort,
		tcpPort:            config.TCPPort,
		protocol:           strings.ToLower(config.Protocol),
		upstreamDNSServers: upstreamServers,
		logger:             config.Logger,
	}
}

// Start starts the DNS responder servers (UDP and/or TCP based on protocol config)
func (r *dnsResponder) Start() error {
	r.logger.Infow("starting DNS responder",
		tags.UDPPortKey, r.udpPort,
		tags.TCPPortKey, r.tcpPort,
		tags.ProtocolKey, r.protocol,
		tags.DNSRecordCountKey, len(r.records),
	)

	// Create dedicated ServeMux instead of using global handler
	serveMux := dns.NewServeMux()
	serveMux.HandleFunc(".", r.handleQuery)

	// Use channels to signal startup completion or errors
	type startResult struct {
		protocol string
		err      error
	}

	resultChan := make(chan startResult, 2)
	expectedServers := 0

	// Start UDP server if needed
	if r.protocol == "udp" || r.protocol == "both" {
		expectedServers++

		udpAddr := fmt.Sprintf("0.0.0.0:%d", r.udpPort)

		r.udpServer = &dns.Server{
			Addr:    udpAddr,
			Net:     "udp",
			Handler: serveMux,
		}

		go func() {
			r.logger.Infow("DNS responder UDP server starting", tags.AddressKey, udpAddr)

			// Signal when server starts successfully
			r.udpServer.NotifyStartedFunc = func() {
				r.logger.Debugw("DNS responder UDP server started")
				resultChan <- startResult{protocol: "udp", err: nil}
			}

			if err := r.udpServer.ListenAndServe(); err != nil {
				r.logger.Errorw("DNS responder UDP server error", tags.ErrorKey, err)
				// Signal error if server failed to start (NotifyStartedFunc was never called)
				select {
				case resultChan <- startResult{protocol: "udp", err: err}:
				default:
				}
			}
		}()
	}

	// Start TCP server if needed
	if r.protocol == "tcp" || r.protocol == "both" {
		expectedServers++

		tcpAddr := fmt.Sprintf("0.0.0.0:%d", r.tcpPort)

		r.tcpServer = &dns.Server{
			Addr:    tcpAddr,
			Net:     "tcp",
			Handler: serveMux,
		}

		go func() {
			r.logger.Infow("DNS responder TCP server starting", tags.AddressKey, tcpAddr)

			// Signal when server starts successfully
			r.tcpServer.NotifyStartedFunc = func() {
				r.logger.Debugw("DNS responder TCP server started")
				resultChan <- startResult{protocol: "tcp", err: nil}
			}

			if err := r.tcpServer.ListenAndServe(); err != nil {
				r.logger.Errorw("DNS responder TCP server error", tags.ErrorKey, err)
				// Signal error if server failed to start (NotifyStartedFunc was never called)
				select {
				case resultChan <- startResult{protocol: "tcp", err: err}:
				default:
				}
			}
		}()
	}

	// Wait for all servers to start or fail
	var startErrors []error

	startedServers := make(map[string]bool) // Track which servers started successfully

	for i := 0; i < expectedServers; i++ {
		result := <-resultChan
		if result.err != nil {
			startErrors = append(startErrors, fmt.Errorf("%s server failed to start: %w", result.protocol, result.err))
		} else {
			startedServers[result.protocol] = true
		}
	}

	if len(startErrors) > 0 {
		// Cleanup: shutdown any servers that did start successfully
		r.logger.Warnw("partial startup failure, shutting down successfully started servers",
			tags.ErrorKey, startErrors,
			"started_servers", startedServers,
		)

		if startedServers["udp"] && r.udpServer != nil {
			if err := r.udpServer.Shutdown(); err != nil {
				r.logger.Errorw("error shutting down UDP server during cleanup", tags.ErrorKey, err)
			}

			r.udpServer = nil
		}

		if startedServers["tcp"] && r.tcpServer != nil {
			if err := r.tcpServer.Shutdown(); err != nil {
				r.logger.Errorw("error shutting down TCP server during cleanup", tags.ErrorKey, err)
			}

			r.tcpServer = nil
		}

		return fmt.Errorf("DNS responder startup failed: %v", startErrors)
	}

	r.logger.Infow("DNS responder started")

	return nil
}

// Stop gracefully shuts down the DNS responder servers
func (r *dnsResponder) Stop() error {
	r.logger.Infow("stopping DNS responder")

	var errs []error

	// Shutdown UDP server if running
	if r.udpServer != nil {
		if err := r.udpServer.Shutdown(); err != nil {
			r.logger.Errorw("error shutting down UDP server", tags.ErrorKey, err)
			errs = append(errs, fmt.Errorf("UDP shutdown error: %w", err))
		}
	}

	// Shutdown TCP server if running
	if r.tcpServer != nil {
		if err := r.tcpServer.Shutdown(); err != nil {
			r.logger.Errorw("error shutting down TCP server", tags.ErrorKey, err)
			errs = append(errs, fmt.Errorf("TCP shutdown error: %w", err))
		}
	}

	r.logger.Infow("DNS responder stopped")

	if len(errs) > 0 {
		return fmt.Errorf("errors during DNS responder shutdown: %v", errs)
	}

	return nil
}

// handleQuery processes incoming DNS queries and applies the configured record manipulation
func (r *dnsResponder) handleQuery(w dns.ResponseWriter, req *dns.Msg) {
	// Ensure there's at least one question in the query
	if len(req.Question) == 0 {
		r.logger.Warnw("received DNS query with no questions")
		return
	}

	// Extract queried domain (remove trailing dot if present)
	queriedDomain := strings.TrimSuffix(req.Question[0].Name, ".")
	qtype := req.Question[0].Qtype

	r.logger.Debugw("received DNS query",
		tags.DNSDomainKey, queriedDomain,
		tags.DNSQueryTypeKey, dns.TypeToString[qtype],
	)

	// Find matching record based on queried domain and query type
	queryTypeName := dns.TypeToString[qtype]
	record := r.findRecord(queriedDomain, queryTypeName)

	if record == nil {
		r.logger.Debugw("no matching record found, forwarding to upstream",
			tags.DNSDomainKey, queriedDomain,
			tags.DNSQueryTypeKey, queryTypeName,
			tags.DNSUpstreamsKey, r.upstreamDNSServers,
		)
		// Not a configured domain/type - forward to upstream DNS resolver
		r.forwardToUpstream(w, req)

		return
	}

	r.logger.Infow("disrupting DNS query",
		tags.DNSDomainKey, queriedDomain,
		tags.DNSRecordTypeKey, record.RecordType,
		tags.ValueKey, record.Value,
	)

	// Check for special values (NXDOMAIN, DROP, SERVFAIL, RANDOM)
	upperValue := strings.ToUpper(strings.TrimSpace(record.Value))
	switch upperValue {
	case "NXDOMAIN":
		r.logger.Debugw("returning NXDOMAIN")

		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeNameError)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing NXDOMAIN response", tags.ErrorKey, err)
		}

		return

	case "DROP":
		r.logger.Debugw("dropping DNS query (no response)")

		return // Don't respond - causes timeout

	case "SERVFAIL":
		r.logger.Debugw("returning SERVFAIL")

		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing SERVFAIL response", tags.ErrorKey, err)
		}

		return

	case "RANDOM":
		r.logger.Debugw("returning random IP")
		r.buildRandomIPResponse(w, req, record)

		return
	}

	// Build response based on record type
	switch strings.ToUpper(record.RecordType) {
	case "A":
		r.buildARecord(w, req, record)
	case "AAAA":
		r.buildAAAARecord(w, req, record)
	case "CNAME":
		r.buildCNAMERecord(w, req, record)
	case "MX":
		r.buildMXRecord(w, req, record)
	case "TXT":
		r.buildTXTRecord(w, req, record)
	case "SRV":
		r.buildSRVRecord(w, req, record)
	default:
		r.logger.Errorw("unknown record type", tags.TypeKey, record.RecordType)

		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing SERVFAIL response", tags.ErrorKey, err)

			return
		}
	}
}

// findRecord finds the DNS record for a given hostname and query type
// For overlapping records (e.g., example.com and api.example.com), prefers the longest (most specific) match
// CNAME records are returned for A/AAAA queries (standard DNS behavior)
func (r *dnsResponder) findRecord(queriedDomain string, queryType string) *DNSRecordEntry {
	// Normalize the queried domain (lowercase, remove trailing dot)
	queriedDomain = strings.ToLower(strings.TrimSuffix(queriedDomain, "."))

	// Direct lookup (exact hostname + query type match)
	recordKey := queriedDomain + "|" + queryType
	if record, found := r.records[recordKey]; found {
		return record
	}

	// For A/AAAA queries, also check for CNAME records (standard DNS behavior)
	// When a resolver queries for an A record and gets a CNAME, it follows the CNAME
	if queryType == "A" || queryType == "AAAA" {
		cnameKey := queriedDomain + "|CNAME"
		if record, found := r.records[cnameKey]; found {
			return record
		}
	}

	// Check for wildcard/subdomain matches
	// Collect all matching hostnames and select the longest (most specific) one
	var (
		bestMatch  string
		bestRecord *DNSRecordEntry
	)

	for recordKey, record := range r.records {
		// Extract hostname from "hostname|type" key
		parts := strings.SplitN(recordKey, "|", 2)
		if len(parts) != 2 {
			continue
		}

		hostname := parts[0]
		recordType := parts[1]

		// Only match if type matches query type (or CNAME for A/AAAA queries)
		matchesType := recordType == queryType
		if !matchesType && (queryType == "A" || queryType == "AAAA") && recordType == "CNAME" {
			matchesType = true
		}

		if !matchesType {
			continue
		}

		// Check if queried domain is a subdomain of configured hostname
		// e.g., if hostname is "example.com", match "www.example.com"
		if strings.HasSuffix(queriedDomain, "."+hostname) {
			// Prefer longer (more specific) matches
			// e.g., for query "foo.api.example.com":
			// - "example.com" (length 11) vs "api.example.com" (length 15)
			// - select "api.example.com" as it's more specific
			if len(hostname) > len(bestMatch) {
				bestMatch = hostname
				bestRecord = record
			}
		}
	}

	return bestRecord
}

// getNextIP returns the next IP in round-robin fashion (thread-safe)
func (r *dnsResponder) getNextIP(record *DNSRecordEntry) string {
	if len(record.IPs) == 0 {
		return ""
	}

	if len(record.IPs) == 1 {
		return record.IPs[0]
	}

	record.ipMutex.Lock()
	defer record.ipMutex.Unlock()

	ip := record.IPs[record.IPIndex]
	record.IPIndex = (record.IPIndex + 1) % len(record.IPs)

	return ip
}

// buildRandomIPResponse builds a response with a random IP (IPv4 for A, IPv6 for AAAA)
func (r *dnsResponder) buildRandomIPResponse(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	recordType := strings.ToUpper(record.RecordType)

	if recordType == "AAAA" {
		// Generate a random IPv6 in the 2001:db8::/32 range (documentation prefix, RFC 3849)
		// Format: 2001:db8:xxxx:xxxx::1
		randomIP := fmt.Sprintf("2001:db8:%x:%x::1", rand.Intn(65536), rand.Intn(65536))

		rr := &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    record.TTL,
			},
			AAAA: net.ParseIP(randomIP),
		}

		msg.Answer = append(msg.Answer, rr)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing random-IPv6 response", tags.ErrorKey, err, tags.IPKey, randomIP)

			return
		}

		r.logger.Debugw("sent random-IPv6 response", tags.IPKey, randomIP)
	} else {
		// Default to A record with IPv4
		// Generate a random IP in the 192.0.2.0/24 range (TEST-NET-1, RFC 5737)
		randomIP := fmt.Sprintf("192.0.2.%d", rand.Intn(256))

		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    record.TTL,
			},
			A: net.ParseIP(randomIP),
		}

		msg.Answer = append(msg.Answer, rr)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing random-IP response", tags.ErrorKey, err, tags.IPKey, randomIP)

			return
		}

		r.logger.Debugw("sent random-IP response", tags.IPKey, randomIP)
	}
}

// buildARecord builds an A record response
func (r *dnsResponder) buildARecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	// Get next IP (round-robin if multiple IPs)
	ipStr := r.getNextIP(record)

	if ipStr == "" {
		r.logger.Errorw("no IP addresses available for A record")
		msg.SetRcode(req, dns.RcodeServerFailure)
		_ = w.WriteMsg(msg)

		return
	}

	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		r.logger.Errorw("invalid IPv4 address", tags.IPKey, ipStr)
		msg.SetRcode(req, dns.RcodeServerFailure)
		_ = w.WriteMsg(msg)

		return
	}

	rr := &dns.A{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    record.TTL,
		},
		A: ip,
	}

	msg.Answer = append(msg.Answer, rr)

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing A record response", tags.ErrorKey, err, tags.IPKey, ipStr)

		return
	}

	r.logger.Debugw("sent A record response", tags.IPKey, ipStr)
}

// buildAAAARecord builds an AAAA record response (IPv6)
func (r *dnsResponder) buildAAAARecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	// Get next IP (round-robin if multiple IPs)
	ipStr := r.getNextIP(record)

	if ipStr == "" {
		r.logger.Errorw("no IP addresses available for AAAA record")
		msg.SetRcode(req, dns.RcodeServerFailure)
		_ = w.WriteMsg(msg)

		return
	}

	ip := net.ParseIP(ipStr)

	if ip == nil {
		r.logger.Errorw("invalid IPv6 address", tags.IPKey, ipStr)
		msg.SetRcode(req, dns.RcodeServerFailure)
		_ = w.WriteMsg(msg)

		return
	}

	rr := &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    record.TTL,
		},
		AAAA: ip,
	}

	msg.Answer = append(msg.Answer, rr)

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing AAAA record response", tags.ErrorKey, err, tags.IPKey, ipStr)

		return
	}

	r.logger.Debugw("sent AAAA record response", tags.IPKey, ipStr)
}

// buildCNAMERecord builds a CNAME record response
func (r *dnsResponder) buildCNAMERecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	target := strings.TrimSpace(record.Value)

	if !strings.HasSuffix(target, ".") {
		target += "."
	}

	rr := &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    record.TTL,
		},
		Target: target,
	}

	msg.Answer = append(msg.Answer, rr)

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing CNAME record response", tags.ErrorKey, err, tags.DNSTargetKey, target)

		return
	}

	r.logger.Debugw("sent CNAME record response", tags.DNSTargetKey, target)
}

// buildMXRecord builds an MX record response
func (r *dnsResponder) buildMXRecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	// Parse MX entries: "priority hostname,priority hostname,..."
	entries := strings.Split(record.Value, ",")
	for _, entry := range entries {
		parts := strings.Fields(strings.TrimSpace(entry))
		if len(parts) != 2 {
			r.logger.Errorw("invalid MX record format", tags.DNSEntryKey, entry)

			continue
		}

		priority, err := strconv.Atoi(parts[0])
		if err != nil {
			r.logger.Errorw("invalid MX priority", tags.DNSPriorityKey, parts[0])

			continue
		}

		mailHost := parts[1]

		if !strings.HasSuffix(mailHost, ".") {
			mailHost += "."
		}

		rr := &dns.MX{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeMX,
				Class:  dns.ClassINET,
				Ttl:    record.TTL,
			},
			Preference: uint16(priority),
			Mx:         mailHost,
		}

		msg.Answer = append(msg.Answer, rr)
	}

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing MX record response", tags.ErrorKey, err)

		return
	}

	r.logger.Debugw("sent MX record response", tags.CountKey, len(msg.Answer))
}

// buildTXTRecord builds a TXT record response
func (r *dnsResponder) buildTXTRecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	rr := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    record.TTL,
		},
		Txt: []string{record.Value},
	}

	msg.Answer = append(msg.Answer, rr)

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing TXT record response", tags.ErrorKey, err)

		return
	}

	r.logger.Debugw("sent TXT record response", tags.DNSTextKey, record.Value)
}

// buildSRVRecord builds an SRV record response
func (r *dnsResponder) buildSRVRecord(w dns.ResponseWriter, req *dns.Msg, record *DNSRecordEntry) {
	msg := new(dns.Msg)
	msg.SetReply(req)

	// Parse SRV entries: "priority weight port target,priority weight port target,..."
	entries := strings.Split(record.Value, ",")
	for _, entry := range entries {
		parts := strings.Fields(strings.TrimSpace(entry))
		if len(parts) != 4 {
			r.logger.Errorw("invalid SRV record format", tags.DNSEntryKey, entry)

			continue
		}

		priority, err := strconv.Atoi(parts[0])
		if err != nil {
			r.logger.Errorw("invalid SRV priority", tags.DNSPriorityKey, parts[0])

			continue
		}

		weight, err := strconv.Atoi(parts[1])
		if err != nil {
			r.logger.Errorw("invalid SRV weight", tags.DNSWeightKey, parts[1])

			continue
		}

		port, err := strconv.Atoi(parts[2])
		if err != nil {
			r.logger.Errorw("invalid SRV port", tags.PortKey, parts[2])

			continue
		}

		target := parts[3]

		if !strings.HasSuffix(target, ".") {
			target += "."
		}

		rr := &dns.SRV{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    record.TTL,
			},
			Priority: uint16(priority),
			Weight:   uint16(weight),
			Port:     uint16(port),
			Target:   target,
		}

		msg.Answer = append(msg.Answer, rr)
	}

	if err := w.WriteMsg(msg); err != nil {
		r.logger.Errorw("error writing SRV record response", tags.ErrorKey, err)

		return
	}

	r.logger.Debugw("sent SRV record response", tags.CountKey, len(msg.Answer))
}

// forwardToUpstream forwards a DNS query to the upstream DNS resolver
func (r *dnsResponder) forwardToUpstream(w dns.ResponseWriter, req *dns.Msg) {
	client := &dns.Client{
		Net: w.RemoteAddr().Network(), // Use same protocol (udp/tcp) as the incoming request
	}

	// Try each upstream DNS server in order until one succeeds
	var lastErr error

	for _, upstream := range r.upstreamDNSServers {
		resp, _, err := client.Exchange(req, upstream)
		if err != nil {
			r.logger.Debugw("error forwarding DNS query to upstream, trying next",
				tags.ErrorKey, err,
				tags.DNSUpstreamKey, upstream,
				tags.DNSDomainKey, req.Question[0].Name,
			)

			lastErr = err

			continue
		}

		// Successfully got response, forward it back to the client
		if err := w.WriteMsg(resp); err != nil {
			r.logger.Errorw("error writing forwarded DNS response", tags.ErrorKey, err)
		}

		return
	}

	// All upstream servers failed
	r.logger.Errorw("all upstream DNS servers failed",
		tags.ErrorKey, lastErr,
		tags.DNSUpstreamsKey, r.upstreamDNSServers,
		tags.DNSDomainKey, req.Question[0].Name,
	)

	// Return SERVFAIL when all upstreams fail
	msg := new(dns.Msg)
	msg.SetRcode(req, dns.RcodeServerFailure)

	if writeErr := w.WriteMsg(msg); writeErr != nil {
		r.logger.Errorw("error writing SERVFAIL response", tags.ErrorKey, writeErr)
	}
}

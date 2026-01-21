// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

// DNSResponder is a lightweight DNS server that can simulate various DNS failure modes
type DNSResponder struct {
	targetDomains []string
	failureMode   string
	port          int
	protocol      string
	upstreamDNS   string
	logger        *zap.SugaredLogger
	udpServer     *dns.Server
	tcpServer     *dns.Server
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// DNSResponderConfig holds the configuration for creating a DNSResponder
type DNSResponderConfig struct {
	TargetDomains []string
	FailureMode   string
	Port          int
	Protocol      string
	UpstreamDNS   string
	Logger        *zap.SugaredLogger
}

// NewDNSResponder creates a new DNS responder with the given configuration
func NewDNSResponder(config DNSResponderConfig) *DNSResponder {
	upstreamDNS := config.UpstreamDNS
	if upstreamDNS == "" {
		// Default to Google's public DNS
		upstreamDNS = "8.8.8.8:53"
	}

	return &DNSResponder{
		targetDomains: config.TargetDomains,
		failureMode:   strings.ToLower(config.FailureMode),
		port:          config.Port,
		protocol:      strings.ToLower(config.Protocol),
		upstreamDNS:   upstreamDNS,
		logger:        config.Logger,
		stopCh:        make(chan struct{}),
	}
}

// Start starts the DNS responder servers (UDP and/or TCP based on protocol config)
func (r *DNSResponder) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", r.port)

	r.logger.Infow("starting DNS responder",
		"address", addr,
		"protocol", r.protocol,
		"failureMode", r.failureMode,
		"targetDomains", r.targetDomains,
	)

	// Create DNS handler
	dns.HandleFunc(".", r.handleQuery)

	// Start UDP server if needed
	if r.protocol == "udp" || r.protocol == "both" {
		r.udpServer = &dns.Server{
			Addr: addr,
			Net:  "udp",
		}

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.logger.Infow("DNS responder UDP server starting", "address", addr)
			if err := r.udpServer.ListenAndServe(); err != nil {
				r.logger.Errorw("DNS responder UDP server error", "error", err)
			}
		}()
	}

	// Start TCP server if needed
	if r.protocol == "tcp" || r.protocol == "both" {
		r.tcpServer = &dns.Server{
			Addr: addr,
			Net:  "tcp",
		}

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.logger.Infow("DNS responder TCP server starting", "address", addr)
			if err := r.tcpServer.ListenAndServe(); err != nil {
				r.logger.Errorw("DNS responder TCP server error", "error", err)
			}
		}()
	}

	r.logger.Infow("DNS responder started successfully")

	return nil
}

// Stop gracefully shuts down the DNS responder servers
func (r *DNSResponder) Stop() error {
	r.logger.Infow("stopping DNS responder")

	close(r.stopCh)

	var errs []error

	// Shutdown UDP server if running
	if r.udpServer != nil {
		if err := r.udpServer.Shutdown(); err != nil {
			r.logger.Errorw("error shutting down UDP server", "error", err)
			errs = append(errs, fmt.Errorf("UDP shutdown error: %w", err))
		}
	}

	// Shutdown TCP server if running
	if r.tcpServer != nil {
		if err := r.tcpServer.Shutdown(); err != nil {
			r.logger.Errorw("error shutting down TCP server", "error", err)
			errs = append(errs, fmt.Errorf("TCP shutdown error: %w", err))
		}
	}

	// Wait for servers to stop
	r.wg.Wait()

	r.logger.Infow("DNS responder stopped")

	if len(errs) > 0 {
		return fmt.Errorf("errors during DNS responder shutdown: %v", errs)
	}

	return nil
}

// handleQuery processes incoming DNS queries and applies the configured failure mode
func (r *DNSResponder) handleQuery(w dns.ResponseWriter, req *dns.Msg) {
	// Ensure there's at least one question in the query
	if len(req.Question) == 0 {
		r.logger.Warnw("received DNS query with no questions")
		return
	}

	// Extract queried domain (remove trailing dot if present)
	queriedDomain := strings.TrimSuffix(req.Question[0].Name, ".")

	r.logger.Debugw("received DNS query",
		"domain", queriedDomain,
		"qtype", dns.TypeToString[req.Question[0].Qtype],
	)

	// Check if this domain should be disrupted
	if !r.matchesDomain(queriedDomain) {
		r.logger.Debugw("domain not in target list, forwarding to upstream",
			"domain", queriedDomain,
			"upstream", r.upstreamDNS,
		)
		// Not a target domain - forward to upstream DNS resolver
		r.forwardToUpstream(w, req)
		return
	}

	r.logger.Infow("disrupting DNS query",
		"domain", queriedDomain,
		"failureMode", r.failureMode,
	)

	// Apply failure mode
	switch r.failureMode {
	case "drop":
		// Don't respond - this causes a timeout
		r.logger.Debugw("dropping DNS query (no response)")
		return

	case "nxdomain":
		// Return NXDOMAIN (domain does not exist)
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeNameError)
		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing NXDOMAIN response", "error", err)
		}

	case "servfail":
		// Return SERVFAIL (server failure)
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)
		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing SERVFAIL response", "error", err)
		}

	case "random-ip":
		// Return a random/invalid IP address
		msg := new(dns.Msg)
		msg.SetReply(req)

		// Generate a random IP in the 192.0.2.0/24 range (TEST-NET-1, RFC 5737)
		// This range is reserved for documentation and should not be routable
		randomIP := fmt.Sprintf("192.0.2.%d", rand.Intn(256))

		// Create an A record with the random IP
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   req.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0, // No caching
			},
			A: net.ParseIP(randomIP),
		}

		msg.Answer = append(msg.Answer, rr)

		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing random-IP response", "error", err, "ip", randomIP)
		} else {
			r.logger.Debugw("sent random-IP response", "ip", randomIP)
		}

	default:
		r.logger.Errorw("unknown failure mode", "mode", r.failureMode)
		// Return SERVFAIL for unknown modes
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)
		if err := w.WriteMsg(msg); err != nil {
			r.logger.Errorw("error writing SERVFAIL response", "error", err)
		}
	}
}

// forwardToUpstream forwards a DNS query to the upstream DNS resolver
func (r *DNSResponder) forwardToUpstream(w dns.ResponseWriter, req *dns.Msg) {
	client := &dns.Client{
		Net: w.RemoteAddr().Network(), // Use same protocol (udp/tcp) as the incoming request
	}

	resp, _, err := client.Exchange(req, r.upstreamDNS)
	if err != nil {
		r.logger.Errorw("error forwarding DNS query to upstream",
			"error", err,
			"upstream", r.upstreamDNS,
			"domain", req.Question[0].Name,
		)
		// Return SERVFAIL on forwarding error
		msg := new(dns.Msg)
		msg.SetRcode(req, dns.RcodeServerFailure)
		if writeErr := w.WriteMsg(msg); writeErr != nil {
			r.logger.Errorw("error writing SERVFAIL response", "error", writeErr)
		}
		return
	}

	// Forward the upstream response back to the client
	if err := w.WriteMsg(resp); err != nil {
		r.logger.Errorw("error writing forwarded DNS response", "error", err)
	}
}

// matchesDomain checks if the queried domain matches any of the target domains
func (r *DNSResponder) matchesDomain(queriedDomain string) bool {
	// Normalize the queried domain (lowercase, remove trailing dot)
	queriedDomain = strings.ToLower(strings.TrimSuffix(queriedDomain, "."))

	for _, targetDomain := range r.targetDomains {
		// Normalize target domain
		targetDomain = strings.ToLower(strings.TrimSuffix(targetDomain, "."))

		// Exact match
		if queriedDomain == targetDomain {
			return true
		}

		// Check if queried domain is a subdomain of target
		// e.g., if target is "example.com", match "www.example.com"
		if strings.HasSuffix(queriedDomain, "."+targetDomain) {
			return true
		}
	}

	return false
}

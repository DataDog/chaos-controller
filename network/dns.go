// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/avast/retry-go"
	"github.com/miekg/dns"
)

type DnsConfig struct {
	DnsServer string
	KubeDns   string
}

// DNSClient is a client being able to resolve the given host
type DNSClient interface {
	Resolve(host string) ([]net.IP, error)
}

type dnsClient struct{}

// NewDNSClient creates a standard DNS client
func NewDNSClient() DNSClient {
	return dnsClient{}
}

func (c dnsClient) Resolve(host string) ([]net.IP, error) {
	ips := []net.IP{}

	// read resolv conf file to get search domain
	// and other dns configurations
	dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("can't read resolve.conf file: %w", err)
	}

	// do the request on the first configured dns resolver
	dnsClient := dns.Client{}
	response := &dns.Msg{}

	err = retry.Do(func() error {
		// query possible resolvers and fqdn based on servers and search domains specified in the dns configuration
		for _, name := range dnsConfig.NameList(host) {
			dnsMessage := dns.Msg{}
			dnsMessage.SetQuestion(name, dns.TypeA)

			for _, server := range dnsConfig.Servers {
				response, _, err = dnsClient.Exchange(&dnsMessage, fmt.Sprintf("%s:53", server))
				if response != nil && len(response.Answer) > 0 {
					return nil
				}
			}
		}

		return err
	}, retry.Attempts(3))
	if err != nil {
		return nil, fmt.Errorf("can't resolve the given hostname %s: %w", host, err)
	}

	// parse returned records
	for _, answer := range response.Answer {
		if ip, ok := answer.(*dns.A); ok {
			ips = append(ips, ip.A)
		}
	}

	// error if no A records can be found
	if len(ips) == 0 {
		return nil, fmt.Errorf("no A records were found for the given hostname %s", host)
	}

	return ips, nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/avast/retry-go"
	"github.com/miekg/dns"
)

type DNSConfig struct {
	DNSServer string
	KubeDNS   string
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

	// NOTE: we read both the pod and the node DNS configurations here
	// in case some of the given hosts are not resolvable from a pod

	// read the pod resolv conf file to get search domain
	// and other dns configurations
	podDNSConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("can't read '/etc/resolv.conf' file: %w", err)
	}

	// also read the node resolv conf file to get search domain
	// and other dns configurations
	nodeDNSConfig, err := dns.ClientConfigFromFile("/mnt/host/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("can't read '/mnt/host/etc/resolv.conf' file: %w", err)
	}

	// compute resolvers list
	resolvers := append([]string{}, podDNSConfig.Servers...)
	resolvers = append(resolvers, nodeDNSConfig.Servers...)

	// compute possible names to resolve
	names := append([]string{}, podDNSConfig.NameList(host)...)
	names = append(names, nodeDNSConfig.NameList(host)...)

	// do the request on the first configured dns resolver
	dnsClient := dns.Client{}
	response := &dns.Msg{}

	err = retry.Do(func() error {
		// query possible resolvers and fqdn based on servers and search domains specified in the dns configuration
		for _, name := range names {
			dnsMessage := dns.Msg{}
			dnsMessage.SetQuestion(name, dns.TypeA)

			for _, server := range resolvers {
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

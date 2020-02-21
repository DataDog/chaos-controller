// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type DNSClient interface {
	Resolve(host string) ([]net.IP, error)
}

type dnsClient struct{}

func NewDNSClient() DNSClient {
	return dnsClient{}
}

func (c dnsClient) Resolve(host string) ([]net.IP, error) {
	// read resolv conf file to get search domain
	// and other dns configurations
	dnsConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, fmt.Errorf("can't read resolve.conf file: %w", err)
	}

	// do the request on the first configured dns resolver
	dnsClient := dns.Client{}
	dnsMessage := dns.Msg{}
	dnsMessage.SetQuestion(host+".", dns.TypeA)

	response, _, err := dnsClient.Exchange(&dnsMessage, dnsConfig.Servers[0]+":53")
	if err != nil {
		return nil, fmt.Errorf("can't resolve the given hostname %s: %w", host, err)
	}

	// parse returned records
	ips := []net.IP{}
	for _, answer := range response.Answer {
		ips = append(ips, answer.(*dns.A).A)
	}

	return ips, nil
}

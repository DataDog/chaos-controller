// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"net"

	"github.com/DataDog/chaos-controller/network"
)

// resolveHost tries to resolve the given host
// it tries to resolve it as a CIDR, as a single IP, or as a hostname
// it returns a list of IP or an error if it fails to resolve the hostname
func resolveHost(client network.DNSClient, host string) ([]*net.IPNet, error) {
	var ips []*net.IPNet

	// return the wildcard 0.0.0.0/0 CIDR if the given host is an empty string
	if host == "" {
		_, null, _ := net.ParseCIDR("0.0.0.0/0")

		return []*net.IPNet{null}, nil
	}

	// try to parse the given host as a CIDR
	_, ipnet, err := net.ParseCIDR(host)
	if err != nil {
		// if it fails, try to parse the given host as a single IP
		ip := net.ParseIP(host)
		if ip == nil {
			// if no IP has been parsed, fallback on a hostname
			// and try to resolve it by using the container resolv.conf file
			resolvedIPs, err := client.Resolve(host)
			if err != nil {
				return nil, fmt.Errorf("can't resolve the given host with the configured dns resolver: %w", err)
			}

			for _, resolvedIP := range resolvedIPs {
				ips = append(ips, &net.IPNet{
					IP:   resolvedIP,
					Mask: net.CIDRMask(32, 32),
				})
			}
		} else {
			// ensure the parsed IP is an IPv4
			// the net.ParseIP function returns an IPv4 with an IPv6 length
			// the code blow ensures the parsed IP prefix is the default (empty) prefix
			// of an IPv6 address:
			// var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
			var a, b [12]byte
			copy(a[:], ip[0:12])
			b[10] = 0xff
			b[11] = 0xff
			if a != b {
				return nil, fmt.Errorf("the given IP (%s) seems to be an IPv6, aborting", host)
			}

			// use a /32 mask for a single IP
			ips = append(ips, &net.IPNet{
				IP:   ip[12:16],
				Mask: net.CIDRMask(32, 32),
			})
		}
	} else {
		// use the given CIDR network
		ips = append(ips, ipnet)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to resolve the given host: %s", host)
	}

	return ips, nil
}

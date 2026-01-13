// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"net"
	"sort"
)

// ipWithHash represents an IP address paired with its hash value for sorting
type ipWithHash struct {
	ip   *net.IPNet
	hash uint64
}

// SelectIPsByPercentage selects a subset of IPs based on percentage using consistent hashing.
// The seed parameter ensures consistent selection across different disruption targets.
// Returns the selected IPs in deterministic order based on their hash values.
//
// Parameters:
//   - ips: The list of IP addresses to select from
//   - percentage: The percentage of IPs to select (1-100)
//   - seed: A string used for consistent hashing (e.g., "hostname+targetPodName")
//
// Returns:
//   - A subset of IPs representing the requested percentage
func SelectIPsByPercentage(ips []*net.IPNet, percentage int, seed string) []*net.IPNet {
	if percentage <= 0 || percentage > 100 {
		return ips
	}

	if len(ips) == 0 {
		return ips
	}

	// If percentage is 100, return all IPs
	if percentage == 100 {
		return ips
	}

	// Calculate how many IPs to select
	count := int(math.Ceil(float64(len(ips)) * float64(percentage) / 100.0))
	if count >= len(ips) {
		return ips
	}

	// Hash each IP with the seed
	ipHashes := make([]ipWithHash, len(ips))
	for i, ip := range ips {
		ipHashes[i] = ipWithHash{
			ip:   ip,
			hash: hashIPWithSeed(ip.String(), seed),
		}
	}

	// Sort by hash value for deterministic selection
	sort.Slice(ipHashes, func(i, j int) bool {
		return ipHashes[i].hash < ipHashes[j].hash
	})

	// Select the first N IPs after sorting by hash
	result := make([]*net.IPNet, count)
	for i := 0; i < count; i++ {
		result[i] = ipHashes[i].ip
	}

	return result
}

// hashIPWithSeed creates a deterministic hash from an IP string and a seed
func hashIPWithSeed(ip string, seed string) uint64 {
	h := sha256.New()
	h.Write([]byte(ip))
	h.Write([]byte(seed))
	hash := h.Sum(nil)

	// Convert first 8 bytes to uint64 for sorting
	return binary.BigEndian.Uint64(hash[:8])
}

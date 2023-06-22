// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

type protocol string

const (
	TCP protocol = "tcp"
	UDP protocol = "udp"
	ARP protocol = "arp"
	ALL protocol = "*"
)

func (p protocol) String() string {
	return string(p)
}

type protocolString interface {
	string | v1.Protocol | protocol
}

func AllProtocols[C protocolString](p C) []protocol {
	prtcl := newProtocol(p)
	if prtcl == ALL {
		return []protocol{
			TCP,
			UDP,
		}
	}

	return []protocol{prtcl}
}

// NewProtocol returns a protocol value based on the given string, possible values are:
// - all: by default if value is not recognised
// - tcp: if provided value is tcp in any case
// - udp: if provided value is udp in any case
// - arp: if provided value is arp in any case
func newProtocol[C protocolString](protocol C) protocol {
	switch strings.ToLower(string(protocol)) {
	case string(UDP):
		return UDP
	case string(ARP):
		return ARP
	case string(TCP):
		return TCP
	default:
		return ALL
	}
}

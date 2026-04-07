// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package bpfdisrupt

// Direction indicates the traffic direction a rule applies to.
type Direction uint32

const (
	DirEgress  Direction = 1
	DirIngress Direction = 2
)

func (d Direction) String() string {
	switch d {
	case DirEgress:
		return "egress"
	case DirIngress:
		return "ingress"
	default:
		return "unknown"
	}
}

// Action defines what the BPF program should do with matched packets.
type Action uint32

const (
	// ActionAllow skips disruption (used for safeguard/allowed hosts).
	ActionAllow Action = 0
	// ActionDisrupt routes to netem band (egress) or redirects to IFB (ingress).
	ActionDisrupt Action = 1
	// ActionDrop drops the packet (ingress only, with optional probability).
	ActionDrop Action = 2
)

func (a Action) String() string {
	switch a {
	case ActionAllow:
		return "allow"
	case ActionDisrupt:
		return "disrupt"
	case ActionDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// Rule represents a single disruption rule to be loaded into the BPF LPM trie map.
type Rule struct {
	// Direction indicates whether this rule applies to egress or ingress traffic.
	Direction Direction
	// CIDR is the IP range to match (e.g., "10.0.0.1/32" or "0.0.0.0/0").
	CIDR string
	// Action defines what to do with matched packets.
	Action Action
	// DropPct is the drop probability (0-100), only used with ActionDrop.
	DropPct int
	// Port is the L4 port to match (0 = match all ports).
	Port int
	// Protocol is the IP protocol to match: "tcp", "udp", or "" (match all).
	Protocol string
}

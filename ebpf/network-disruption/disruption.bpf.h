// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

#include "../includes/bpf_common.h"
#include "../includes/bpf_endian.h"

// TC action return codes (not defined in vmlinux.h)
#define TC_ACT_OK       0
#define TC_ACT_SHOT     2

// Ethernet header length
#define ETH_HLEN        14

// Ethernet protocol types
#define ETH_P_IP        0x0800
#define ETH_P_IPV6      0x86DD

// IP protocol numbers
#define IPPROTO_TCP     6
#define IPPROTO_UDP     17

// Maximum entries in the LPM trie
#define MAX_DISRUPTION_ENTRIES 4096

// Actions stored in LPM trie values
#define ACTION_ALLOW    0  // Skip this packet (safeguard/allowed host)
#define ACTION_DISRUPT  1  // Route to disruption band (egress) or redirect to IFB (ingress)
#define ACTION_DROP     2  // Drop packet (ingress only)

// Direction bitmask
#define DIR_EGRESS      1
#define DIR_INGRESS     2

// LPM trie key: prefix length + IPv6 address (IPv4 stored as ::ffff:x.x.x.x)
struct lpm_key {
    __u32 prefix_len;
    __u8  ip[16];
};

// LPM trie value: action parameters with optional L4 matching
struct lpm_val {
    __u32 action;        // ACTION_ALLOW, ACTION_DISRUPT, or ACTION_DROP
    __u32 drop_pct;      // 0-100, percentage for probabilistic drop (ACTION_DROP only)
    __u32 ifb_ifindex;   // IFB device ifindex for redirect (ACTION_DISRUPT ingress only)
    __u16 src_port;      // Source port to match (0 = match all)
    __u16 dst_port;      // Destination port to match (0 = match all)
    __u8  protocol;      // IP protocol to match: IPPROTO_TCP, IPPROTO_UDP (0 = match all)
    __u8  _pad[3];       // Padding to align struct to 4 bytes
};

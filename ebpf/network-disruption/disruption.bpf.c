// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

// +build ignore
#include "disruption.bpf.h"

// Shared LPM trie map for both egress and ingress disruption rules.
// Key: {prefix_len, ip[16]} where IPv4 is stored as IPv4-mapped IPv6 (::ffff:x.x.x.x).
// Value: {action, drop_pct, ifb_ifindex}.
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, MAX_DISRUPTION_ENTRIES);
    __uint(map_flags, BPF_F_NO_PREALLOC);
    __type(key, struct lpm_key);
    __type(value, struct lpm_val);
} disruption_rules SEC(".maps");

// Convert an IPv4 address (in network byte order) to an IPv4-mapped IPv6 LPM key.
// The key prefix_len is set to 128 for exact /32 match.
static __always_inline void ipv4_to_lpm_key(struct lpm_key *key, __u32 addr) {
    __builtin_memset(key, 0, sizeof(*key));
    // IPv4-mapped IPv6: ::ffff:x.x.x.x
    // Bytes 10-11 are 0xff, bytes 12-15 are the IPv4 address
    key->ip[10] = 0xff;
    key->ip[11] = 0xff;
    // Copy IPv4 address bytes (already in network byte order)
    __builtin_memcpy(&key->ip[12], &addr, 4);
    // Prefix length: full 128-bit match for /32 IPv4
    // The actual prefix_len will be overridden by the map entry for CIDR matching
    key->prefix_len = 128;
}

// Convert an IPv6 address to an LPM key.
static __always_inline void ipv6_to_lpm_key(struct lpm_key *key, __u8 addr[16]) {
    __builtin_memset(key, 0, sizeof(*key));
    __builtin_memcpy(key->ip, addr, 16);
    key->prefix_len = 128;
}

// Check if the packet's L4 header matches the rule's port/protocol constraints.
// Returns true if the packet matches (or if the rule has no L4 constraints).
static __always_inline bool match_l4(struct __sk_buff *skb, __u16 eth_proto,
                                     struct lpm_val *val) {
    // No L4 constraints: match all
    if (val->protocol == 0 && val->src_port == 0 && val->dst_port == 0)
        return true;

    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    __u8 ip_proto = 0;
    __u32 l4_off = 0;

    if (eth_proto == ETH_P_IP) {
        if (data + ETH_HLEN + 20 > data_end)
            return false;
        // IP protocol is at byte 9 of the IP header
        ip_proto = *(__u8 *)(data + ETH_HLEN + 9);
        // IHL (IP header length) is the lower nibble of byte 0, in 4-byte units
        __u8 ihl = (*(__u8 *)(data + ETH_HLEN)) & 0x0f;
        l4_off = ETH_HLEN + ((__u32)ihl * 4);
    } else if (eth_proto == ETH_P_IPV6) {
        if (data + ETH_HLEN + 40 > data_end)
            return false;
        // Next header is at byte 6 of the IPv6 header
        ip_proto = *(__u8 *)(data + ETH_HLEN + 6);
        l4_off = ETH_HLEN + 40;
    } else {
        return false;
    }

    // Check protocol if specified
    if (val->protocol != 0 && val->protocol != ip_proto)
        return false;

    // Check ports for TCP/UDP only
    if (val->src_port != 0 || val->dst_port != 0) {
        if (ip_proto != IPPROTO_TCP && ip_proto != IPPROTO_UDP)
            return false; // Port matching only for TCP/UDP

        // TCP and UDP both have src_port at offset 0, dst_port at offset 2
        if (data + l4_off + 4 > data_end)
            return false;

        __u16 pkt_src_port = bpf_ntohs(*(__u16 *)(data + l4_off));
        __u16 pkt_dst_port = bpf_ntohs(*(__u16 *)(data + l4_off + 2));

        if (val->src_port != 0 && val->src_port != pkt_src_port)
            return false;
        if (val->dst_port != 0 && val->dst_port != pkt_dst_port)
            return false;
    }

    return true;
}

// TC egress classifier: looks up dst_ip in the LPM trie.
// Returns -1 to use the command-line flowid (1:4, disruption band) on match.
// Returns 0 (no match) to let the packet flow to the default prio band.
// This follows the same pattern as classifier_methods in injection.bpf.c.
SEC("tc_egress_disruption")
int cls_egress_disruption(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // Parse Ethernet header
    if (data + ETH_HLEN > data_end)
        return 0;

    __u16 eth_proto = bpf_ntohs(*(__u16 *)(data + ETH_HLEN - 2));

    struct lpm_key key;

    if (eth_proto == ETH_P_IP) {
        // IPv4: dst_ip is at offset 16 in the IP header
        if (data + ETH_HLEN + 20 > data_end)
            return 0;

        __u32 dst_ip;
        __builtin_memcpy(&dst_ip, data + ETH_HLEN + 16, 4);
        ipv4_to_lpm_key(&key, dst_ip);
    } else if (eth_proto == ETH_P_IPV6) {
        // IPv6: dst_ip is at offset 24 in the IPv6 header (16 bytes)
        if (data + ETH_HLEN + 40 > data_end)
            return 0;

        __u8 dst_ip[16];
        __builtin_memcpy(dst_ip, data + ETH_HLEN + 24, 16);
        ipv6_to_lpm_key(&key, dst_ip);
    } else {
        // Not IP traffic, skip
        return 0;
    }

    struct lpm_val *val = bpf_map_lookup_elem(&disruption_rules, &key);
    if (!val)
        return 0; // No match, default band

    if (val->action == ACTION_ALLOW)
        return 0; // Safeguard/allowed host, skip disruption

    // Check L4 (port/protocol) constraints if specified
    if (!match_l4(skb, eth_proto, val))
        return 0; // L4 mismatch, skip disruption

    if (val->action == ACTION_DISRUPT || val->action == ACTION_DROP) {
        // Return -1 to apply the tc rule (use cmd-line flowid 1:4)
        return -1;
    }

    return 0;
}

// TC ingress classifier with DirectAction: looks up src_ip in the LPM trie.
// This is the true ingress filter - matches the real originating IP, not cluster VIPs.
// Returns TC_ACT_SHOT to drop, bpf_redirect() for IFB shaping, or TC_ACT_OK to pass.
SEC("tc_ingress_disruption")
int cls_ingress_disruption(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // Parse Ethernet header
    if (data + ETH_HLEN > data_end)
        return TC_ACT_OK;

    __u16 eth_proto = bpf_ntohs(*(__u16 *)(data + ETH_HLEN - 2));

    struct lpm_key key;

    if (eth_proto == ETH_P_IP) {
        // IPv4: src_ip is at offset 12 in the IP header
        if (data + ETH_HLEN + 20 > data_end)
            return TC_ACT_OK;

        __u32 src_ip;
        __builtin_memcpy(&src_ip, data + ETH_HLEN + 12, 4);
        ipv4_to_lpm_key(&key, src_ip);
    } else if (eth_proto == ETH_P_IPV6) {
        // IPv6: src_ip is at offset 8 in the IPv6 header (16 bytes)
        if (data + ETH_HLEN + 40 > data_end)
            return TC_ACT_OK;

        __u8 src_ip[16];
        __builtin_memcpy(src_ip, data + ETH_HLEN + 8, 16);
        ipv6_to_lpm_key(&key, src_ip);
    } else {
        // Not IP traffic, pass through
        return TC_ACT_OK;
    }

    struct lpm_val *val = bpf_map_lookup_elem(&disruption_rules, &key);
    if (!val)
        return TC_ACT_OK; // No match, pass through

    if (val->action == ACTION_ALLOW)
        return TC_ACT_OK; // Safeguard/allowed host

    // Check L4 (port/protocol) constraints if specified
    if (!match_l4(skb, eth_proto, val))
        return TC_ACT_OK; // L4 mismatch, pass through

    if (val->action == ACTION_DROP) {
        // Probabilistic drop using BPF random number generator
        if (val->drop_pct >= 100)
            return TC_ACT_SHOT;

        if (val->drop_pct > 0) {
            __u32 rand = bpf_get_prandom_u32();
            // Scale: rand % 100 gives 0-99, drop if < drop_pct
            if ((rand % 100) < val->drop_pct)
                return TC_ACT_SHOT;
        }

        return TC_ACT_OK;
    }

    if (val->action == ACTION_DISRUPT) {
        // Redirect to IFB device for shaping (delay, jitter, bandwidth, corruption)
        if (val->ifb_ifindex > 0)
            return bpf_redirect(val->ifb_ifindex, 0);

        // No IFB configured, drop as fallback
        return TC_ACT_SHOT;
    }

    return TC_ACT_OK;
}

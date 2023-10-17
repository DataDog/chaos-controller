// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

#include "./bpf_endian.h"

// This determines the size of the payload fragment that is captured for each HTTP request
#define DEFAULT_HTTP_BUFFER_SIZE 25
#define LARGE_HTTP_BUFFER_SIZE 100

// IPv6 over bluebook
#define ETH_P_IPV6      0x86DD
// Internet Protocol packet
#define ETH_P_IP        0x0800
// Total octets in header.
#define ETH_HLEN        14

#define TCP_FLAGS_OFFSET 13

typedef enum
{
    HTTP_PACKET_UNKNOWN,
    HTTP_REQUEST,
    HTTP_RESPONSE
} http_packet_t;

typedef struct {
    /* Using the type unsigned __int128 generates an error in the ebpf verifier */
    __u64 saddr_h;
    __u64 saddr_l;
    __u64 daddr_h;
    __u64 daddr_l;
    __u16 sport;
    __u16 dport;
    __u32 netns;
    __u32 pid;
    // Metadata description:
    // First bit indicates if the connection is TCP (1) or UDP (0)
    // Second bit indicates if the connection is V6 (1) or V4 (0)
    __u32 metadata; // This is that big because it seems that we atleast need a 32-bit aligned struct
} conn_tuple_t;

// skb_info_t embeds a conn_tuple_t extracted from the skb object as well as
// some ancillary data such as the data offset (the byte offset pointing to
// where the application payload begins) and the TCP flags if applicable.
// This struct is populated by calling `read_conn_tuple_skb` from a program type
// that manipulates a `__sk_buff` object.
typedef struct {
    conn_tuple_t tup;
    __u32 data_off;
    __u8 tcp_flags;
} skb_info_t;


unsigned long long load_byte(void *skb, unsigned long long off)
	asm ("llvm.bpf.load.byte");

unsigned long long load_half(void *skb, unsigned long long off)
	asm ("llvm.bpf.load.half");

unsigned long long load_word(void *skb, unsigned long long off)
	asm ("llvm.bpf.load.word");

static __always_inline void read_ipv4_skb(struct __sk_buff *skb, __u64 off, __u64 *addr) {
    *addr = load_word(skb, off);
    *addr = bpf_ntohll(*addr) >> 32;
}

// Metadata bit masks
// 0 << x is only for readability
typedef enum
{
    // Connection type
    CONN_TYPE_UDP = 0,
    CONN_TYPE_TCP = 1,

    // Connection family
    CONN_V4 = 0 << 1,
    CONN_V6 = 1 << 1,
} metadata_mask_t;

static __always_inline void read_ipv6_skb(struct __sk_buff *skb, __u64 off, __u64 *addr_l, __u64 *addr_h) {
    *addr_h |= (__u64)load_word(skb, off) << 32;
    *addr_h |= (__u64)load_word(skb, off + 4);
    *addr_h = bpf_ntohll(*addr_h);

    *addr_l |= (__u64)load_word(skb, off + 8) << 32;
    *addr_l |= (__u64)load_word(skb, off + 12);
    *addr_l = bpf_ntohll(*addr_l);
}

static __always_inline __u64 read_conn_tuple_skb(struct __sk_buff *skb, skb_info_t *info) {
    __builtin_memset(info, 0, sizeof(skb_info_t));
    info->data_off = ETH_HLEN;

    __u16 l3_proto = load_half(skb, offsetof(struct ethhdr, h_proto));
    __u8 l4_proto = 0;
    switch (l3_proto) {
    case ETH_P_IP:
        l4_proto = load_byte(skb, info->data_off + offsetof(struct iphdr, protocol));
        info->tup.metadata |= CONN_V4;
        read_ipv4_skb(skb, info->data_off + offsetof(struct iphdr, saddr), &info->tup.saddr_l);
        read_ipv4_skb(skb, info->data_off + offsetof(struct iphdr, daddr), &info->tup.daddr_l);
        info->data_off += sizeof(struct iphdr); // TODO: this assumes there are no IP options
        break;
    case ETH_P_IPV6:
        l4_proto = load_byte(skb, info->data_off + offsetof(struct ipv6hdr, nexthdr));
        info->tup.metadata |= CONN_V6;
        read_ipv6_skb(skb, info->data_off + offsetof(struct ipv6hdr, saddr), &info->tup.saddr_l, &info->tup.saddr_h);
        read_ipv6_skb(skb, info->data_off + offsetof(struct ipv6hdr, daddr), &info->tup.daddr_l, &info->tup.daddr_h);
        info->data_off += sizeof(struct ipv6hdr);
        break;
    default:
        return 0;
    }

    switch (l4_proto) {
    case IPPROTO_UDP:
        info->tup.metadata |= CONN_TYPE_UDP;
        info->tup.sport = load_half(skb, info->data_off + offsetof(struct udphdr, source));
        info->tup.dport = load_half(skb, info->data_off + offsetof(struct udphdr, dest));
        info->data_off += sizeof(struct udphdr);
        break;
    case IPPROTO_TCP:
        info->tup.metadata |= CONN_TYPE_TCP;
        info->tup.sport = load_half(skb, info->data_off + offsetof(struct tcphdr, source));
        info->tup.dport = load_half(skb, info->data_off + offsetof(struct tcphdr, dest));

        info->tcp_flags = load_byte(skb, info->data_off + TCP_FLAGS_OFFSET);
        // TODO: Improve readability and explain the bit twiddling below
        info->data_off += ((load_byte(skb, info->data_off + offsetof(struct tcphdr, ack_seq) + 4) & 0xF0) >> 4) * 4;
        break;
    default:
        return 0;
    }

    return 1;
}

static __always_inline char*  get_method(char* p) {
    char *method = "UNKNOWN";

    if ((p[0] == 'G') && (p[1] == 'E') && (p[2] == 'T')) {
        method = "GET";
    } else if ((p[0] == 'P') && (p[1] == 'O') && (p[2] == 'S') && (p[3] == 'T')) {
        method = "POST";
    } else if ((p[0] == 'P') && (p[1] == 'U') && (p[2] == 'T')) {
        method = "PUT";
    } else if ((p[0] == 'D') && (p[1] == 'E') && (p[2] == 'L') && (p[3] == 'E') && (p[4] == 'T') && (p[5] == 'E')) {
        method = "DELETE";
    } else if ((p[0] == 'H') && (p[1] == 'E') && (p[2] == 'A') && (p[3] == 'D')) {
        method = "HEAD";
    } else if ((p[0] == 'O') && (p[1] == 'P') && (p[2] == 'T') && (p[3] == 'I') && (p[4] == 'O') && (p[5] == 'N') && (p[6] == 'S')) {
        method = "OPTIONS";
    } else if ((p[0] == 'P') && (p[1] == 'A') && (p[2] == 'T') && (p[3] == 'C') && (p[4] == 'H')) {
        method = "PATCH";
    } else if ((p[0] == 'C') && (p[1] == 'O') && (p[2] == 'N') && (p[3] == 'N') && (p[4] == 'E') && (p[5] == 'C') && (p[6] == 'T')) {
        method = "CONNECT";
    } else if ((p[0] == 'T') && (p[1] == 'R') && (p[2] == 'A') && (p[3] == 'C') && (p[4] == 'E')) {
        method = "TRACE";
    }

    return method;
}

static __always_inline bool skb_revalidate_data(struct __sk_buff *skb,
                                                uint8_t **head, uint8_t **tail,
                                                const __u32 offset) {
    if (*head + offset > *tail) {
        if (bpf_skb_pull_data(skb, offset) < 0)
            return false;

        *head = (uint8_t *)(long)skb->data;
        *tail = (uint8_t *)(long)skb->data_end;

        if (*head + offset > *tail)
            return false;
    }

    return true;
}
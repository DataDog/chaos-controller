// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

// +build ignore
#include "injection.bpf.h"

#define MAX_PATH_LEN 100
#define MAX_METHOD_LEN 8

// Define the eBPF map to store the flags
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 2);
    __type(key, int);
    __type(value, char[MAX_PATH_LEN]);
} flags_map SEC(".maps");


static __always_inline bool  validate_path(char* path) {
    // Get the expected path
    __u32 expected_path_key = 0;
    char expected_path[MAX_PATH_LEN];
    bpf_probe_read_kernel_str(&expected_path, sizeof(expected_path), bpf_map_lookup_elem(&flags_map, &expected_path_key));

    // Consider the path is not valid if the expected path is not defined.
    if (expected_path[0] == NULL)
        return false;

    // Get the path of the response.
    char request_path[MAX_PATH_LEN];
    bpf_probe_read_kernel_str(&request_path, sizeof(request_path), path);

    // Check if the prefix match the path.
    return has_prefix(request_path, expected_path);
}

static __always_inline bool  validate_method(char* method) {
     // Get the expected method.
     __u32 expected_method_key = 1;
     char expected_method[MAX_METHOD_LEN];
     bpf_probe_read_kernel_str(&expected_method, sizeof(expected_method), bpf_map_lookup_elem(&flags_map, &expected_method_key));

     // Don't apply the tc rule if the method is not defined.
     if (expected_method[0] == NULL)
         return false;

     // If the method is ALL apply the next tc rule.
     if ((expected_method[0] == 'A') && (expected_method[1] == 'L') && (expected_method[2] == 'L'))
         return true;

     // Get the method of the request to compare it with the expected method.
     char request_method[MAX_METHOD_LEN];
     bpf_probe_read_kernel_str(&request_method, sizeof(request_method), method);

     // Check if the prefix match the method.
     return has_prefix(request_method, expected_method);
}

SEC("classifier")
int cls_entry(struct __sk_buff *skb)
{
    skb_info_t skb_info;

    if (!read_conn_tuple_skb(skb, &skb_info))
        return 0;

    char p[HTTP_BUFFER_SIZE];
    http_packet_t packet_type;

    if (skb->len - skb_info.data_off < HTTP_BUFFER_SIZE) {
        printt("http buffer reach the limit");
        return 0;
    }

    for (int i = 0; i < HTTP_BUFFER_SIZE; i++) {
        p[i] = load_byte(skb, skb_info.data_off + i);
    }

    char *method = get_method(p);
    if (method == "UNKNOWN") {
       printt("not an http request");
       return 0;
    }

    int i;
    char path[MAX_PATH_LEN];
    int path_length = 0;

    // Extract the path from the response
    for (i = 0; i < HTTP_BUFFER_SIZE; i++) {
        if (p[i] == ' ') {
            i++;
            // Find the end of the path
            while (i < HTTP_BUFFER_SIZE && p[i] != ' ' && path_length < MAX_PATH_LEN - 1) {
                path[path_length] = p[i];
                path_length++;
                i++;
            }

            // Null-terminate the path
            path[path_length] = '\0';
            break;
        }
    }

    printt("PATH: %s", path);

    if (!validate_path(path)) {
        return 0;
    }

    if (validate_method(method)) {
        printt("DISRUPTED PATH %s!", path);
        return -1;
    }

    // Don't apply the next tc rule.
    return 0;
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

// +build ignore
#include "injection.bpf.h"

#define MAX_PATH_LEN 90
#define MAX_METHOD_LEN 8
#define MAX_PATHS_ENTRIES 20
#define MAX_METHODS_ENTRIES 9

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_METHODS_ENTRIES);
    __type(key, int);
    __type(value, char[MAX_METHOD_LEN]);
} filter_methods SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_PATHS_ENTRIES);
    __type(key, int);
    __type(value, char[MAX_PATH_LEN]);
} filter_paths SEC(".maps");


static __always_inline bool  validate_path(char* path) {
     for (__u32 i = 0; i < MAX_PATHS_ENTRIES; i++) {
        // Get the expected path
        char expected_path[MAX_PATH_LEN];
        __u32 key = i; // Used to avoid runtime infinity loop error.
        int err = bpf_probe_read_kernel(&expected_path, sizeof(expected_path), bpf_map_lookup_elem(&filter_paths, &key));
        if (err != 0) {
            printt("could not get the path. Key: %d. Map: filter_paths", i);
            break;
        }

        char request_path[MAX_PATH_LEN];
        bpf_probe_read_kernel_str(&request_path, sizeof(request_path), path);

        // Check if the prefix match the method.
        for (int i = 0; i < MAX_PATH_LEN; ++i) {
            // Break the loop if the prefix is completed
            if (expected_path[i] == NULL)
                return true;

            // If the prefix does not match the str return false
            if (expected_path[i] != request_path[i])
                break;
        }
     }

    return false;
}

static __always_inline bool  validate_method(char* method) {
     for (__u32 i = 0; i < MAX_METHODS_ENTRIES; i++) {
        // Get the expected method
        char expected_method[MAX_METHOD_LEN];
        __u32 key = i; // Used to avoid runtime infinity loop error
        int err = bpf_probe_read_kernel(&expected_method, sizeof(expected_method), bpf_map_lookup_elem(&filter_methods, &key));
        if (err != 0) {
            printt("could not get the method. Key: %d. Map: filter_methods", i);
            break;
        }

        // Check if the prefix match the method.
        for (int i = 0; i < MAX_METHOD_LEN; ++i) {
            // Break the loop if the prefix is completed
            if (expected_method[i] == NULL)
                return true;

            // If the prefix does not match the str return false
            if (expected_method[i] != method[i])
                break;
        }
     }

    return false;
}

SEC("classifier_methods")
int cls_classifier_methods(struct __sk_buff *skb)
{
    skb_info_t skb_info;

    if (!read_conn_tuple_skb(skb, &skb_info))
        return 0;

    if (skb->len - skb_info.data_off < DEFAULT_HTTP_BUFFER_SIZE) {
        printt("http buffer reach the limit");
        return 0;
    }

    char p[DEFAULT_HTTP_BUFFER_SIZE];

    for (int i = 0; i < DEFAULT_HTTP_BUFFER_SIZE; i++) {
        p[i] = load_byte(skb, skb_info.data_off + i);
    }

    char *method = get_method(p);
    if (method == "UNKNOWN") {
       printt("not an http request");
       return 0;
    }

    if (validate_method(method)) {
        printt("MATCH METHOD %s!", method);
        return -1;
    }

    // Don't apply the next tc rule.
    return 0;
}

SEC("classifier_paths")
int cls_classifier_paths(struct __sk_buff *skb)
{
    skb_info_t skb_info;

    if (!read_conn_tuple_skb(skb, &skb_info))
        return 0;

    if (skb->len - skb_info.data_off < LARGE_HTTP_BUFFER_SIZE) {
        printt("http buffer reach the limit");
        return 0;
    }

    char p[LARGE_HTTP_BUFFER_SIZE];

    for (int i = 0; i < LARGE_HTTP_BUFFER_SIZE; i++) {
        p[i] = load_byte(skb, skb_info.data_off + i);
    }

    char path[MAX_PATH_LEN];
    int path_length = 0;

    // Extract the path from the response
    for (int i = 0; i < LARGE_HTTP_BUFFER_SIZE; i++) {
        if (p[i] == ' ') {
            i++;
            // Find the end of the path
            while (i < LARGE_HTTP_BUFFER_SIZE && p[i] != ' ' && path_length < MAX_PATH_LEN - 1) {
                path[path_length] = p[i];
                path_length++;
                i++;
            }

            // Null-terminate the path
            path[path_length] = '\0';
            break;
        }
    }

    if (validate_path(path)) {
        printt("MATCH PATH %s!", path);
        return -1;
    }

    // Don't apply the next tc rule.
    return 0;
}

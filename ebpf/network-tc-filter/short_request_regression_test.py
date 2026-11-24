# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2026 Datadog, Inc.

"""Source-level regressions executing the actual production C function bodies.

Run from the chaos-controller root using Python 3 and a C compiler:
    python3 ebpf/network-tc-filter/short_request_regression_test.py

Set CC to choose the compiler. Kernel helpers are substituted only at their
boundary; classifiers and disk prefix comparison are extracted verbatim from
production. This does not qualify BPF verifier compatibility or attachment.
"""

import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

ROOT = Path.cwd()


def function(source, name):
    match = re.search(r"\b" + re.escape(name) + r"\s*\([^;]*?\)\s*\{", source)
    if match is None:
        raise AssertionError("production function not found: " + name)
    start = source.rfind("\n", 0, match.start()) + 1
    opening = source.index("{", match.start())
    depth = 1
    end = opening + 1
    while depth:
        depth += (source[end] == "{") - (source[end] == "}")
        end += 1
    return source[start:end]


def execute_c(source):
    with tempfile.TemporaryDirectory() as directory:
        path = Path(directory)
        (path / "regression.c").write_text(source)
        subprocess.run([os.environ.get("CC", "cc"), "-std=gnu11", "-O2",
                        str(path / "regression.c"), "-o", str(path / "regression")], check=True)
        subprocess.run([str(path / "regression")], check=True)


COMMON = r'''
#include <assert.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#define printt(...) ((void)0)
#ifndef __always_inline
#define __always_inline inline __attribute__((always_inline))
#endif
typedef uint32_t __u32;
'''


def network_program(main):
    source = (ROOT / "ebpf/network-tc-filter/injection.bpf.c").read_text()
    http = (ROOT / "ebpf/includes/http.h").read_text()
    constants = "\n".join(re.findall(r"^#define (?:MAX_\w+|(?:DEFAULT|LARGE)_HTTP_BUFFER_SIZE) \d+$",
                                    source + "\n" + http, re.MULTILINE))
    helpers = r'''
int filter_paths, filter_methods;
char paths[MAX_PATHS_ENTRIES][MAX_PATH_LEN];
char methods[MAX_METHODS_ENTRIES][MAX_METHOD_LEN];
void *bpf_map_lookup_elem(void *map, const __u32 *key) {
    return map == &filter_paths ? (void *)paths[*key] : (void *)methods[*key];
}
int bpf_probe_read_kernel(void *dest, unsigned size, const void *src) {
    memcpy(dest, src, size);
    return 0;
}
int bpf_probe_read_kernel_str(void *dest, unsigned size, const void *src) {
    unsigned length = strnlen(src, size - 1);
    memcpy(dest, src, length);
    ((char *)dest)[length] = 0;
    return length + 1;
}
struct __sk_buff { unsigned len; const unsigned char *payload; };
typedef struct { unsigned data_off; } skb_info_t;
int read_conn_tuple_skb(struct __sk_buff *skb, skb_info_t *info) {
    (void)skb;
    info->data_off = 0;
    return 1;
}
unsigned char load_byte(struct __sk_buff *skb, unsigned offset) {
    assert(offset < skb->len);
    return skb->payload[offset];
}
'''
    functions = "\n".join(function(text, name) for text, name in [
        (http, "get_method"), (source, "validate_method"), (source, "validate_path"),
        (source, "cls_classifier_methods"), (source, "cls_classifier_paths")])
    return COMMON + constants + "\n" + helpers + functions + main


class EBPFBoundaryRegression(unittest.TestCase):
    def test_short_complete_http_requests(self):
        execute_c(network_program(r'''
int main(void) {
    strcpy(methods[0], "GET");
    strcpy(paths[0], "/");
    const unsigned char short_request[] = "GET / HTTP/1.0\r\n\r\n";
    struct __sk_buff short_skb = {sizeof(short_request) - 1, short_request};
    assert(cls_classifier_methods(&short_skb) == -1);
    assert(cls_classifier_paths(&short_skb) == -1);
    const unsigned char request[] = "GET /health HTTP/1.1\r\nHost: test\r\n\r\n";
    strcpy(paths[0], "/health");
    struct __sk_buff skb = {sizeof(request) - 1, request};
    assert(cls_classifier_methods(&skb) == -1);
    assert(cls_classifier_paths(&skb) == -1);
    strcpy(paths[0], "/other");
    assert(cls_classifier_paths(&skb) == 0);
    return 0;
}
'''))

if __name__ == "__main__":
    unittest.main()

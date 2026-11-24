# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2026 Datadog, Inc.

"""Source-level regressions executing the actual production C function bodies.

Run from the chaos-controller root using Python 3 and a C compiler:
    python3 ebpf/disk-failure/path_length_regression_test.py

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
    def test_disk_sixty_one_and_sixty_two_byte_prefixes(self):
        source = (ROOT / "ebpf/disk-failure/injection.bpf.c").read_text()
        capacity = int(re.search(r"filter_path\[(\d+)\]", source).group(1))
        start = source.index("    // Allow only file with the desired prefix.")
        end = source.index("#endif", start)
        prefix_check = source[start:end]
        helpers = r'''
struct pt_regs { void *first; void *second; };
#define PT_REGS_PARM1(ctx) ((ctx)->first)
#define PT_REGS_PARM2_CORE(ctx) ((ctx)->second)
int bpf_probe_read(void *dest, unsigned size, const void *src) {
    if (src == filter_path && size > sizeof(filter_path)) {
        memset(dest, 0, size);
        memcpy(dest, src, sizeof(filter_path));
    } else {
        memcpy(dest, src, size);
    }
    return 0;
}
'''
        main = r'''
int main(void) {
    char expected[128] = {0}, candidate[128] = {0};
    for (unsigned length = 61; length <= 62; length++) {
        assert(sizeof(filter_path) >= length);
        memset(expected, 'a', length);
        expected[0] = '/';
        expected[length] = 0;
        memset(filter_path, 0, sizeof(filter_path));
        memcpy(filter_path, expected, length);
        memcpy(candidate, expected, sizeof(candidate));
        struct pt_regs syscall_regs = {0, candidate};
        struct pt_regs probe_regs = {&syscall_regs, 0};
        assert(matches(&probe_regs) == 1);
        candidate[length - 1] = 'b';
        assert(matches(&probe_regs) == 0);
    }
    return 0;
}
'''
        execute_c(COMMON + f"char filter_path[{capacity}];\n" + helpers +
                  "int matches(struct pt_regs *ctx) {\n" + prefix_check +
                  "return 1;\n}\n" + main)


if __name__ == "__main__":
    unittest.main()

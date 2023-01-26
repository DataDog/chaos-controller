/* In Linux 5.4 asm_inline was introduced, but it's not supported by clang.
 * Redefine it to just asm to enable successful compilation.
 * see https://github.com/iovisor/bcc/commit/2d1497cde1cc9835f759a707b42dea83bee378b8 for more details
 */
#if defined(__x86_64__) || defined(__TARGET_ARCH_x86)
#include "amd64/vmlinux.h"
#endif
#if defined(__aarch64__) || defined(__TARGET_ARCH_arm64)
#include "aarch64/vmlinux.h"
#endif
//#if defined(__TARGET_ARCH_x86)
//#include "amd64/vmlinux.h"
//#elif defined(__TARGET_ARCH_arm64)
//#include "aarch64/vmlinux.h"
//#endif
#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

/* TODO!! This is too generic for this example, where can we pull it from? 
*/
//#define BPF_MAP(_name, _type, _key_type, _value_type, _max_entries) \
//    struct {						            \
//        __uint(_type, BPF_MAP_TYPE_ARRAY);      		    \
//        __uint(_max_entries, 4);                                    \
//        __type(_key_type, int);					    \
//        __type(_value_type, struct ipv_counts);  		    \
//    } _name SEC(".maps");
//
//#define BPF_HASH(_name, _key_type, _value_type) \
//    BPF_MAP(_name, BPF_MAP_TYPE_HASH, _key_type, _value_type, 10240);
//
//#define BPF_PERF_OUTPUT(_name) \
//    BPF_MAP(_name, BPF_MAP_TYPE_PERF_EVENT_ARRAY, int, __u32, 1024);

char LICENSE[] SEC("license") = "Dual BSD/GPL";


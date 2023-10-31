// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ebpf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// ConfigInformer is an interface that defines a set of methods for retrieving and validating
// system configuration information related to kernel features and maps.
type ConfigInformer interface {
	// GetKernelFeatures retrieves kernel features and returns them as a 'Features' struct,
	// along with any encountered error.
	GetKernelFeatures() (Features, error)

	// ValidateRequiredSystemConfig validates the required system configuration.
	// It returns an error if the validation fails.
	ValidateRequiredSystemConfig() error

	// GetRequiredSystemConfig retrieves the required system configuration parameters as
	// a 'KernelParams' struct.
	GetRequiredSystemConfig() KernelParams

	// GetMapTypes retrieves information about available map types and returns them as a 'MapTypes' struct.
	GetMapTypes() MapTypes

	// IsKernelConfigAvailable checks if the kernel configuration is available
	IsKernelConfigAvailable() bool
}

type configInformer struct {
	bptfool  Executor
	features Features
	fs       fs.StatFS
	uname    func() (unix.Utsname, error)
}

// KernelOption holds information about kernel parameters to probe.
type KernelOption struct {
	Description string
	Enabled     bool
}

// KernelParam is a type based on string which represents CONFIG_* kernel
// parameters which usually have values "y", "n" or "m".
type KernelParam string

// KernelParams is a map that associates KernelParam keys with their corresponding
// KernelOption values. It is used to store and manage kernel parameters and their options.
type KernelParams map[KernelParam]KernelOption

// SystemConfig contains kernel configuration and sysctl parameters related to
// BPF functionality.
type SystemConfig struct {
	UnprivilegedBpfDisabled      int         `json:"unprivileged_bpf_disabled"`
	BpfJitEnable                 int         `json:"bpf_jit_enable"`
	BpfJitHarden                 int         `json:"bpf_jit_harden"`
	BpfJitKallsyms               int         `json:"bpf_jit_kallsyms"`
	BpfJitLimit                  int         `json:"bpf_jit_limit"`
	ConfigBpf                    KernelParam `json:"CONFIG_BPF"`
	ConfigBpfSyscall             KernelParam `json:"CONFIG_BPF_SYSCALL"`
	ConfigHaveEbpfJit            KernelParam `json:"CONFIG_HAVE_EBPF_JIT"`
	ConfigBpfJit                 KernelParam `json:"CONFIG_BPF_JIT"`
	ConfigBpfJitAlwaysOn         KernelParam `json:"CONFIG_BPF_JIT_ALWAYS_ON"`
	ConfigCgroups                KernelParam `json:"CONFIG_CGROUPS"`
	ConfigCgroupBpf              KernelParam `json:"CONFIG_CGROUP_BPF"`
	ConfigCgroupNetClassID       KernelParam `json:"CONFIG_CGROUP_NET_CLASSID"`
	ConfigSockCgroupData         KernelParam `json:"CONFIG_SOCK_CGROUP_DATA"`
	ConfigBpfEvents              KernelParam `json:"CONFIG_BPF_EVENTS"`
	ConfigKprobeEvents           KernelParam `json:"CONFIG_KPROBE_EVENTS"`
	ConfigUprobeEvents           KernelParam `json:"CONFIG_UPROBE_EVENTS"`
	ConfigTracing                KernelParam `json:"CONFIG_TRACING"`
	ConfigFtraceSyscalls         KernelParam `json:"CONFIG_FTRACE_SYSCALLS"`
	ConfigFunctionErrorInjection KernelParam `json:"CONFIG_FUNCTION_ERROR_INJECTION"`
	ConfigBpfKprobeOverride      KernelParam `json:"CONFIG_BPF_KPROBE_OVERRIDE"`
	ConfigNet                    KernelParam `json:"CONFIG_NET"`
	ConfigXdpSockets             KernelParam `json:"CONFIG_XDP_SOCKETS"`
	ConfigLwtunnelBpf            KernelParam `json:"CONFIG_LWTUNNEL_BPF"`
	ConfigNetActBpf              KernelParam `json:"CONFIG_NET_ACT_BPF"`
	ConfigNetClsBpf              KernelParam `json:"CONFIG_NET_CLS_BPF"`
	ConfigNetClsAct              KernelParam `json:"CONFIG_NET_CLS_ACT"`
	ConfigNetSchIngress          KernelParam `json:"CONFIG_NET_SCH_INGRESS"`
	ConfigXfrm                   KernelParam `json:"CONFIG_XFRM"`
	ConfigIPRouteClassID         KernelParam `json:"CONFIG_IP_ROUTE_CLASSID"`
	ConfigIPv6Seg6Bpf            KernelParam `json:"CONFIG_IPV6_SEG6_BPF"`
	ConfigBpfLircMode2           KernelParam `json:"CONFIG_BPF_LIRC_MODE2"`
	ConfigBpfStreamParser        KernelParam `json:"CONFIG_BPF_STREAM_PARSER"`
	ConfigNetfilterXtMatchBpf    KernelParam `json:"CONFIG_NETFILTER_XT_MATCH_BPF"`
	ConfigBpfilter               KernelParam `json:"CONFIG_BPFILTER"`
	ConfigBpfilterUmh            KernelParam `json:"CONFIG_BPFILTER_UMH"`
	ConfigTestBpf                KernelParam `json:"CONFIG_TEST_BPF"`
	ConfigKernelHz               KernelParam `json:"CONFIG_HZ"`
}

// MapTypes contains bools indicating which types of BPF maps the currently
// running kernel supports.
type MapTypes struct {
	HaveHashMapType                bool `json:"have_hash_map_type"`
	HaveArrayMapType               bool `json:"have_array_map_type"`
	HaveProgArrayMapType           bool `json:"have_prog_array_map_type"`
	HavePerfEventArrayMapType      bool `json:"have_perf_event_array_map_type"`
	HavePercpuHashMapType          bool `json:"have_percpu_hash_map_type"`
	HavePercpuArrayMapType         bool `json:"have_percpu_array_map_type"`
	HaveStackTraceMapType          bool `json:"have_stack_trace_map_type"`
	HaveCgroupArrayMapType         bool `json:"have_cgroup_array_map_type"`
	HaveLruHashMapType             bool `json:"have_lru_hash_map_type"`
	HaveLruPercpuHashMapType       bool `json:"have_lru_percpu_hash_map_type"`
	HaveLpmTrieMapType             bool `json:"have_lpm_trie_map_type"`
	HaveArrayOfMapsMapType         bool `json:"have_array_of_maps_map_type"`
	HaveHashOfMapsMapType          bool `json:"have_hash_of_maps_map_type"`
	HaveDevmapMapType              bool `json:"have_devmap_map_type"`
	HaveSockmapMapType             bool `json:"have_sockmap_map_type"`
	HaveCpumapMapType              bool `json:"have_cpumap_map_type"`
	HaveXskmapMapType              bool `json:"have_xskmap_map_type"`
	HaveSockhashMapType            bool `json:"have_sockhash_map_type"`
	HaveCgroupStorageMapType       bool `json:"have_cgroup_storage_map_type"`
	HaveReuseportSockarrayMapType  bool `json:"have_reuseport_sockarray_map_type"`
	HavePercpuCgroupStorageMapType bool `json:"have_percpu_cgroup_storage_map_type"`
	HaveQueueMapType               bool `json:"have_queue_map_type"`
	HaveStackMapType               bool `json:"have_stack_map_type"`
}

// Features is a struct that represents a collection of features.
type Features struct {
	// SystemConfig represents a system's configuration information.
	SystemConfig `json:"system_config"`

	// MapTypes represents information about available map types.
	MapTypes `json:"map_types"`
}

const BpftoolBinary = "bpftool"

// osFS implements fs.StatFS using the local disk.
type osFS struct {
	fs.StatFS
}

func (o osFS) Open(name string) (fs.File, error) { return os.Open(name) }

func (o osFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

// Enabled checks whether the kernel parameter is enabled.
func (kp KernelParam) Enabled() bool {
	return kp == "y"
}

// NewConfigInformer creates and returns a new instance of a ConfigInformer, which is an interface
// that provides methods to retrieve system configuration information.
//
// Parameters:
//   - log: Required - A SugaredLogger from the zap library used for logging.
//   - dryRun: Required - A boolean flag indicating whether the operations should be executed as a dry run.
//   - executor: An Executor interface that defines how system commands are executed. If nil, a default
//     BpftoolExecutor is created.
//   - FSMock: A FileSystem interface used for file operations. If nil, the default osFS implementation is used.
//   - unameFuncMock: A function that provides Utsname information. If nil, the system's Uname function is used.
//
// Returns:
//   - ConfigInformer: An interface for retrieving system configuration information.
//   - error: An error, if any, encountered during the creation and initialization of the ConfigInformer.
func NewConfigInformer(log *zap.SugaredLogger, dryRun bool, executor Executor, fsMock fs.StatFS, unameFuncMock func() (unix.Utsname, error)) (ConfigInformer, error) {
	if executor == nil {
		executor = NewBpftoolExecutor(log, dryRun)
	}

	configInf := &configInformer{
		bptfool: executor,
	}

	if fsMock == nil {
		configInf.fs = osFS{}
	} else {
		configInf.fs = fsMock
	}

	if unameFuncMock == nil {
		configInf.uname = func() (unix.Utsname, error) {
			info := unix.Utsname{}
			err := unix.Uname(&info)

			return info, err
		}
	} else {
		configInf.uname = unameFuncMock
	}

	features, err := configInf.GetKernelFeatures()
	if err != nil {
		return configInf, err
	}

	configInf.features = features

	return configInf, nil
}

// GetKernelFeatures retrieves kernel features using the bpftool utility.
// It returns a 'Features' struct representing the features and any encountered error.
func (v configInformer) GetKernelFeatures() (features Features, err error) {
	// Run the bpftool utility to fetch features in JSON format.
	_, stdout, err := v.bptfool.Run([]string{"-j", "feature", "probe"})
	if err != nil {
		return features, fmt.Errorf("could not run bpftool: %w", err)
	}

	err = json.Unmarshal([]byte(stdout), &features)

	return
}

// ValidateRequiredSystemConfig checks if the required system configuration is satisfied.
// It verifies the presence of the kernel configuration file and validates that specific
// kernel parameters are enabled as required.
//
// Returns:
//   - error: An error indicating any issues with the required system configuration.
func (v configInformer) ValidateRequiredSystemConfig() error {
	var multiErr error

	if !v.IsKernelConfigAvailable() {
		return fmt.Errorf("kernel config file not found")
	}

	requiredParams := v.GetRequiredSystemConfig()
	for param, kernelOption := range requiredParams {
		if !kernelOption.Enabled {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s kernel parameter is required (needed for: %s)", param, kernelOption.Description))
		}
	}

	return multiErr
}

// GetRequiredSystemConfig retrieves the required system configuration parameters and their options.
//
// Returns:
//   - KernelParams: A map of required kernel parameters and their associated KernelOption settings.
func (v configInformer) GetRequiredSystemConfig() KernelParams {
	config := v.features.SystemConfig
	coreInfraDescription := "Essential eBPF infrastructure"

	return KernelParams{
		"CONFIG_BPF": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigBpf.Enabled(),
		},
		"CONFIG_BPF_SYSCALL": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigBpfSyscall.Enabled(),
		},
		"CONFIG_BPF_JIT": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigBpfJit.Enabled(),
		},
		"CONFIG_HAVE_EBPF_JIT": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigHaveEbpfJit.Enabled(),
		},
		"CONFIG_BPF_KPROBE_OVERRIDE": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigBpfKprobeOverride.Enabled(),
		},
		"CONFIG_NET_CLS_ACT": KernelOption{
			Description: coreInfraDescription,
			Enabled:     config.ConfigNetClsAct.Enabled(),
		},
	}
}

// GetMapTypes retrieves information about available map types from the system configuration features.
//
// Returns:
//   - MapTypes: A MapTypes struct containing information about available map types.
func (v configInformer) GetMapTypes() MapTypes {
	return v.features.MapTypes
}

// IsKernelConfigAvailable checks if the kernel configuration is available on the system.
//
// Returns:
//   - bool: true if the kernel configuration is available, false otherwise.
func (v configInformer) IsKernelConfigAvailable() bool {
	info, err := v.uname()
	if err != nil {
		return false
	}

	release := strings.TrimSpace(string(bytes.Trim(info.Release[:], "\x00")))
	// Any error checking these files will return Kernel config not found error
	if _, err := v.fs.Stat(fmt.Sprintf("/boot/config-%s", release)); err != nil {
		if _, err = v.fs.Stat("/proc/config.gz"); err != nil {
			return false
		}
	}

	return true
}

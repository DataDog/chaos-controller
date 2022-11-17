// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package injector

import (
	"os"
	"os/exec"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
)

const (
	ebpfaultConfig = `
{
	"syscall_name": "openat",

	"error_list": [
		{
			"exit_code": "-ENOENT",
			"probability": 50
		}
	]
}
`
)

type DiskFailureInjector struct {
	spec   v1beta1.DiskFailureSpec
	config DiskFailureInjectorConfig
}

// DiskFailureInjectorConfig is the disk pressure injector config
type DiskFailureInjectorConfig struct {
	Config
	EBPFaultProcess *os.Process
}

// NewDiskFailureInjector creates a disk pressure injector with the given config
func NewDiskFailureInjector(spec v1beta1.DiskFailureSpec, config DiskFailureInjectorConfig) (Injector, error) {
	return &DiskFailureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *DiskFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskFailure
}

func (i *DiskFailureInjector) Inject() (err error) {
	// Write the config.json
	var f *os.File
	f, err = os.Create("config.json")
	if err != nil {
		return
	}
	defer f.Close()

	if _, err = f.WriteString(ebpfaultConfig); err != nil {
		return
	}

	// Start the execution of ebpfault
	cmd := exec.Command("ebpfault", []string{"--config", "config.json", "-p", strconv.Itoa(int(i.config.Config.TargetContainer.PID()))}...)

	if err = cmd.Run(); err != nil {
		return
	}

	// Store the PID of our ebpfault execution
	i.config.EBPFaultProcess = cmd.Process

	return nil
}

func (i *DiskFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *DiskFailureInjector) Clean() error {
	// Kill the process
	if i.config.EBPFaultProcess != nil {
		return i.config.EBPFaultProcess.Kill()
	}

	return nil
}

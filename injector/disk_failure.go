// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
)

type DiskFailureInjector struct {
	spec   v1beta1.DiskFailureSpec
	config DiskFailureInjectorConfig
}

// DiskFailureInjectorConfig is the disk pressure injector config
type DiskFailureInjectorConfig struct {
	Config
	CmdFactory        command.Factory
	ProcessManager    process.Manager
	BPFConfigInformer ebpf.ConfigInformer
}

const EBPFDiskFailureCmd = "bpf-disk-failure"

// NewDiskFailureInjector creates a disk failure injector with the given config
func NewDiskFailureInjector(spec v1beta1.DiskFailureSpec, config DiskFailureInjectorConfig) (Injector, error) {
	if config.CmdFactory == nil {
		config.CmdFactory = command.NewFactory(config.Disruption.DryRun)
	}

	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.Disruption.DryRun)
	}

	if config.BPFConfigInformer == nil {
		var err error
		config.BPFConfigInformer, err = ebpf.NewConfigInformer(config.Log, config.Disruption.DryRun, nil, nil, nil)

		if err != nil {
			return nil, fmt.Errorf("could not create an instance of eBPF config informer for the disk failure disruption: %w", err)
		}
	}

	return &DiskFailureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *DiskFailureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *DiskFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskFailure
}

func (i *DiskFailureInjector) Inject() error {
	if err := i.config.BPFConfigInformer.ValidateRequiredSystemConfig(); err != nil {
		return fmt.Errorf("the disk failure needs a kernel supporting eBPF programs: %w", err)
	}

	if !i.config.BPFConfigInformer.GetMapTypes().HavePerfEventArrayMapType {
		return fmt.Errorf("the disk failure needs the perf event array map type, but the current kernel does not support this type of map")
	}

	pid := 0
	if i.config.Disruption.Level == types.DisruptionLevelPod {
		pid = int(i.config.Config.TargetContainer.PID())
	}

	exitCode := 0

	if i.spec.OpenatSyscall != nil {
		exitCode = i.spec.OpenatSyscall.GetExitCodeInt()
	}

	for _, path := range i.spec.Paths {
		args := []string{"-process", strconv.Itoa(pid)}

		if path != "" {
			args = append(args, "-path", path)
		}

		if exitCode != 0 {
			args = append(args, "-exit-code", fmt.Sprintf("%v", exitCode))
		}

		args = append(args, "-probability", strings.TrimSuffix(i.spec.Probability, "%"))

		cmd := i.config.CmdFactory.NewCmd(context.Background(), EBPFDiskFailureCmd, args)

		bgCmd := command.NewBackgroundCmd(cmd, i.config.Log, i.config.ProcessManager)
		if err := bgCmd.Start(); err != nil {
			return fmt.Errorf("unable to run the eBPF disk failure: %w", err)
		}
	}

	return nil
}

func (i *DiskFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *DiskFailureInjector) Clean() error {
	return nil
}

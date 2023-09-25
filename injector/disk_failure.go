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
	CmdFactory     command.Factory
	ProcessManager process.Manager
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

	return &DiskFailureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *DiskFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskFailure
}

func (i *DiskFailureInjector) Inject() error {
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
			launchDNSServerErr = fmt.Errorf("unable to run eBPF disk failure: %w", err)
			return nil
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

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

type DiskFailureInjector struct {
	spec   v1beta1.DiskFailureSpec
	config DiskFailureInjectorConfig
}

// DiskFailureInjectorConfig is the disk pressure injector config
type DiskFailureInjectorConfig struct {
	Config
	Cmd BPFDiskFailureCommand
}

type BPFDiskFailureCommand interface {
	Run(pid int, path string, exitCode int) error
}

type bPFDiskFailureCommand struct {
	log *zap.SugaredLogger
}

const EBPFDiskFailureCmd = "bpf-disk-failure"

func (d bPFDiskFailureCommand) Run(pid int, path string, exitCode int) error {
	commandPath := []string{"-p", strconv.Itoa(pid)}

	if path != "" {
		commandPath = append(commandPath, "-f", path)
	}

	if exitCode != 0 {
		commandPath = append(commandPath, "-c", fmt.Sprintf("%v", exitCode))
	}

	execCmd := exec.Command(EBPFDiskFailureCmd, commandPath...)

	d.log.Infow(
		"injecting disk failure",
		zap.String("command", EBPFDiskFailureCmd),
		zap.String("args", strings.Join(commandPath, " ")),
	)

	go func() {
		err := execCmd.Run()
		if err != nil {
			d.log.Errorw(
				"error during the disk failure",
				zap.String("command", EBPFDiskFailureCmd),
				zap.String("args", strings.Join(commandPath, " ")),
				zap.String("error", err.Error()),
			)
		}
	}()

	return nil
}

// NewDiskFailureInjector creates a disk failure injector with the given config
func NewDiskFailureInjector(spec v1beta1.DiskFailureSpec, config DiskFailureInjectorConfig) (Injector, error) {
	if config.Cmd == nil {
		config.Cmd = &bPFDiskFailureCommand{
			log: config.Log,
		}
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

	return i.config.Cmd.Run(pid, i.spec.Path, exitCode)
}

func (i *DiskFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *DiskFailureInjector) Clean() error {
	return nil
}

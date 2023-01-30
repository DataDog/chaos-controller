// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type DiskFailureInjector struct {
	spec   v1beta1.DiskFailureSpec
	config DiskFailureInjectorConfig
}

// DiskFailureInjectorConfig is the disk pressure injector config
type DiskFailureInjectorConfig struct {
	Config
	cmd     BPFDiskFailureCommand
	Process *os.Process
}

type BPFDiskFailureCommand interface {
	Run(pid int, path string) error
	GetProcess() *os.Process
}

type bPFDiskFailureCommand struct {
	process *os.Process
	log     *zap.SugaredLogger
}

const EBPFDiskFailureCmd = "bpf-disk-failure"

func (d *bPFDiskFailureCommand) Run(pid int, path string) error {
	commandPath := []string{"-p", strconv.Itoa(pid)}

	if path != "" {
		commandPath = append(commandPath, "-f", path)
	}

	execCmd := exec.Command(EBPFDiskFailureCmd, commandPath...)
	d.process = execCmd.Process

	d.log.Infow(
		"injecting disk failure",
		zap.String("command", EBPFDiskFailureCmd),
		zap.String("args", strings.Join(commandPath, " ")),
	)

	err := execCmd.Run()
	if err != nil {
		d.log.Errorw(
			"error during the disk failure",
			zap.String("command", EBPFDiskFailureCmd),
			zap.String("args", strings.Join(commandPath, " ")),
			zap.String("error", err.Error()),
		)
	}

	return err
}

func (d bPFDiskFailureCommand) GetProcess() *os.Process {
	return d.process
}

// NewDiskFailureInjector creates a disk pressure injector with the given config
func NewDiskFailureInjector(spec v1beta1.DiskFailureSpec, config DiskFailureInjectorConfig) (Injector, error) {
	if config.cmd == nil {
		config.cmd = &bPFDiskFailureCommand{
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

func (i *DiskFailureInjector) Inject() (err error) {
	// Start the execution of bpf-disk-failure
	pid := 0
	if i.config.Level == types.DisruptionLevelPod {
		pid = int(i.config.Config.TargetContainer.PID())
	}

	go func() {
		err = i.config.cmd.Run(pid, i.spec.Path)
	}()

	// Store the PID for our execution
	i.config.Process = i.config.cmd.GetProcess()

	return
}

func (i *DiskFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *DiskFailureInjector) Clean() error {
	// Kill the process
	if i.config.Process != nil {
		return i.config.Process.Kill()
	}

	return nil
}

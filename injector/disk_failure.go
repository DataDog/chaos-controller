// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package injector

import (
	"os"
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
	Process *os.Process
}

const BpfDiskfailureCmd = "bpf-diskfailure"

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
	// Start the execution of bpf-diskfailure
	pid := int(i.config.Config.TargetContainer.PID())
	commandPath := []string{"-p", strconv.Itoa(pid)}

	if i.spec.Path != "" {
		commandPath = append(commandPath, "-f", i.spec.Path)
	}

	cmd := exec.Command(BpfDiskfailureCmd, commandPath...)

	i.config.Log.Infow(
		"injecting disk failure",
		zap.String("command", BpfDiskfailureCmd),
		zap.String("args", strings.Join(commandPath, " ")),
	)

	go func() {
		if err := cmd.Run(); err != nil {
			i.config.Log.Errorw(
				"error during the disk failure",
				zap.String("command", BpfDiskfailureCmd),
				zap.String("args", strings.Join(commandPath, " ")),
				zap.String("error", err.Error()),
			)
		}
	}()

	// Store the PID for our execution
	i.config.Process = cmd.Process

	return nil
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

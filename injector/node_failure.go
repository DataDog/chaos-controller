// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/types"
)

// nodeFailureInjector describes a node failure injector
type nodeFailureInjector struct {
	spec             v1beta1.NodeFailureSpec
	config           NodeFailureInjectorConfig
	sysrqPath        string
	sysrqTriggerPath string
}

// NodeFailureInjectorConfig contains needed drivers to
// create a NodeFailureInjector
type NodeFailureInjectorConfig struct {
	Config
	FileWriter         FileWriter
	WaitBeforeShutdown time.Duration
}

// NewNodeFailureInjector creates a NodeFailureInjector object with the given config,
// missing fields being initialized with the defaults
func NewNodeFailureInjector(spec v1beta1.NodeFailureSpec, config NodeFailureInjectorConfig) (Injector, error) {
	if config.FileWriter == nil {
		config.FileWriter = standardFileWriter{
			dryRun: config.Disruption.DryRun,
		}
	}

	// retrieve mount path environment variables
	sysrqPath, ok := os.LookupEnv(env.InjectorMountSysrq)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountSysrq)
	}

	sysrqTriggerPath, ok := os.LookupEnv(env.InjectorMountSysrqTrigger)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountSysrqTrigger)
	}

	if config.WaitBeforeShutdown == 0 {
		config.WaitBeforeShutdown = 10 * time.Second
	}

	return &nodeFailureInjector{
		spec:             spec,
		config:           config,
		sysrqPath:        sysrqPath,
		sysrqTriggerPath: sysrqTriggerPath,
	}, nil
}

func (i *nodeFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindNodeFailure
}

// Inject triggers a kernel panic through the sysrq trigger
func (i *nodeFailureInjector) Inject() error {
	var err error

	i.config.Log.Infow("injecting a node failure by triggering a kernel panic",
		"sysrq_path", i.sysrqPath,
		"sysrq_trigger_path", i.sysrqTriggerPath,
	)

	// Ensure sysrq value is set to 1 (to accept the kernel panic trigger)
	if err := i.config.FileWriter.Write(i.sysrqPath, 0o644, "1"); err != nil {
		return fmt.Errorf("error while writing to the sysrq file (%s): %w", i.sysrqPath, err)
	}

	// Trigger kernel panic
	i.config.Log.Infow("the injector will write to the sysrq trigger file in 10s")
	i.config.Log.Infow("from this point, if no fatal log occurs, the injection succeeded and the system will crash")
	_ = i.config.Log.Sync() // If we can't flush the logger, why would logging the error help? so we just ignore

	go func() { // Wait for the logs to be flushed and collected, as the shutdown will be immediate
		time.Sleep(i.config.WaitBeforeShutdown)

		if i.spec.Shutdown {
			err = i.config.FileWriter.Write(i.sysrqTriggerPath, 0o200, "o")
		} else {
			err = i.config.FileWriter.Write(i.sysrqTriggerPath, 0o200, "c")
		}

		if err != nil {
			i.config.Log.Errorf("error while writing to the sysrq trigger file (%s): %v", i.sysrqTriggerPath, err)
		}
	}()

	return nil
}

// Not implemented for node failures
func (i *nodeFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *nodeFailureInjector) Clean() error {
	return nil
}

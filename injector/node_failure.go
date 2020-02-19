// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"go.uber.org/zap"
)

const (
	nodeFailureSysrqPath        = "/mnt/sysrq"
	nodeFailureSysrqTriggerPath = "/mnt/sysrq-trigger"
)

// nodeFailureInjector describes a node failure injector
type nodeFailureInjector struct {
	injector
	spec   v1beta1.NodeFailureSpec
	config NodeFailureInjectorConfig
}

type NodeFailureInjectorConfig struct {
	FileWriter FileWriter
}

func NewNodeFailureInjector(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger) Injector {
	config := NodeFailureInjectorConfig{
		FileWriter: StandardFileWriter{},
	}

	return NewNodeFailureInjectorWithConfig(uid, spec, log, config)
}

func NewNodeFailureInjectorWithConfig(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger, config NodeFailureInjectorConfig) Injector {
	return nodeFailureInjector{
		injector: injector{
			uid: uid,
			log: log,
		},
		spec:   spec,
		config: config,
	}
}

// Inject triggers a kernel panic through the sysrq trigger
func (i nodeFailureInjector) Inject() {
	i.log.Infow("injecting a node failure by triggering a kernel panic",
		"sysrq_path", nodeFailureSysrqPath,
		"sysrq_trigger_path", nodeFailureSysrqTriggerPath,
	)

	// Ensure sysrq value is set to 1 (to accept the kernel panic trigger)
	err := i.config.FileWriter.Write(nodeFailureSysrqPath, 0644, "1")
	if err != nil {
		i.log.Fatalw("error while writing to the sysrq file",
			"error", err,
			"path", nodeFailureSysrqPath,
		)
	}

	// Trigger kernel panic
	i.log.Infow("the injector is about to write to the sysrq trigger file")
	i.log.Infow("from this point, if no fatal log occurs, the injection succeeded and the system will crash")
	if i.spec.Shutdown {
		err = i.config.FileWriter.Write(nodeFailureSysrqTriggerPath, 0200, "o")
	} else {
		err = i.config.FileWriter.Write(nodeFailureSysrqTriggerPath, 0200, "c")
	}
	if err != nil {
		i.log.Fatalw("error while writing to the sysrq trigger file",
			"error", err,
			"path", nodeFailureSysrqTriggerPath,
		)
	}
}

func (i nodeFailureInjector) Clean() {}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
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

// NodeFailureInjectorConfig contains needed drivers to
// create a NodeFailureInjector
type NodeFailureInjectorConfig struct {
	FileWriter FileWriter
}

// NewNodeFailureInjector creates a NodeFailureInjector object with the default drivers
func NewNodeFailureInjector(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	config := NodeFailureInjectorConfig{
		FileWriter: standardFileWriter{},
	}

	return NewNodeFailureInjectorWithConfig(uid, spec, log, ms, config)
}

// NewNodeFailureInjectorWithConfig creates a NodeFailureInjector object with the given config,
// missing fields being initialized with the defaults
func NewNodeFailureInjectorWithConfig(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger, ms metrics.Sink, config NodeFailureInjectorConfig) Injector {
	return nodeFailureInjector{
		injector: injector{
			uid:  uid,
			log:  log,
			ms:   ms,
			kind: types.DisruptionKindNodeFailure,
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

	i.ms.EventWithTags(
		"node failure injected",
		"failing node by triggering a kernel panic",
		[]string{
			"UID:" + i.uid,
		},
	)

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

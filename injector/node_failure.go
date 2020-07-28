// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

const ()

// nodeFailureInjector describes a node failure injector
type nodeFailureInjector struct {
	injector
	spec             v1beta1.NodeFailureSpec
	config           NodeFailureInjectorConfig
	sysrqPath        string
	sysrqTriggerPath string
}

// NodeFailureInjectorConfig contains needed drivers to
// create a NodeFailureInjector
type NodeFailureInjectorConfig struct {
	FileWriter FileWriter
}

// NewNodeFailureInjector creates a NodeFailureInjector object with the default drivers
func NewNodeFailureInjector(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger, ms metrics.Sink) (Injector, error) {
	config := NodeFailureInjectorConfig{
		FileWriter: standardFileWriter{},
	}

	return NewNodeFailureInjectorWithConfig(uid, spec, log, ms, config)
}

// NewNodeFailureInjectorWithConfig creates a NodeFailureInjector object with the given config,
// missing fields being initialized with the defaults
func NewNodeFailureInjectorWithConfig(uid string, spec v1beta1.NodeFailureSpec, log *zap.SugaredLogger, ms metrics.Sink, config NodeFailureInjectorConfig) (Injector, error) {
	// retrieve mount path environment variables
	sysrqPath, ok := os.LookupEnv(env.InjectorMountSysrq)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountSysrq)
	}

	sysrqTriggerPath, ok := os.LookupEnv(env.InjectorMountSysrqTrigger)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountSysrqTrigger)
	}

	return nodeFailureInjector{
		injector: injector{
			uid:  uid,
			log:  log,
			ms:   ms,
			kind: types.DisruptionKindNodeFailure,
		},
		spec:             spec,
		config:           config,
		sysrqPath:        sysrqPath,
		sysrqTriggerPath: sysrqTriggerPath,
	}, nil
}

// Inject triggers a kernel panic through the sysrq trigger
func (i nodeFailureInjector) Inject() {
	var err error

	// handle metrics
	// those ones might not be sent because of the node crash, it is better
	// to rely on the controller metrics for this failure injection
	defer func() {
		i.handleMetricSinkError(i.ms.MetricInjected("", i.uid, err == nil, i.kind, []string{}))
	}()

	i.log.Infow("injecting a node failure by triggering a kernel panic",
		"sysrq_path", i.sysrqPath,
		"sysrq_trigger_path", i.sysrqTriggerPath,
	)

	// Ensure sysrq value is set to 1 (to accept the kernel panic trigger)
	err = i.config.FileWriter.Write(i.sysrqPath, 0644, "1")
	if err != nil {
		i.log.Fatalw("error while writing to the sysrq file",
			"error", err,
			"path", i.sysrqPath,
		)
	}

	// Trigger kernel panic
	i.log.Infow("the injector is about to write to the sysrq trigger file")
	i.log.Infow("from this point, if no fatal log occurs, the injection succeeded and the system will crash")

	if i.spec.Shutdown {
		err = i.config.FileWriter.Write(i.sysrqTriggerPath, 0200, "o")
	} else {
		err = i.config.FileWriter.Write(i.sysrqTriggerPath, 0200, "c")
	}

	if err != nil {
		i.log.Fatalw("error while writing to the sysrq trigger file",
			"error", err,
			"path", i.sysrqTriggerPath,
		)
	}
}

func (i nodeFailureInjector) Clean() {}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
)

const (
	nodeFailureSysrqPath        = "/mnt/sysrq"
	nodeFailureSysrqTriggerPath = "/mnt/sysrq-trigger"
)

// NodeFailureInjector describes a node failure injector
type NodeFailureInjector struct {
	Injector
	Spec       *v1beta1.NodeFailureSpec
	FileWriter FileWriter
}

// Inject triggers a kernel panic through the sysrq trigger
func (i NodeFailureInjector) Inject() {
	i.Log.Infow("injecting a node failure by triggering a kernel panic",
		"sysrq_path", nodeFailureSysrqPath,
		"sysrq_trigger_path", nodeFailureSysrqTriggerPath,
	)

	// Ensure sysrq value is set to 1 (to accept the kernel panic trigger)
	err := i.FileWriter.Write(nodeFailureSysrqPath, 0644, "1")
	if err != nil {
		i.Log.Fatalw("error while writing to the sysrq file",
			"error", err,
			"path", nodeFailureSysrqPath,
		)
	}

	// Trigger kernel panic
	i.Log.Infow("the injector is about to write to the sysrq trigger file")
	i.Log.Infow("from this point, if no fatal log occurs, the injection succeeded and the system will crash")
	if i.Spec.Shutdown {
		err = i.FileWriter.Write(nodeFailureSysrqTriggerPath, 0200, "o")
	} else {
		err = i.FileWriter.Write(nodeFailureSysrqTriggerPath, 0200, "c")
	}
	if err != nil {
		i.Log.Fatalw("error while writing to the sysrq trigger file",
			"error", err,
			"path", nodeFailureSysrqTriggerPath,
		)
	}
}

package injector

import (
	"os"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/logger"
)

const (
	nodeFailureSysrqPath        = "/mnt/sysrq"
	nodeFailureSysrqTriggerPath = "/mnt/sysrq-trigger"
	nodeFailureMetricPrefix     = "chaos.nofi"
)

var nodeFailureEventTags = []string{"failure_kind:node_failure"}

// NodeFailureInjector describes a node failure injector
type NodeFailureInjector struct {
	Injector
	Spec *v1beta1.NodeFailureInjectionSpec
}

// Inject triggers a kernel panic through the sysrq trigger
func (i NodeFailureInjector) Inject() {
	logger.Instance().Infow("injecting a node failure by triggering a kernel panic",
		"sysrq_path", nodeFailureSysrqPath,
		"sysrq_trigger_path", nodeFailureSysrqTriggerPath,
	)

	// Ensure sysrq value is set to 1 (to accept the kernel panic trigger)
	err := write(nodeFailureSysrqPath, 0644, "1")
	if err != nil {
		logger.Instance().Fatalw("error while writing to the sysrq file",
			"error", err,
			"path", nodeFailureSysrqPath,
		)
	}

	// Trigger kernel panic
	logger.Instance().Infow("the injector is about to write to the sysrq trigger file")
	logger.Instance().Infow("from this point, if no fatal log occurs, the injection succeeded and the system will crash")
	if i.Spec.Shutdown {
		err = write(nodeFailureSysrqTriggerPath, 0200, "o")
	} else {
		err = write(nodeFailureSysrqTriggerPath, 0200, "c")
	}
	if err != nil {
		logger.Instance().Fatalw("error while writing to the sysrq trigger file",
			"error", err,
			"path", nodeFailureSysrqTriggerPath,
		)
	}
}

// write writes the given data to the given path without creating it if it doesn't exist
func write(path string, mode os.FileMode, data string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, err = f.WriteString(data)
	if err != nil {
		return err
	}
	return f.Close()
}

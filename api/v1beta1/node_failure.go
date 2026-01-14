// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

// NodeFailureSpec represents a node failure injection
type NodeFailureSpec struct {
	Shutdown bool `json:"shutdown,omitempty"`
}

// Validate validates args for the given disruption
func (s *NodeFailureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NodeFailureSpec) GenerateArgs() []string {
	args := []string{
		"node-failure",
		"inject",
	}

	if s.Shutdown {
		args = append(args, "--shutdown")
	}

	return args
}

func (s *NodeFailureSpec) Explain() []string {
	var explanation string
	if s.Shutdown {
		explanation = "spec.nodeFailure.shutdown writes an \"o\" to the kernel's sysrq-trigger file, shutting down the host immediately. " +
			"This will affect ALL pods on the host node. Depending on cloud provider behavior, the node may not be restarted at all, and " +
			"all pods might be rescheduled onto other nodes."
	} else {
		explanation = "spec.nodeFailure will trigger a kernel panic on the node by writing to the host's sysrq-trigger file. " +
			"This will affect ALL pods on the host node. Depending on cloud provider behavior, the node may not be restarted at all, and " +
			"all pods might be rescheduled onto other nodes."
	}

	return []string{"", explanation}
}

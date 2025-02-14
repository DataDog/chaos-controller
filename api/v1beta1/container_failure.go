// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

// ContainerFailureSpec represents a container failure injection
type ContainerFailureSpec struct {
	Forced bool `json:"forced,omitempty"`
}

// Validate validates args for the given disruption
func (s *ContainerFailureSpec) Validate() error {
	return nil
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *ContainerFailureSpec) GenerateArgs() []string {
	args := []string{
		"container-failure",
	}

	if s.Forced {
		args = append(args, "--forced")
	}

	return args
}

func (s *ContainerFailureSpec) Explain() []string {
	var explanation string
	if s.Forced {
		explanation = "spec.containerFailure.forced injects a container failure which sends the SIGKILL signal to the pod's container(s). " +
			"If you'd prefer a SIGTERM, remove containerFailure.forced."
	} else {
		explanation = "spec.containerFailure injects a container failure which sends the SIGTERM signal to the pod's container(s). " +
			"If you'd prefer a SIGKILL, set containerFailure.forced."
	}

	return []string{"", explanation}
}

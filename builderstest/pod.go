// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package builderstest_test

import v1 "k8s.io/api/core/v1"

// PodsBuilder is a list of PodBuilder.
type PodsBuilder []*PodBuilder

// PodBuilder is a struct used for building v1.Pod instances with modifications.
type PodBuilder struct {
	*v1.Pod             // The built v1.Pod instance
	parent  PodsBuilder // The parent PodsBuilder instance associated with this PodBuilder
}

// NewPodsBuilder creates a new PodsBuilder instance with predefined pod data.
func NewPodsBuilder() PodsBuilder {
	return PodsBuilder{
		{
			Pod: &v1.Pod{
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							State: v1.ContainerState{},
						},
					},
				},
			},
		},
		{
			Pod: &v1.Pod{
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							State: v1.ContainerState{},
						},
					},
				},
			},
		},
	}
}

// Build constructs and returns a slice of v1.Pod based on the configuration set in the PodsBuilder.
func (p PodsBuilder) Build() []v1.Pod {
	if p == nil {
		return nil
	}

	pods := make([]v1.Pod, 0, len(p))

	for _, pod := range p {
		pods = append(pods, *pod.Pod)
	}

	return pods
}

// Take returns a pointer to a PodBuilder for the specified index from the PodsBuilder.
func (p PodsBuilder) Take(index int) *PodBuilder {
	// Check if the parent of the PodBuilder at the specified index is uninitialized (nil).
	// If uninitialized, set the parent of the PodBuilder to the current PodsBuilder.
	if p[index].parent == nil {
		p[index].parent = p
	}

	return p[index]
}

// One returns a pointer to a PodBuilder for the first pod in the PodsBuilder.
func (p PodsBuilder) One() *PodBuilder {
	return p.Take(0)
}

// Two returns a pointer to a PodBuilder for the second pod in the PodsBuilder.
func (p PodsBuilder) Two() *PodBuilder {
	return p.Take(1)
}

// Parent returns the parent PodsBuilder associated with the PodBuilder.
func (p *PodBuilder) Parent() PodsBuilder {
	return p.parent
}

// TerminatedWith sets the termination state of the container in the Pod to a terminated state with the specified exit code.
func (p *PodBuilder) TerminatedWith(exitCode int32) *PodBuilder {
	p.Pod.Status.ContainerStatuses[0].State.Terminated = &v1.ContainerStateTerminated{
		ExitCode: exitCode,
	}

	return p
}

// Terminated sets the termination state of the container in the Pod to a terminated state with exit code 0.
func (p *PodBuilder) Terminated() *PodBuilder {
	return p.TerminatedWith(0)
}

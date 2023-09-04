// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package builders

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DisruptionBuilder is a struct used to build a disruption instance.
type DisruptionBuilder struct {
	*v1beta1.Disruption
	// we store action we want to perform instead of performing them right away because they are time sensitive
	// this enables us to ensure time.Now is as late as it can be without faking it (that we should do at some point)
	modifiers []func()
}

// NewDisruptionBuilder creates a new DisruptionBuilder instance with an initial disruption spec and a creation timestamp modifier.
func NewDisruptionBuilder() *DisruptionBuilder {
	return (&DisruptionBuilder{
		Disruption: &v1beta1.Disruption{
			Spec: v1beta1.DisruptionSpec{
				Duration: "1m", // per spec definition a valid disruption going to the reconcile loop MUST have a duration, let's not test wrong test cases
			},
		},
	}).WithCreation(30 * time.Second)
}

// WithDisruptionKind sets the specified kind of disruption in the DisruptionBuilder's spec.
func (b *DisruptionBuilder) WithDisruptionKind(kind types.DisruptionKindName) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			switch kind {
			case types.DisruptionKindNodeFailure:
				if b.Spec.NodeFailure == nil {
					b.Spec.NodeFailure = &v1beta1.NodeFailureSpec{}
				}
			case types.DisruptionKindContainerFailure:
				if b.Spec.ContainerFailure == nil {
					b.Spec.ContainerFailure = &v1beta1.ContainerFailureSpec{}
				}
			case types.DisruptionKindNetworkDisruption:
				if b.Spec.Network == nil {
					b.Spec.Network = &v1beta1.NetworkDisruptionSpec{}
				}
			case types.DisruptionKindCPUPressure:
				if b.Spec.CPUPressure == nil {
					b.Spec.CPUPressure = &v1beta1.CPUPressureSpec{}
				}
			case types.DisruptionKindDiskPressure:
				if b.Spec.DiskPressure == nil {
					b.Spec.DiskPressure = &v1beta1.DiskPressureSpec{}
				}
			case types.DisruptionKindDNSDisruption:
				if b.Spec.DNS == nil {
					b.Spec.DNS = v1beta1.DNSDisruptionSpec{}
				}
			case types.DisruptionKindGRPCDisruption:
				if b.Spec.GRPC == nil {
					b.Spec.GRPC = &v1beta1.GRPCDisruptionSpec{}
				}
			case types.DisruptionKindDiskFailure:
				if b.Spec.DiskFailure == nil {
					b.Spec.DiskFailure = &v1beta1.DiskFailureSpec{}
				}
			}
		})

	return b
}

// Build generates a v1.Disruption instance based on the configuration.
func (b *DisruptionBuilder) Build() v1beta1.Disruption {
	for _, modifier := range b.modifiers {
		modifier()
	}

	return *b.Disruption
}

// Reset resets the DisruptionBuilder by clearing all modifiers.
func (b *DisruptionBuilder) Reset() *DisruptionBuilder {
	b.modifiers = nil

	return b
}

// WithCreation adjusts the creation timestamp.
func (b *DisruptionBuilder) WithCreation(past time.Duration) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.CreationTimestamp = v1.NewTime(time.Now().Add(-past))
		})

	return b
}

// WithDeletion sets the deletion timestamp to the current time.
func (b *DisruptionBuilder) WithDeletion() *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			v1t := v1.NewTime(time.Now())

			b.DeletionTimestamp = &v1t
		})

	return b
}

// WithNamespace sets the namespace.
func (b *DisruptionBuilder) WithNamespace(namespace string) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Namespace = namespace
		})

	return b
}

// WithName sets the name.
func (b *DisruptionBuilder) WithName(name string) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Name = name
		})

	return b
}

// WithNetworkDisruptionCloudSpec sets the NetworkDisruptionCloudSpecs to the Network spec.
func (b *DisruptionBuilder) WithNetworkDisruptionCloudSpec(spec *v1beta1.NetworkDisruptionCloudSpec) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			if b.Spec.Network == nil {
				b.Spec.Network = &v1beta1.NetworkDisruptionSpec{}
			}

			b.Spec.Network.Cloud = spec
		})

	return b
}

// WithSpecPulse sets the DisruptionPulse to the Pulse spec.
func (b *DisruptionBuilder) WithSpecPulse(specPulse *v1beta1.DisruptionPulse) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Spec.Pulse = specPulse
		})

	return b
}

// WithNetworkDisableDefaultAllowedHosts set the NetworkDisruptionSpec to the network spec.
func (b *DisruptionBuilder) WithNetworkDisableDefaultAllowedHosts(enable bool) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			if b.Spec.Network == nil {
				b.Spec.Network = &v1beta1.NetworkDisruptionSpec{}
			}

			b.Spec.Network.DisableDefaultAllowedHosts = enable
		})

	return b
}

// WithInjectionStatus sets the specified injection status in the DisruptionBuilder's status.
func (b *DisruptionBuilder) WithInjectionStatus(status types.DisruptionInjectionStatus) *DisruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Status.InjectionStatus = status
		})

	return b
}

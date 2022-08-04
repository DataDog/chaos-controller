// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

// UnsafemodeSpec represents a spec with parameters to turn off specific safety nets designed to catch common traps or issues running a disruption
// All of these are turned off by default, so disabling safety nets requires manually changing these booleans to true
type UnsafemodeSpec struct {
	DisableAll                 bool    `json:"disableAll,omitempty"`
	DisableCountTooLarge       bool    `json:"disableCountTooLarge,omitempty"`
	DisableNeitherHostNorPort  bool    `json:"disableNeitherHostNorPort,omitempty"`
	DisableSpecificContainDisk bool    `json:"disableSpecificContainDisk,omitempty"`
	Config                     *Config `json:"config,omitempty"`
}

// Config represents any configurable parameters for the safetynets, all of which have defaults
type Config struct {
	CountTooLarge *CountTooLargeConfig `json:"countTooLarge,omitempty"`
}

// CountTooLargeConfig represents the configuration for the countTooLarge safetynet
// +ddmark:validation:AtLeastOneOf={NamespaceThreshold,ClusterThreshold}
type CountTooLargeConfig struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=1
	// +ddmark:validation:Maximum=100
	NamespaceThreshold *int `json:"namespaceThreshold,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=1
	// +ddmark:validation:Maximum=100
	ClusterThreshold *int `json:"clusterThreshold,omitempty"`
}

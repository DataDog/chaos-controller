// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

// UnsafemodeSpec represents a spec with parameters to turn off specific safety nets designed to catch common traps or issues running a disruption
// All of these are turned off by default, so disabling safety nets requires manually changing these booleans to true
type UnsafemodeSpec struct {
	DisableAll                 bool    `json:"disableAll,omitempty"`
	DisableCountTooLarge       bool    `json:"disableCountTooLarge,omitempty"`
	DisableNeitherHostNorPort  bool    `json:"disableNeitherHostNorPort,omitempty"`
	DisableSpecificContainDisk bool    `json:"disableSpecificContainDisk,omitempty"`
	AllowRootDiskFailure       bool    `json:"allowRootDiskFailure,omitempty"`
	Config                     *Config `json:"config,omitempty"`
}

// Config represents any configurable parameters for the safetynets, all of which have defaults
type Config struct {
	CountTooLarge *CountTooLargeConfig `json:"countTooLarge,omitempty"`
}

// CountTooLargeConfig represents the configuration for the countTooLarge safetynet
type CountTooLargeConfig struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	NamespaceThreshold *int `json:"namespaceThreshold,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	ClusterThreshold *int `json:"clusterThreshold,omitempty"`
}

// MaxClusterThreshold is the float64 of 1.0, representing 100%. clusterThreshold is passed into the config
// as integer percentage values from 1-100, and then divided by 100.0 for working with in safetyNetCountNotTooLarge
const MaxClusterThreshold = float64(1)

// MaxNamespaceThreshold is the float64 of 1.0, representing 100%.  namespaceThreshold is passed into the config
// as integer percentage values from 1-100, and then divided by 100.0 for working with in safetyNetCountNotTooLarge
const MaxNamespaceThreshold = float64(1)

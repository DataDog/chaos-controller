// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package types

type CloudProviderName string

const (
	CloudProviderDatadog CloudProviderName = "Datadog"
	CloudProviderGCP     CloudProviderName = "GCP"
	CloudProviderAWS     CloudProviderName = "AWS"
)

// CloudProviderIPRangeInfo information related to the ip ranges pulled from a cloud provider
type CloudProviderIPRangeInfo struct {
	Version                  string
	CloudProviderServiceName CloudProviderName
	IPRanges                 map[string][]string
}

// CloudProviderConfig Single configuration for any cloud provider
type CloudProviderConfig struct {
	IPRangesURL string `json:"iprangesurl"`
}

// CloudProviderConfigs all cloud provider configurations for the manager
type CloudProviderConfigs struct {
	PullInterval string              `json:"pullinterval"`
	Aws          CloudProviderConfig `json:"aws"`
}

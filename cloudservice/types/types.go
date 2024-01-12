// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package types

import "time"

type CloudProviderName string

const (
	CloudProviderDatadog CloudProviderName = "Datadog"
	CloudProviderGCP     CloudProviderName = "GCP"
	CloudProviderAWS     CloudProviderName = "AWS"
)

var (
	AllCloudProviders = []CloudProviderName{CloudProviderAWS, CloudProviderGCP, CloudProviderDatadog}
)

// CloudProviderIPRangeInfo information related to the ip ranges pulled from a cloud provider
type CloudProviderIPRangeInfo struct {
	Version     string
	IPRanges    map[string][]string
	ServiceList []string // Makes the process of getting the services names easier
}

// CloudProviderConfig Single configuration for any cloud provider
type CloudProviderConfig struct {
	Enabled     bool   `json:"enabled"`
	IPRangesURL string `json:"ipRangesURL"`
}

// CloudProviderConfigs all cloud provider configurations for the manager
type CloudProviderConfigs struct {
	DisableAll   bool                `json:"disableAll"`
	PullInterval time.Duration       `json:"pullInterval"`
	AWS          CloudProviderConfig `json:"aws"`
	GCP          CloudProviderConfig `json:"gcp"`
	Datadog      CloudProviderConfig `json:"datadog"`
}

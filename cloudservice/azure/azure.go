// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package azure

import (
	"encoding/json"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudProviderIPRangeManager struct{}

type AzureIPRange struct {
	IPPrefix string `json:"ip_prefix"`
	Service  string `json:"service"`
}

type AzureIPRanges struct {
	SyncToken string         `json:"syncToken"`
	Prefixes  []AzureIPRange `json:"prefixes"`
}

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) (bool, error) {
	ipRanges := AzureIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false, err
	}

	return ipRanges.SyncToken != oldVersion, nil
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from AWS to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := AzureIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	result := &types.CloudProviderIPRangeInfo{
		ServiceList: []string{},
		IPRanges:    make(map[string][]string),
		Version:     ipRanges.SyncToken,
	}

	for _, ipRange := range ipRanges.Prefixes {
		// TODO check for the global service issue that aws has
		if ipRange.Service == "AZURE" || ipRange.IPPrefix == "" {
			continue
		}

		if len(result.IPRanges[ipRange.Service]) == 0 {
			result.ServiceList = append(result.ServiceList, ipRange.Service)
		}

		result.IPRanges[ipRange.Service] = append(result.IPRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return result, nil
}

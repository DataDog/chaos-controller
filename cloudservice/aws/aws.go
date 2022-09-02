// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package aws

import (
	"encoding/json"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudProviderIPRangeManager struct {
}

type AWSIPRange struct {
	IPPrefix           string `json:"ip_prefix"`
	Region             string `json:"region"`
	Service            string `json:"service"`
	NetworkBorderGroup string `json:"network_border_group"`
}

type AWSIPRanges struct {
	SyncToken string       `json:"syncToken"`
	Prefixes  []AWSIPRange `json:"prefixes"`
}

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

func (s *CloudProviderIPRangeManager) GetName() types.CloudProviderName {
	return types.CloudProviderAWS
}

func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldIPRangesInfo types.CloudProviderIPRangeInfo) bool {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false
	}

	return ipRanges.SyncToken != oldIPRangesInfo.Version
}

func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	genericIPRanges := types.CloudProviderIPRangeInfo{
		CloudProviderServiceName: types.CloudProviderAWS,
		Version:                  ipRanges.SyncToken,
		IPRanges:                 make(map[string][]string),
	}

	for _, ipRange := range ipRanges.Prefixes {
		if len(genericIPRanges.IPRanges[ipRange.Service]) == 0 {
			genericIPRanges.IPRanges[ipRange.Service] = []string{}
		}

		genericIPRanges.IPRanges[ipRange.Service] = append(genericIPRanges.IPRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return &genericIPRanges, nil
}

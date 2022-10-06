// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package gcp

import (
	"encoding/json"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudProviderIPRangeManager struct {
}

// GCPIpRange from the model of the ip range file from GCP
type GCPIPRange struct {
	IPPrefix string `json:"ipv4Prefix"`
	Region   string `json:"scope"`
	Service  string `json:"service"`
}

// GCPIpRanges from the model of the ip range file from GCP
type GCPIPRanges struct {
	SyncToken string       `json:"syncToken"`
	Prefixes  []GCPIPRange `json:"prefixes"`
}

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

// IsNewVersion Check if the ip ranges pulled are newer than the one we already have
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldIPRangesInfo types.CloudProviderIPRangeInfo) bool {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false
	}

	return ipRanges.SyncToken != oldIPRangesInfo.Version
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from GCP to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	genericIPRanges := types.CloudProviderIPRangeInfo{
		CloudProviderServiceName: types.CloudProviderGCP,
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

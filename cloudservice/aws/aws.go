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

// AWSIpRange from the model of the ip range file from AWS
type AWSIPRange struct {
	IPPrefix           string `json:"ip_prefix"`
	Region             string `json:"region"`
	Service            string `json:"service"`
	NetworkBorderGroup string `json:"network_border_group"`
}

// AWSIpRanges from the model of the ip range file from AWS
type AWSIPRanges struct {
	SyncToken string       `json:"syncToken"`
	Prefixes  []AWSIPRange `json:"prefixes"`
}

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

// IsNewVersion Check if the ip ranges pulled are newer than the one we already have
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) (bool, error) {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false, err
	}

	return ipRanges.SyncToken != oldVersion, nil
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from AWS to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	result := &types.CloudProviderIPRangeInfo{
		ServiceList: []string{},
		IPRanges:    make(map[string][]string),
		Version:     ipRanges.SyncToken,
	}

	for _, ipRange := range ipRanges.Prefixes {
		// the service AMAZON is the list of all ip ranges of all services + misc ones. We don't need that
		// it's also too big for us to be able to filter all ips
		// remove empty IPPrefix for safety
		if ipRange.Service == "AMAZON" || ipRange.IPPrefix == "" {
			continue
		}

		if len(result.IPRanges[ipRange.Service]) == 0 {
			result.ServiceList = append(result.ServiceList, ipRange.Service)
		}

		result.IPRanges[ipRange.Service] = append(result.IPRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return result, nil
}

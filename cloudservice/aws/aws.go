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
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) bool {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false
	}

	return ipRanges.SyncToken != oldVersion
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from AWS to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := AWSIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	serviceList := []string{}
	genericIPRanges := make(map[string][]string)

	for _, ipRange := range ipRanges.Prefixes {
		// this service is the list of all ip ranges of all services + misc ones. We don't need that
		// it's also too big for us to be able to filter all ips
		if ipRange.Service == "AMAZON" || ipRange.IPPrefix == "" {
			continue
		}

		if len(genericIPRanges[ipRange.Service]) == 0 {
			serviceList = append(serviceList, ipRange.Service)
			genericIPRanges[ipRange.Service] = []string{}
		}

		genericIPRanges[ipRange.Service] = append(genericIPRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return &types.CloudProviderIPRangeInfo{
		Version:     ipRanges.SyncToken,
		ServiceList: serviceList,
		IPRanges:    genericIPRanges,
	}, nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package gcp

import (
	"encoding/json"
	"strings"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudProviderIPRangeManager struct {
}

// GCPIpRange from the model of the ip range file from GCP
type GCPIPRange struct {
	IPPrefix string `json:"ipv4Prefix"`
}

// GCPIpRanges from the model of the ip range file from GCP
type GCPIPRanges struct {
	SyncToken string       `json:"syncToken"`
	Prefixes  []GCPIPRange `json:"prefixes"`
}

const (
	// As of today, the file used to parse google ip ranges does not contain information about which ip ranges is assigned to which service
	// We assign every ip ranges to the service "Google" for this reason
	GoogleCloudService = "Google"
)

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

// IsNewVersion Check if the ip ranges pulled are newer than the one we already have
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) (bool, error) {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false, err
	}

	return ipRanges.SyncToken != oldVersion, nil
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from GCP to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	result := &types.CloudProviderIPRangeInfo{
		ServiceList: []string{GoogleCloudService},
		IPRanges: map[string][]string{
			GoogleCloudService: {},
		},
		Version: ipRanges.SyncToken,
	}

	for _, ipRange := range ipRanges.Prefixes {
		// Remove empty IPPrefixes (can happen if we only have IpV6) and remove the dns servers of Google in the list of ip ranges available to disrupt
		if ipRange.IPPrefix == "" || strings.HasPrefix(ipRange.IPPrefix, "8.8") {
			continue
		}

		result.IPRanges[GoogleCloudService] = append(result.IPRanges[GoogleCloudService], ipRange.IPPrefix)
	}

	return result, nil
}

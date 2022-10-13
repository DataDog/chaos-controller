// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package gcp

import (
	"encoding/json"
	"strings"
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
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) bool {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false
	}

	return ipRanges.SyncToken != oldVersion
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from GCP to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (string, map[string][]string, error) {
	ipRanges := GCPIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return "", nil, err
	}

	genericIPRanges := make(map[string][]string)

	for _, ipRange := range ipRanges.Prefixes {
		if ipRange.Service == "" {
			ipRange.Service = "Google Cloud"
		}

		if len(genericIPRanges[ipRange.Service]) == 0 {
			genericIPRanges[ipRange.Service] = []string{}
		}

		// Remove empty and remove the dns servers of Google in the list of ip ranges available to disrupt
		if ipRange.IPPrefix == "" || strings.HasPrefix(ipRange.IPPrefix, "8.8") {
			continue
		}

		genericIPRanges[ipRange.Service] = append(genericIPRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return ipRanges.SyncToken, genericIPRanges, nil
}

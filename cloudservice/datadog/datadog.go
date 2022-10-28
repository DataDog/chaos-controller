// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package datadog

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudProviderIPRangeManager struct {
}

type DatadogPrefixes struct {
	IPv4 []string `json:"prefixes_ipv4"`
}

// DatadogIPRanges from the model of the ip range file from Datadog
type DatadogIPRanges struct {
	Version  int
	IPRanges map[string][]string
}

// UnmarshalJSON custom unmarshalling method as the ip range file from Datadog is essentially map[product-name] = {ipV4: [ip], ...}
func (c *DatadogIPRanges) UnmarshalJSON(data []byte) error {
	allData := make(map[string]interface{})

	c.IPRanges = map[string][]string{}

	// we first unmarshall to a map to extract all the possible products. We don't have a dedicated struct because we could have new products adding up
	err := json.Unmarshal(data, &allData)
	if err != nil {
		return fmt.Errorf("failing on initial unmarshal of the datadog ip ranges file: %s", err)
	}

	for field, fieldValue := range allData {
		// we set the version of the file
		if field == "version" {
			versionFloat, ok := fieldValue.(float64)
			if !ok {
				return fmt.Errorf("failing to retrieve the version of the datadog ip ranges file. Version is not type float64")
			}

			c.Version = int(versionFloat)
		} else if field != "modified" {
			// we marshal the ip prefixes we previously unmarshalled to be able to unmarshall it using a dedicated struct
			jsonPrefixes, err := json.Marshal(fieldValue)
			if err != nil {
				return fmt.Errorf("failing to marshal the datadog ip ranges file previously unmarshalled: %s", err)
			}

			ipRanges := DatadogPrefixes{}

			err = json.Unmarshal(jsonPrefixes, &ipRanges)
			if err != nil {
				return fmt.Errorf("failing on final unmarshalling to extract the ip ranges of the datadog ip ranges file: %s", err)
			}

			if len(ipRanges.IPv4) > 0 {
				c.IPRanges[field] = ipRanges.IPv4
			}
		}
	}

	return nil
}

func New() *CloudProviderIPRangeManager {
	return &CloudProviderIPRangeManager{}
}

// IsNewVersion Check if the ip ranges pulled are newer than the one we already have
func (s *CloudProviderIPRangeManager) IsNewVersion(newIPRanges []byte, oldVersion string) (bool, error) {
	ipRanges := DatadogIPRanges{}
	if err := json.Unmarshal(newIPRanges, &ipRanges); err != nil {
		return false, err
	}

	return strconv.Itoa(ipRanges.Version) != oldVersion, nil
}

// ConvertToGenericIPRanges From an unmarshalled json ip range file from Datadog to a generic ip range struct
func (s *CloudProviderIPRangeManager) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	ipRanges := DatadogIPRanges{}
	if err := json.Unmarshal(unparsedIPRanges, &ipRanges); err != nil {
		return nil, err
	}

	genericIPRanges := types.CloudProviderIPRangeInfo{
		Version:     strconv.Itoa(ipRanges.Version),
		ServiceList: []string{},
		IPRanges:    ipRanges.IPRanges,
	}

	for service := range ipRanges.IPRanges {
		genericIPRanges.ServiceList = append(genericIPRanges.ServiceList, service)
	}

	return &genericIPRanges, nil
}

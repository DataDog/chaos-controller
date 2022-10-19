// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package cloudservice

import (
	"github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/stretchr/testify/mock"
)

// CloudserviceMock mocking struct to test the logic in manager.go
// for now we pass the results of ConvertToGenericIPRanges. It enables us to run the manager without parsing the actual ip ranges file
type CloudserviceMock struct {
	mock.Mock

	isNewVersionMockValue              bool
	convertToGenericIPRangesVersion    string
	convertToGenericIPRanges           map[string][]string
	converToGenericIPRangesServiceList []string
	convertToGenericIPRangesError      error
}

func NewCloudServiceMock(isNewVersionMockValue bool, convertToGenericIPRangesVersion string, convertToGenericIPRangesServiceList []string, convertToGenericIPRanges map[string][]string, convertToGenericIPRangesError error) *CloudserviceMock {
	return &CloudserviceMock{
		isNewVersionMockValue:              isNewVersionMockValue,
		convertToGenericIPRangesVersion:    convertToGenericIPRangesVersion,
		convertToGenericIPRanges:           convertToGenericIPRanges,
		converToGenericIPRangesServiceList: convertToGenericIPRangesServiceList,
		convertToGenericIPRangesError:      convertToGenericIPRangesError,
	}
}

func (a *CloudserviceMock) IsNewVersion(newIPRanges []byte, oldVersion string) bool {
	return a.isNewVersionMockValue
}

func (a *CloudserviceMock) ConvertToGenericIPRanges(unparsedIPRanges []byte) (*types.CloudProviderIPRangeInfo, error) {
	return &types.CloudProviderIPRangeInfo{
		Version:     a.convertToGenericIPRangesVersion,
		IPRanges:    a.convertToGenericIPRanges,
		ServiceList: a.converToGenericIPRangesServiceList,
	}, a.convertToGenericIPRangesError
}

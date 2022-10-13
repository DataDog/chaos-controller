// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package cloudservice

import (
	"github.com/stretchr/testify/mock"
)

type CloudserviceMock struct {
	mock.Mock

	isNewVersionMockValue           bool
	convertToGenericIPRangesVersion string
	convertToGenericIPRanges        map[string][]string
	convertToGenericIPRangesError   error
}

func NewCloudServiceMock(isNewVersionMockValue bool, convertToGenericIPRangesVersion string, convertToGenericIPRanges map[string][]string, convertToGenericIPRangesError error) *CloudserviceMock {
	return &CloudserviceMock{
		isNewVersionMockValue:           isNewVersionMockValue,
		convertToGenericIPRangesVersion: convertToGenericIPRangesVersion,
		convertToGenericIPRanges:        convertToGenericIPRanges,
		convertToGenericIPRangesError:   convertToGenericIPRangesError,
	}
}

func (a *CloudserviceMock) IsNewVersion(newIPRanges []byte, oldVersion string) bool {
	return a.isNewVersionMockValue
}

func (a *CloudserviceMock) ConvertToGenericIPRanges(unparsedIPRanges []byte) (string, map[string][]string, error) {
	return a.convertToGenericIPRangesVersion, a.convertToGenericIPRanges, a.convertToGenericIPRangesError
}

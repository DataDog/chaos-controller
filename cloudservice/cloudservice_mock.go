package cloudservice

import (
	"github.com/stretchr/testify/mock"
)

type CloudserviceMock struct {
	mock.Mock

	isNewVersionMockValue           bool
	convertToGenericIpRangesVersion string
	convertToGenericIpRanges        map[string][]string
	convertToGenericIpRangesError   error
}

func NewCloudServiceMock(isNewVersionMockValue bool, convertToGenericIpRangesVersion string, convertToGenericIpRanges map[string][]string, convertToGenericIpRangesError error) *CloudserviceMock {
	return &CloudserviceMock{
		isNewVersionMockValue:           isNewVersionMockValue,
		convertToGenericIpRangesVersion: convertToGenericIpRangesVersion,
		convertToGenericIpRanges:        convertToGenericIpRanges,
		convertToGenericIpRangesError:   convertToGenericIpRangesError,
	}
}

func (a *CloudserviceMock) IsNewVersion(newIPRanges []byte, oldVersion string) bool {
	return a.isNewVersionMockValue
}

func (a *CloudserviceMock) ConvertToGenericIPRanges(unparsedIPRanges []byte) (string, map[string][]string, error) {
	return a.convertToGenericIpRangesVersion, a.convertToGenericIpRanges, a.convertToGenericIpRangesError
}

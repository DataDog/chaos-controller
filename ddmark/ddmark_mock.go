// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark

import (
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/mock"
)

// DDMarkMock is a mock of the DDMark interface
type DDMarkMock struct {
	mock.Mock
}

//nolint:golint
func (d *DDMarkMock) ValidateStruct(marshalledStruct interface{}, filePath string) []error {
	args := d.Called(marshalledStruct, filePath)

	return args.Get(0).([]error)
}

//nolint:golint
func (d *DDMarkMock) ValidateStructMultierror(marshalledStruct interface{}, filePath string) (retErr *multierror.Error) {
	args := d.Called(marshalledStruct, filePath)

	return args.Get(0).(*multierror.Error)
}

//nolint:golint
func (d *DDMarkMock) CleanupLibraries() error {
	args := d.Called()

	return args.Error(0)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark

import (
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/mock"
)

// DdmarkMock is a mock of the Ddmark interface
type DdmarkMock struct {
	mock.Mock
}

//nolint:golint
func (d *DdmarkMock) ValidateStruct(marshalledStruct interface{}, filePath string, structPkgs ...string) []error {
	args := d.Called(marshalledStruct, filePath, structPkgs)

	return args.Get(0).([]error)
}

//nolint:golint
func (d *DdmarkMock) ValidateStructMultierror(marshalledStruct interface{}, filePath string, structPkgs ...string) (retErr *multierror.Error) {
	args := d.Called(marshalledStruct, filePath, structPkgs)

	return args.Get(0).(*multierror.Error)
}

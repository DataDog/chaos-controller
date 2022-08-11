// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package disk

import "github.com/stretchr/testify/mock"

// InformerMock is a mock implementation of the Informer interface
type InformerMock struct {
	mock.Mock
}

//nolint:golint
func (f *InformerMock) Major() int {
	args := f.Called()

	return args.Int(0)
}

//nolint:golint
func (f *InformerMock) Source() string {
	args := f.Called()

	return args.String(0)
}

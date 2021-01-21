// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import "github.com/stretchr/testify/mock"

// RuntimeMock is a mock implementation of the Runtime interface
type RuntimeMock struct {
	mock.Mock
}

//nolint:golint
func (f *RuntimeMock) PID(id string) (uint32, error) {
	args := f.Called(id)

	return args.Get(0).(uint32), args.Error(1)
}

//nolint:golint
func (f *RuntimeMock) CgroupPath(id string) (string, error) {
	args := f.Called(id)

	return args.String(0), args.Error(1)
}

//nolint:golint
func (f *RuntimeMock) HostPath(id, path string) (string, error) {
	args := f.Called(id, path)

	return args.String(0), args.Error(1)
}

//nolint:golint
func (f *RuntimeMock) Name(id string) (string, error) {
	args := f.Called(id)

	return args.String(0), args.Error(1)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cgroup

import "github.com/stretchr/testify/mock"

// ManagerMock is a mock implementation of the Cgroup interface
type ManagerMock struct {
	mock.Mock
}

//nolint:golint
func (f *ManagerMock) Join(kind string, pid int, inherit bool) error {
	args := f.Called(kind, pid, inherit)

	return args.Error(0)
}

//nolint:golint
func (f *ManagerMock) Read(kind, file string) (string, error) {
	args := f.Called(kind, file)

	return args.String(0), args.Error(1)
}

//nolint:golint
func (f *ManagerMock) Write(kind, file, data string) error {
	args := f.Called(kind, file, data)

	return args.Error(0)
}

//nolint:golint
func (f *ManagerMock) Exists(kind string) (bool, error) {
	args := f.Called(kind)

	return args.Bool(0), args.Error(1)
}

//nolint:golint
func (f *ManagerMock) DiskThrottleRead(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

//nolint:golint
func (f *ManagerMock) DiskThrottleWrite(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

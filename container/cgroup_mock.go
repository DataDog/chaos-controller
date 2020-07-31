// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import "github.com/stretchr/testify/mock"

// CgroupMock is a mock implementation of the Cgroup interface
type CgroupMock struct {
	mock.Mock
}

//nolint:golint
func (f *CgroupMock) JoinCPU() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *CgroupMock) DiskThrottleRead(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

//nolint:golint
func (f *CgroupMock) DiskThrottleWrite(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

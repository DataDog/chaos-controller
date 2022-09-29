// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package container

import "github.com/stretchr/testify/mock"

// ContainerMock is a mock implementation of the Container interface
//
//nolint:golint
type ContainerMock struct {
	mock.Mock
}

//nolint:golint
func (f *ContainerMock) ID() string {
	args := f.Called()

	return args.String(0)
}

//nolint:golint
func (f *ContainerMock) Runtime() Runtime {
	args := f.Called()

	return args.Get(0).(Runtime)
}

//nolint:golint
func (f *ContainerMock) CgroupPath() string {
	args := f.Called()

	return args.String(0)
}

//nolint:golint
func (f *ContainerMock) PID() uint32 {
	args := f.Called()

	return args.Get(0).(uint32)
}

//nolint:golint
func (f *ContainerMock) Name() string {
	args := f.Called()

	return args.String(0)
}

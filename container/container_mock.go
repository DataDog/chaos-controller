// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import "github.com/stretchr/testify/mock"

// ContainerMock is a mock implementation of the Container interface
//nolint:golint
type ContainerMock struct {
	mock.Mock
}

//nolint:golint
func (f *ContainerMock) ID() string {
	return "fake"
}

//nolint:golint
func (f *ContainerMock) Runtime() Runtime {
	return nil
}

//nolint:golint
func (f *ContainerMock) Netns() Netns {
	return nil
}

//nolint:golint
func (f *ContainerMock) EnterNetworkNamespace() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *ContainerMock) ExitNetworkNamespace() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *ContainerMock) Cgroup() Cgroup {
	args := f.Called()

	return args.Get(0).(Cgroup)
}

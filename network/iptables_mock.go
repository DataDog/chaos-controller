// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import "github.com/stretchr/testify/mock"

// IptablesMock is a mock implementation of the Iptables interface
type IptablesMock struct {
	mock.Mock
}

//nolint:golint
func (f *IptablesMock) Clear() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) RedirectTo(protocol string, port string, destinationIP string) error {
	args := f.Called(protocol, port, destinationIP)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) Intercept(protocol string, port string, cgroupPath string, cgroupClassID string, injectorPodIP string) error {
	args := f.Called(protocol, port, cgroupPath, cgroupClassID, injectorPodIP)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) MarkCgroupPath(cgroupPath string, mark string) error {
	args := f.Called(cgroupPath, mark)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) MarkClassID(classID string, mark string) error {
	args := f.Called(classID, mark)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) LogConntrack() error {
	args := f.Called()

	return args.Error(0)
}

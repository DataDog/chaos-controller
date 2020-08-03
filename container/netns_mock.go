// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container

import "github.com/stretchr/testify/mock"

// NetnsMock is a mock implementation of the Netns interface
type NetnsMock struct {
	mock.Mock

	Currentns int
	Fakens    int
}

//nolint:golint
func (f *NetnsMock) Set(ns int) error {
	f.Currentns = ns
	args := f.Called(ns)

	return args.Error(0)
}

//nolint:golint
func (f *NetnsMock) GetCurrent() (int, error) {
	args := f.Called()

	return args.Int(0), args.Error(1)
}

//nolint:golint
func (f *NetnsMock) GetFromPID(pid uint32) (int, error) {
	args := f.Called(pid)

	return args.Int(0), args.Error(1)
}

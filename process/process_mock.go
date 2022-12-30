// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package process

import (
	"os"

	"github.com/stretchr/testify/mock"
)

// ManagerMock is a mock implementation of the Manager interface
type ManagerMock struct {
	mock.Mock
}

//nolint:golint
func (f *ManagerMock) Prioritize() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *ManagerMock) ThreadID() int {
	args := f.Called()

	return args.Int(0)
}

//nolint:golint
func (f *ManagerMock) Find(pid int) (*os.Process, error) {
	args := f.Called(pid)

	return args.Get(0).(*os.Process), args.Error(1)
}

//nolint:golint
func (f *ManagerMock) Signal(process *os.Process, signal os.Signal) error {
	args := f.Called(process, signal)

	return args.Error(0)
}

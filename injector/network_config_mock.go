// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// NetworkConfigMock is a mock implementation of the NetworkConfig interface
type NetworkConfigMock struct {
	mock.Mock
}

//nolint:golint
func (f *NetworkConfigMock) AddNetem(delay time.Duration, delayJitter time.Duration, drop int, corrupt int, duplicate int) {
	f.Called(delay, delayJitter, drop, corrupt, duplicate)
}

//nolint:golint
func (f *NetworkConfigMock) AddOutputLimit(bytesPerSec uint) {
	f.Called(bytesPerSec)
}

//nolint:golint
func (f *NetworkConfigMock) ApplyOperations() error {
	args := f.Called()

	return args.Error(0)
}

//nolint:golint
func (f *NetworkConfigMock) ClearOperations() error {
	args := f.Called()

	return args.Error(0)
}

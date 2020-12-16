// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package netns

import "github.com/stretchr/testify/mock"

// ManagerMock is a mock implementation of the Netns interface
type ManagerMock struct {
	mock.Mock
}

//nolint:golint
func (m *ManagerMock) Enter() error {
	args := m.Called()

	return args.Error(0)
}

//nolint:golint
func (m *ManagerMock) Exit() error {
	args := m.Called()

	return args.Error(0)
}

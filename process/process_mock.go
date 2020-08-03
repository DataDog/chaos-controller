// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package process

import "github.com/stretchr/testify/mock"

// ManagerMock is a mock implementation of the Manager interface
type ManagerMock struct {
	mock.Mock
}

//nolint:golint
func (f *ManagerMock) Prioritize() error {
	args := f.Called()

	return args.Error(0)
}

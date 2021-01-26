// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import "github.com/stretchr/testify/mock"

// PythonRunnerMock is a mock implementation of the PythonRunner interface
// used in unit tests
type PythonRunnerMock struct {
	mock.Mock
}

//nolint:golint
func (p *PythonRunnerMock) RunPython(args ...string) (int, string, error) {
	mockArgs := p.Called(args)

	return mockArgs.Int(0), mockArgs.String(1), mockArgs.Error(2)
}

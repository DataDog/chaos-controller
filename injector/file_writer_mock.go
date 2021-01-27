// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"os"

	"github.com/stretchr/testify/mock"
)

// FileWriterMock is a mock implementation of the FileWriter interface
type FileWriterMock struct {
	mock.Mock
}

//nolint:golint
func (fw *FileWriterMock) Write(path string, mode os.FileMode, data string) error {
	args := fw.Called(path, mode, data)

	return args.Error(0)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"github.com/stretchr/testify/mock"
	"os"
)

// BPFDiskFailureCommandMock is a mock implementation of the DiskFailureCmd interface
type BPFDiskFailureCommandMock struct {
	mock.Mock
}

//nolint:golint
func (d *BPFDiskFailureCommandMock) Run(pid int, path string) error {
	args := d.Called(pid, path)

	return args.Error(0)
}

func (d *BPFDiskFailureCommandMock) GetProcess() *os.Process {
	args := d.Called()

	return args.Get(0).(*os.Process)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package stress

import "github.com/stretchr/testify/mock"

// StresserMock is a mock implementation of the Stresser interface
type StresserMock struct {
	mock.Mock
}

//nolint:golint
func (f *StresserMock) Stress(exit <-chan struct{}) {
	f.Called()
	<-exit
}

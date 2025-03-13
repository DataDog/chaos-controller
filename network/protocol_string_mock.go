// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package network

import mock "github.com/stretchr/testify/mock"

// protocolStringMock is an autogenerated mock type for the protocolString type
type protocolStringMock struct {
	mock.Mock
}

type protocolStringMock_Expecter struct {
	mock *mock.Mock
}

func (_m *protocolStringMock) EXPECT() *protocolStringMock_Expecter {
	return &protocolStringMock_Expecter{mock: &_m.Mock}
}

// newProtocolStringMock creates a new instance of protocolStringMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newProtocolStringMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *protocolStringMock {
	mock := &protocolStringMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

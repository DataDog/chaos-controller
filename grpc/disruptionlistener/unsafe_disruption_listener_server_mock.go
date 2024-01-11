// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package disruptionlistener

import mock "github.com/stretchr/testify/mock"

// UnsafeDisruptionListenerServerMock is an autogenerated mock type for the UnsafeDisruptionListenerServer type
type UnsafeDisruptionListenerServerMock struct {
	mock.Mock
}

type UnsafeDisruptionListenerServerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *UnsafeDisruptionListenerServerMock) EXPECT() *UnsafeDisruptionListenerServerMock_Expecter {
	return &UnsafeDisruptionListenerServerMock_Expecter{mock: &_m.Mock}
}

// mustEmbedUnimplementedDisruptionListenerServer provides a mock function with given fields:
func (_m *UnsafeDisruptionListenerServerMock) mustEmbedUnimplementedDisruptionListenerServer() {
	_m.Called()
}

// UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'mustEmbedUnimplementedDisruptionListenerServer'
type UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call struct {
	*mock.Call
}

// mustEmbedUnimplementedDisruptionListenerServer is a helper method to define mock.On call
func (_e *UnsafeDisruptionListenerServerMock_Expecter) mustEmbedUnimplementedDisruptionListenerServer() *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call {
	return &UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call{Call: _e.mock.On("mustEmbedUnimplementedDisruptionListenerServer")}
}

func (_c *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call) Run(run func()) *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call) Return() *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call {
	_c.Call.Return()
	return _c
}

func (_c *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call) RunAndReturn(run func()) *UnsafeDisruptionListenerServerMock_mustEmbedUnimplementedDisruptionListenerServer_Call {
	_c.Call.Return(run)
	return _c
}

// NewUnsafeDisruptionListenerServerMock creates a new instance of UnsafeDisruptionListenerServerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUnsafeDisruptionListenerServerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *UnsafeDisruptionListenerServerMock {
	mock := &UnsafeDisruptionListenerServerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package mocks

import mock "github.com/stretchr/testify/mock"

// NetNSManagerMock is an autogenerated mock type for the Manager type
type NetNSManagerMock struct {
	mock.Mock
}

type NetNSManagerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *NetNSManagerMock) EXPECT() *NetNSManagerMock_Expecter {
	return &NetNSManagerMock_Expecter{mock: &_m.Mock}
}

// Enter provides a mock function with given fields:
func (_m *NetNSManagerMock) Enter() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NetNSManagerMock_Enter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enter'
type NetNSManagerMock_Enter_Call struct {
	*mock.Call
}

// Enter is a helper method to define mock.On call
func (_e *NetNSManagerMock_Expecter) Enter() *NetNSManagerMock_Enter_Call {
	return &NetNSManagerMock_Enter_Call{Call: _e.mock.On("Enter")}
}

func (_c *NetNSManagerMock_Enter_Call) Run(run func()) *NetNSManagerMock_Enter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetNSManagerMock_Enter_Call) Return(_a0 error) *NetNSManagerMock_Enter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetNSManagerMock_Enter_Call) RunAndReturn(run func() error) *NetNSManagerMock_Enter_Call {
	_c.Call.Return(run)
	return _c
}

// Exit provides a mock function with given fields:
func (_m *NetNSManagerMock) Exit() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NetNSManagerMock_Exit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exit'
type NetNSManagerMock_Exit_Call struct {
	*mock.Call
}

// Exit is a helper method to define mock.On call
func (_e *NetNSManagerMock_Expecter) Exit() *NetNSManagerMock_Exit_Call {
	return &NetNSManagerMock_Exit_Call{Call: _e.mock.On("Exit")}
}

func (_c *NetNSManagerMock_Exit_Call) Run(run func()) *NetNSManagerMock_Exit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetNSManagerMock_Exit_Call) Return(_a0 error) *NetNSManagerMock_Exit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetNSManagerMock_Exit_Call) RunAndReturn(run func() error) *NetNSManagerMock_Exit_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewNetNSManagerMock interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetNSManagerMock creates a new instance of NetNSManagerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetNSManagerMock(t mockConstructorTestingTNewNetNSManagerMock) *NetNSManagerMock {
	mock := &NetNSManagerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

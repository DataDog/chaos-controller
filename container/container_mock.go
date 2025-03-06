// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.
package container

import mock "github.com/stretchr/testify/mock"

// ContainerMock is an autogenerated mock type for the Container type
type ContainerMock struct {
	mock.Mock
}

type ContainerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ContainerMock) EXPECT() *ContainerMock_Expecter {
	return &ContainerMock_Expecter{mock: &_m.Mock}
}

// ID provides a mock function with no fields
func (_m *ContainerMock) ID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ContainerMock_ID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ID'
type ContainerMock_ID_Call struct {
	*mock.Call
}

// ID is a helper method to define mock.On call
func (_e *ContainerMock_Expecter) ID() *ContainerMock_ID_Call {
	return &ContainerMock_ID_Call{Call: _e.mock.On("ID")}
}

func (_c *ContainerMock_ID_Call) Run(run func()) *ContainerMock_ID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ContainerMock_ID_Call) Return(_a0 string) *ContainerMock_ID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ContainerMock_ID_Call) RunAndReturn(run func() string) *ContainerMock_ID_Call {
	_c.Call.Return(run)
	return _c
}

// Name provides a mock function with no fields
func (_m *ContainerMock) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// ContainerMock_Name_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Name'
type ContainerMock_Name_Call struct {
	*mock.Call
}

// Name is a helper method to define mock.On call
func (_e *ContainerMock_Expecter) Name() *ContainerMock_Name_Call {
	return &ContainerMock_Name_Call{Call: _e.mock.On("Name")}
}

func (_c *ContainerMock_Name_Call) Run(run func()) *ContainerMock_Name_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ContainerMock_Name_Call) Return(_a0 string) *ContainerMock_Name_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ContainerMock_Name_Call) RunAndReturn(run func() string) *ContainerMock_Name_Call {
	_c.Call.Return(run)
	return _c
}

// PID provides a mock function with no fields
func (_m *ContainerMock) PID() uint32 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for PID")
	}

	var r0 uint32
	if rf, ok := ret.Get(0).(func() uint32); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint32)
	}

	return r0
}

// ContainerMock_PID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PID'
type ContainerMock_PID_Call struct {
	*mock.Call
}

// PID is a helper method to define mock.On call
func (_e *ContainerMock_Expecter) PID() *ContainerMock_PID_Call {
	return &ContainerMock_PID_Call{Call: _e.mock.On("PID")}
}

func (_c *ContainerMock_PID_Call) Run(run func()) *ContainerMock_PID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ContainerMock_PID_Call) Return(_a0 uint32) *ContainerMock_PID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ContainerMock_PID_Call) RunAndReturn(run func() uint32) *ContainerMock_PID_Call {
	_c.Call.Return(run)
	return _c
}

// Runtime provides a mock function with no fields
func (_m *ContainerMock) Runtime() Runtime {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Runtime")
	}

	var r0 Runtime
	if rf, ok := ret.Get(0).(func() Runtime); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Runtime)
		}
	}

	return r0
}

// ContainerMock_Runtime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Runtime'
type ContainerMock_Runtime_Call struct {
	*mock.Call
}

// Runtime is a helper method to define mock.On call
func (_e *ContainerMock_Expecter) Runtime() *ContainerMock_Runtime_Call {
	return &ContainerMock_Runtime_Call{Call: _e.mock.On("Runtime")}
}

func (_c *ContainerMock_Runtime_Call) Run(run func()) *ContainerMock_Runtime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ContainerMock_Runtime_Call) Return(_a0 Runtime) *ContainerMock_Runtime_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ContainerMock_Runtime_Call) RunAndReturn(run func() Runtime) *ContainerMock_Runtime_Call {
	_c.Call.Return(run)
	return _c
}

// NewContainerMock creates a new instance of ContainerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewContainerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ContainerMock {
	mock := &ContainerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package process

import (
	os "os"

	mock "github.com/stretchr/testify/mock"
)

// ManagerMock is an autogenerated mock type for the Manager type
type ManagerMock struct {
	mock.Mock
}

type ManagerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ManagerMock) EXPECT() *ManagerMock_Expecter {
	return &ManagerMock_Expecter{mock: &_m.Mock}
}

// Exists provides a mock function with given fields: pid
func (_m *ManagerMock) Exists(pid int) (bool, error) {
	ret := _m.Called(pid)

	if len(ret) == 0 {
		panic("no return value specified for Exists")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (bool, error)); ok {
		return rf(pid)
	}
	if rf, ok := ret.Get(0).(func(int) bool); ok {
		r0 = rf(pid)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ManagerMock_Exists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exists'
type ManagerMock_Exists_Call struct {
	*mock.Call
}

// Exists is a helper method to define mock.On call
//   - pid int
func (_e *ManagerMock_Expecter) Exists(pid interface{}) *ManagerMock_Exists_Call {
	return &ManagerMock_Exists_Call{Call: _e.mock.On("Exists", pid)}
}

func (_c *ManagerMock_Exists_Call) Run(run func(pid int)) *ManagerMock_Exists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *ManagerMock_Exists_Call) Return(_a0 bool, _a1 error) *ManagerMock_Exists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ManagerMock_Exists_Call) RunAndReturn(run func(int) (bool, error)) *ManagerMock_Exists_Call {
	_c.Call.Return(run)
	return _c
}

// Find provides a mock function with given fields: pid
func (_m *ManagerMock) Find(pid int) (*os.Process, error) {
	ret := _m.Called(pid)

	if len(ret) == 0 {
		panic("no return value specified for Find")
	}

	var r0 *os.Process
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (*os.Process, error)); ok {
		return rf(pid)
	}
	if rf, ok := ret.Get(0).(func(int) *os.Process); ok {
		r0 = rf(pid)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*os.Process)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ManagerMock_Find_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Find'
type ManagerMock_Find_Call struct {
	*mock.Call
}

// Find is a helper method to define mock.On call
//   - pid int
func (_e *ManagerMock_Expecter) Find(pid interface{}) *ManagerMock_Find_Call {
	return &ManagerMock_Find_Call{Call: _e.mock.On("Find", pid)}
}

func (_c *ManagerMock_Find_Call) Run(run func(pid int)) *ManagerMock_Find_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *ManagerMock_Find_Call) Return(_a0 *os.Process, _a1 error) *ManagerMock_Find_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ManagerMock_Find_Call) RunAndReturn(run func(int) (*os.Process, error)) *ManagerMock_Find_Call {
	_c.Call.Return(run)
	return _c
}

// Prioritize provides a mock function with no fields
func (_m *ManagerMock) Prioritize() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Prioritize")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ManagerMock_Prioritize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Prioritize'
type ManagerMock_Prioritize_Call struct {
	*mock.Call
}

// Prioritize is a helper method to define mock.On call
func (_e *ManagerMock_Expecter) Prioritize() *ManagerMock_Prioritize_Call {
	return &ManagerMock_Prioritize_Call{Call: _e.mock.On("Prioritize")}
}

func (_c *ManagerMock_Prioritize_Call) Run(run func()) *ManagerMock_Prioritize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ManagerMock_Prioritize_Call) Return(_a0 error) *ManagerMock_Prioritize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ManagerMock_Prioritize_Call) RunAndReturn(run func() error) *ManagerMock_Prioritize_Call {
	_c.Call.Return(run)
	return _c
}

// ProcessID provides a mock function with no fields
func (_m *ManagerMock) ProcessID() int {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ProcessID")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// ManagerMock_ProcessID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ProcessID'
type ManagerMock_ProcessID_Call struct {
	*mock.Call
}

// ProcessID is a helper method to define mock.On call
func (_e *ManagerMock_Expecter) ProcessID() *ManagerMock_ProcessID_Call {
	return &ManagerMock_ProcessID_Call{Call: _e.mock.On("ProcessID")}
}

func (_c *ManagerMock_ProcessID_Call) Run(run func()) *ManagerMock_ProcessID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ManagerMock_ProcessID_Call) Return(_a0 int) *ManagerMock_ProcessID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ManagerMock_ProcessID_Call) RunAndReturn(run func() int) *ManagerMock_ProcessID_Call {
	_c.Call.Return(run)
	return _c
}

// SetAffinity provides a mock function with given fields: _a0
func (_m *ManagerMock) SetAffinity(_a0 []int) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetAffinity")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]int) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ManagerMock_SetAffinity_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetAffinity'
type ManagerMock_SetAffinity_Call struct {
	*mock.Call
}

// SetAffinity is a helper method to define mock.On call
//   - _a0 []int
func (_e *ManagerMock_Expecter) SetAffinity(_a0 interface{}) *ManagerMock_SetAffinity_Call {
	return &ManagerMock_SetAffinity_Call{Call: _e.mock.On("SetAffinity", _a0)}
}

func (_c *ManagerMock_SetAffinity_Call) Run(run func(_a0 []int)) *ManagerMock_SetAffinity_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]int))
	})
	return _c
}

func (_c *ManagerMock_SetAffinity_Call) Return(_a0 error) *ManagerMock_SetAffinity_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ManagerMock_SetAffinity_Call) RunAndReturn(run func([]int) error) *ManagerMock_SetAffinity_Call {
	_c.Call.Return(run)
	return _c
}

// Signal provides a mock function with given fields: _a0, signal
func (_m *ManagerMock) Signal(_a0 *os.Process, signal os.Signal) error {
	ret := _m.Called(_a0, signal)

	if len(ret) == 0 {
		panic("no return value specified for Signal")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*os.Process, os.Signal) error); ok {
		r0 = rf(_a0, signal)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ManagerMock_Signal_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Signal'
type ManagerMock_Signal_Call struct {
	*mock.Call
}

// Signal is a helper method to define mock.On call
//   - _a0 *os.Process
//   - signal os.Signal
func (_e *ManagerMock_Expecter) Signal(_a0 interface{}, signal interface{}) *ManagerMock_Signal_Call {
	return &ManagerMock_Signal_Call{Call: _e.mock.On("Signal", _a0, signal)}
}

func (_c *ManagerMock_Signal_Call) Run(run func(_a0 *os.Process, signal os.Signal)) *ManagerMock_Signal_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*os.Process), args[1].(os.Signal))
	})
	return _c
}

func (_c *ManagerMock_Signal_Call) Return(_a0 error) *ManagerMock_Signal_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ManagerMock_Signal_Call) RunAndReturn(run func(*os.Process, os.Signal) error) *ManagerMock_Signal_Call {
	_c.Call.Return(run)
	return _c
}

// ThreadID provides a mock function with no fields
func (_m *ManagerMock) ThreadID() int {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ThreadID")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// ManagerMock_ThreadID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ThreadID'
type ManagerMock_ThreadID_Call struct {
	*mock.Call
}

// ThreadID is a helper method to define mock.On call
func (_e *ManagerMock_Expecter) ThreadID() *ManagerMock_ThreadID_Call {
	return &ManagerMock_ThreadID_Call{Call: _e.mock.On("ThreadID")}
}

func (_c *ManagerMock_ThreadID_Call) Run(run func()) *ManagerMock_ThreadID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ManagerMock_ThreadID_Call) Return(_a0 int) *ManagerMock_ThreadID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ManagerMock_ThreadID_Call) RunAndReturn(run func() int) *ManagerMock_ThreadID_Call {
	_c.Call.Return(run)
	return _c
}

// NewManagerMock creates a new instance of ManagerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewManagerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ManagerMock {
	mock := &ManagerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package network

import mock "github.com/stretchr/testify/mock"

// tcExecuterMock is an autogenerated mock type for the tcExecuter type
type tcExecuterMock struct {
	mock.Mock
}

type tcExecuterMock_Expecter struct {
	mock *mock.Mock
}

func (_m *tcExecuterMock) EXPECT() *tcExecuterMock_Expecter {
	return &tcExecuterMock_Expecter{mock: &_m.Mock}
}

// Run provides a mock function with given fields: args
func (_m *tcExecuterMock) Run(args []string) (int, string, error) {
	ret := _m.Called(args)

	var r0 int
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func([]string) (int, string, error)); ok {
		return rf(args)
	}
	if rf, ok := ret.Get(0).(func([]string) int); ok {
		r0 = rf(args)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func([]string) string); ok {
		r1 = rf(args)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func([]string) error); ok {
		r2 = rf(args)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// tcExecuterMock_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type tcExecuterMock_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - args []string
func (_e *tcExecuterMock_Expecter) Run(args interface{}) *tcExecuterMock_Run_Call {
	return &tcExecuterMock_Run_Call{Call: _e.mock.On("Run", args)}
}

func (_c *tcExecuterMock_Run_Call) Run(run func(args []string)) *tcExecuterMock_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]string))
	})
	return _c
}

func (_c *tcExecuterMock_Run_Call) Return(exitCode int, stdout string, stderr error) *tcExecuterMock_Run_Call {
	_c.Call.Return(exitCode, stdout, stderr)
	return _c
}

func (_c *tcExecuterMock_Run_Call) RunAndReturn(run func([]string) (int, string, error)) *tcExecuterMock_Run_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewTcExecuterMock interface {
	mock.TestingT
	Cleanup(func())
}

// newTcExecuterMock creates a new instance of tcExecuterMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newTcExecuterMock(t mockConstructorTestingTnewTcExecuterMock) *tcExecuterMock {
	mock := &tcExecuterMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

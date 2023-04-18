// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package mocks

import mock "github.com/stretchr/testify/mock"

// BPFDiskFailureCommandMock is an autogenerated mock type for the BPFDiskFailureCommand type
type BPFDiskFailureCommandMock struct {
	mock.Mock
}

type BPFDiskFailureCommandMock_Expecter struct {
	mock *mock.Mock
}

func (_m *BPFDiskFailureCommandMock) EXPECT() *BPFDiskFailureCommandMock_Expecter {
	return &BPFDiskFailureCommandMock_Expecter{mock: &_m.Mock}
}

// Run provides a mock function with given fields: pid, path
func (_m *BPFDiskFailureCommandMock) Run(pid int, path string) error {
	ret := _m.Called(pid, path)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, string) error); ok {
		r0 = rf(pid, path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// BPFDiskFailureCommandMock_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type BPFDiskFailureCommandMock_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - pid int
//   - path string
func (_e *BPFDiskFailureCommandMock_Expecter) Run(pid interface{}, path interface{}) *BPFDiskFailureCommandMock_Run_Call {
	return &BPFDiskFailureCommandMock_Run_Call{Call: _e.mock.On("Run", pid, path)}
}

func (_c *BPFDiskFailureCommandMock_Run_Call) Run(run func(pid int, path string)) *BPFDiskFailureCommandMock_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int), args[1].(string))
	})
	return _c
}

func (_c *BPFDiskFailureCommandMock_Run_Call) Return(_a0 error) *BPFDiskFailureCommandMock_Run_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *BPFDiskFailureCommandMock_Run_Call) RunAndReturn(run func(int, string) error) *BPFDiskFailureCommandMock_Run_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewBPFDiskFailureCommandMock interface {
	mock.TestingT
	Cleanup(func())
}

// NewBPFDiskFailureCommandMock creates a new instance of BPFDiskFailureCommandMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBPFDiskFailureCommandMock(t mockConstructorTestingTNewBPFDiskFailureCommandMock) *BPFDiskFailureCommandMock {
	mock := &BPFDiskFailureCommandMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

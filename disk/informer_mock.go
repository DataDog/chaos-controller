// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package disk

import mock "github.com/stretchr/testify/mock"

// InformerMock is an autogenerated mock type for the Informer type
type InformerMock struct {
	mock.Mock
}

type InformerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *InformerMock) EXPECT() *InformerMock_Expecter {
	return &InformerMock_Expecter{mock: &_m.Mock}
}

// Major provides a mock function with no fields
func (_m *InformerMock) Major() int {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Major")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// InformerMock_Major_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Major'
type InformerMock_Major_Call struct {
	*mock.Call
}

// Major is a helper method to define mock.On call
func (_e *InformerMock_Expecter) Major() *InformerMock_Major_Call {
	return &InformerMock_Major_Call{Call: _e.mock.On("Major")}
}

func (_c *InformerMock_Major_Call) Run(run func()) *InformerMock_Major_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InformerMock_Major_Call) Return(_a0 int) *InformerMock_Major_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InformerMock_Major_Call) RunAndReturn(run func() int) *InformerMock_Major_Call {
	_c.Call.Return(run)
	return _c
}

// Source provides a mock function with no fields
func (_m *InformerMock) Source() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Source")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// InformerMock_Source_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Source'
type InformerMock_Source_Call struct {
	*mock.Call
}

// Source is a helper method to define mock.On call
func (_e *InformerMock_Expecter) Source() *InformerMock_Source_Call {
	return &InformerMock_Source_Call{Call: _e.mock.On("Source")}
}

func (_c *InformerMock_Source_Call) Run(run func()) *InformerMock_Source_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InformerMock_Source_Call) Return(_a0 string) *InformerMock_Source_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InformerMock_Source_Call) RunAndReturn(run func() string) *InformerMock_Source_Call {
	_c.Call.Return(run)
	return _c
}

// NewInformerMock creates a new instance of InformerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInformerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *InformerMock {
	mock := &InformerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

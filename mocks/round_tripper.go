// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package mocks

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
)

// RoundTripperMock is an autogenerated mock type for the RoundTripper type
type RoundTripperMock struct {
	mock.Mock
}

type RoundTripperMock_Expecter struct {
	mock *mock.Mock
}

func (_m *RoundTripperMock) EXPECT() *RoundTripperMock_Expecter {
	return &RoundTripperMock_Expecter{mock: &_m.Mock}
}

// RoundTrip provides a mock function with given fields: _a0
func (_m *RoundTripperMock) RoundTrip(_a0 *http.Request) (*http.Response, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for RoundTrip")
	}

	var r0 *http.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(*http.Request) (*http.Response, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*http.Request) *http.Response); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Response)
		}
	}

	if rf, ok := ret.Get(1).(func(*http.Request) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RoundTripperMock_RoundTrip_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RoundTrip'
type RoundTripperMock_RoundTrip_Call struct {
	*mock.Call
}

// RoundTrip is a helper method to define mock.On call
//   - _a0 *http.Request
func (_e *RoundTripperMock_Expecter) RoundTrip(_a0 interface{}) *RoundTripperMock_RoundTrip_Call {
	return &RoundTripperMock_RoundTrip_Call{Call: _e.mock.On("RoundTrip", _a0)}
}

func (_c *RoundTripperMock_RoundTrip_Call) Run(run func(_a0 *http.Request)) *RoundTripperMock_RoundTrip_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*http.Request))
	})
	return _c
}

func (_c *RoundTripperMock_RoundTrip_Call) Return(_a0 *http.Response, _a1 error) *RoundTripperMock_RoundTrip_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *RoundTripperMock_RoundTrip_Call) RunAndReturn(run func(*http.Request) (*http.Response, error)) *RoundTripperMock_RoundTrip_Call {
	_c.Call.Return(run)
	return _c
}

// NewRoundTripperMock creates a new instance of RoundTripperMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRoundTripperMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *RoundTripperMock {
	mock := &RoundTripperMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

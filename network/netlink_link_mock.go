// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package network

import mock "github.com/stretchr/testify/mock"

// NetlinkLinkMock is an autogenerated mock type for the NetlinkLink type
type NetlinkLinkMock struct {
	mock.Mock
}

type NetlinkLinkMock_Expecter struct {
	mock *mock.Mock
}

func (_m *NetlinkLinkMock) EXPECT() *NetlinkLinkMock_Expecter {
	return &NetlinkLinkMock_Expecter{mock: &_m.Mock}
}

// Name provides a mock function with given fields:
func (_m *NetlinkLinkMock) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NetlinkLinkMock_Name_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Name'
type NetlinkLinkMock_Name_Call struct {
	*mock.Call
}

// Name is a helper method to define mock.On call
func (_e *NetlinkLinkMock_Expecter) Name() *NetlinkLinkMock_Name_Call {
	return &NetlinkLinkMock_Name_Call{Call: _e.mock.On("Name")}
}

func (_c *NetlinkLinkMock_Name_Call) Run(run func()) *NetlinkLinkMock_Name_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetlinkLinkMock_Name_Call) Return(_a0 string) *NetlinkLinkMock_Name_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetlinkLinkMock_Name_Call) RunAndReturn(run func() string) *NetlinkLinkMock_Name_Call {
	_c.Call.Return(run)
	return _c
}

// SetTxQLen provides a mock function with given fields: qlen
func (_m *NetlinkLinkMock) SetTxQLen(qlen int) error {
	ret := _m.Called(qlen)

	var r0 error
	if rf, ok := ret.Get(0).(func(int) error); ok {
		r0 = rf(qlen)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NetlinkLinkMock_SetTxQLen_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetTxQLen'
type NetlinkLinkMock_SetTxQLen_Call struct {
	*mock.Call
}

// SetTxQLen is a helper method to define mock.On call
//   - qlen int
func (_e *NetlinkLinkMock_Expecter) SetTxQLen(qlen interface{}) *NetlinkLinkMock_SetTxQLen_Call {
	return &NetlinkLinkMock_SetTxQLen_Call{Call: _e.mock.On("SetTxQLen", qlen)}
}

func (_c *NetlinkLinkMock_SetTxQLen_Call) Run(run func(qlen int)) *NetlinkLinkMock_SetTxQLen_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *NetlinkLinkMock_SetTxQLen_Call) Return(_a0 error) *NetlinkLinkMock_SetTxQLen_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetlinkLinkMock_SetTxQLen_Call) RunAndReturn(run func(int) error) *NetlinkLinkMock_SetTxQLen_Call {
	_c.Call.Return(run)
	return _c
}

// TxQLen provides a mock function with given fields:
func (_m *NetlinkLinkMock) TxQLen() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// NetlinkLinkMock_TxQLen_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'TxQLen'
type NetlinkLinkMock_TxQLen_Call struct {
	*mock.Call
}

// TxQLen is a helper method to define mock.On call
func (_e *NetlinkLinkMock_Expecter) TxQLen() *NetlinkLinkMock_TxQLen_Call {
	return &NetlinkLinkMock_TxQLen_Call{Call: _e.mock.On("TxQLen")}
}

func (_c *NetlinkLinkMock_TxQLen_Call) Run(run func()) *NetlinkLinkMock_TxQLen_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetlinkLinkMock_TxQLen_Call) Return(_a0 int) *NetlinkLinkMock_TxQLen_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetlinkLinkMock_TxQLen_Call) RunAndReturn(run func() int) *NetlinkLinkMock_TxQLen_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewNetlinkLinkMock interface {
	mock.TestingT
	Cleanup(func())
}

// NewNetlinkLinkMock creates a new instance of NetlinkLinkMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNetlinkLinkMock(t mockConstructorTestingTNewNetlinkLinkMock) *NetlinkLinkMock {
	mock := &NetlinkLinkMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

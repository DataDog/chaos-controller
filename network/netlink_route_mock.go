// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package network

import (
	net "net"

	mock "github.com/stretchr/testify/mock"
)

// NetlinkRouteMock is an autogenerated mock type for the NetlinkRoute type
type NetlinkRouteMock struct {
	mock.Mock
}

type NetlinkRouteMock_Expecter struct {
	mock *mock.Mock
}

func (_m *NetlinkRouteMock) EXPECT() *NetlinkRouteMock_Expecter {
	return &NetlinkRouteMock_Expecter{mock: &_m.Mock}
}

// Gateway provides a mock function with given fields:
func (_m *NetlinkRouteMock) Gateway() net.IP {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Gateway")
	}

	var r0 net.IP
	if rf, ok := ret.Get(0).(func() net.IP); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(net.IP)
		}
	}

	return r0
}

// NetlinkRouteMock_Gateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Gateway'
type NetlinkRouteMock_Gateway_Call struct {
	*mock.Call
}

// Gateway is a helper method to define mock.On call
func (_e *NetlinkRouteMock_Expecter) Gateway() *NetlinkRouteMock_Gateway_Call {
	return &NetlinkRouteMock_Gateway_Call{Call: _e.mock.On("Gateway")}
}

func (_c *NetlinkRouteMock_Gateway_Call) Run(run func()) *NetlinkRouteMock_Gateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetlinkRouteMock_Gateway_Call) Return(_a0 net.IP) *NetlinkRouteMock_Gateway_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetlinkRouteMock_Gateway_Call) RunAndReturn(run func() net.IP) *NetlinkRouteMock_Gateway_Call {
	_c.Call.Return(run)
	return _c
}

// Link provides a mock function with given fields:
func (_m *NetlinkRouteMock) Link() NetlinkLink {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Link")
	}

	var r0 NetlinkLink
	if rf, ok := ret.Get(0).(func() NetlinkLink); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(NetlinkLink)
		}
	}

	return r0
}

// NetlinkRouteMock_Link_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Link'
type NetlinkRouteMock_Link_Call struct {
	*mock.Call
}

// Link is a helper method to define mock.On call
func (_e *NetlinkRouteMock_Expecter) Link() *NetlinkRouteMock_Link_Call {
	return &NetlinkRouteMock_Link_Call{Call: _e.mock.On("Link")}
}

func (_c *NetlinkRouteMock_Link_Call) Run(run func()) *NetlinkRouteMock_Link_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NetlinkRouteMock_Link_Call) Return(_a0 NetlinkLink) *NetlinkRouteMock_Link_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *NetlinkRouteMock_Link_Call) RunAndReturn(run func() NetlinkLink) *NetlinkRouteMock_Link_Call {
	_c.Call.Return(run)
	return _c
}

// NewNetlinkRouteMock creates a new instance of NetlinkRouteMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewNetlinkRouteMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *NetlinkRouteMock {
	mock := &NetlinkRouteMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

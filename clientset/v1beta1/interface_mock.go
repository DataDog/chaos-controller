// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package v1beta1

import (
	internalversion "github.com/DataDog/chaos-controller/clientset/v1beta1/typed/v1beta1/internalversion"
	mock "github.com/stretchr/testify/mock"
	discovery "k8s.io/client-go/discovery"
)

// InterfaceMock is an autogenerated mock type for the Interface type
type InterfaceMock struct {
	mock.Mock
}

type InterfaceMock_Expecter struct {
	mock *mock.Mock
}

func (_m *InterfaceMock) EXPECT() *InterfaceMock_Expecter {
	return &InterfaceMock_Expecter{mock: &_m.Mock}
}

// Chaos provides a mock function with given fields:
func (_m *InterfaceMock) Chaos() internalversion.ChaosInterface {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Chaos")
	}

	var r0 internalversion.ChaosInterface
	if rf, ok := ret.Get(0).(func() internalversion.ChaosInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internalversion.ChaosInterface)
		}
	}

	return r0
}

// InterfaceMock_Chaos_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Chaos'
type InterfaceMock_Chaos_Call struct {
	*mock.Call
}

// Chaos is a helper method to define mock.On call
func (_e *InterfaceMock_Expecter) Chaos() *InterfaceMock_Chaos_Call {
	return &InterfaceMock_Chaos_Call{Call: _e.mock.On("Chaos")}
}

func (_c *InterfaceMock_Chaos_Call) Run(run func()) *InterfaceMock_Chaos_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InterfaceMock_Chaos_Call) Return(_a0 internalversion.ChaosInterface) *InterfaceMock_Chaos_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InterfaceMock_Chaos_Call) RunAndReturn(run func() internalversion.ChaosInterface) *InterfaceMock_Chaos_Call {
	_c.Call.Return(run)
	return _c
}

// Discovery provides a mock function with given fields:
func (_m *InterfaceMock) Discovery() discovery.DiscoveryInterface {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Discovery")
	}

	var r0 discovery.DiscoveryInterface
	if rf, ok := ret.Get(0).(func() discovery.DiscoveryInterface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(discovery.DiscoveryInterface)
		}
	}

	return r0
}

// InterfaceMock_Discovery_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Discovery'
type InterfaceMock_Discovery_Call struct {
	*mock.Call
}

// Discovery is a helper method to define mock.On call
func (_e *InterfaceMock_Expecter) Discovery() *InterfaceMock_Discovery_Call {
	return &InterfaceMock_Discovery_Call{Call: _e.mock.On("Discovery")}
}

func (_c *InterfaceMock_Discovery_Call) Run(run func()) *InterfaceMock_Discovery_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InterfaceMock_Discovery_Call) Return(_a0 discovery.DiscoveryInterface) *InterfaceMock_Discovery_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InterfaceMock_Discovery_Call) RunAndReturn(run func() discovery.DiscoveryInterface) *InterfaceMock_Discovery_Call {
	_c.Call.Return(run)
	return _c
}

// NewInterfaceMock creates a new instance of InterfaceMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInterfaceMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *InterfaceMock {
	mock := &InterfaceMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

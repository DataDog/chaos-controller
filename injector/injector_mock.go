// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.
package injector

import (
	types "github.com/DataDog/chaos-controller/types"
	mock "github.com/stretchr/testify/mock"
)

// InjectorMock is an autogenerated mock type for the Injector type
type InjectorMock struct {
	mock.Mock
}

type InjectorMock_Expecter struct {
	mock *mock.Mock
}

func (_m *InjectorMock) EXPECT() *InjectorMock_Expecter {
	return &InjectorMock_Expecter{mock: &_m.Mock}
}

// Clean provides a mock function with given fields:
func (_m *InjectorMock) Clean() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Clean")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InjectorMock_Clean_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Clean'
type InjectorMock_Clean_Call struct {
	*mock.Call
}

// Clean is a helper method to define mock.On call
func (_e *InjectorMock_Expecter) Clean() *InjectorMock_Clean_Call {
	return &InjectorMock_Clean_Call{Call: _e.mock.On("Clean")}
}

func (_c *InjectorMock_Clean_Call) Run(run func()) *InjectorMock_Clean_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InjectorMock_Clean_Call) Return(_a0 error) *InjectorMock_Clean_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InjectorMock_Clean_Call) RunAndReturn(run func() error) *InjectorMock_Clean_Call {
	_c.Call.Return(run)
	return _c
}

// GetDisruptionKind provides a mock function with given fields:
func (_m *InjectorMock) GetDisruptionKind() types.DisruptionKindName {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDisruptionKind")
	}

	var r0 types.DisruptionKindName
	if rf, ok := ret.Get(0).(func() types.DisruptionKindName); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(types.DisruptionKindName)
	}

	return r0
}

// InjectorMock_GetDisruptionKind_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetDisruptionKind'
type InjectorMock_GetDisruptionKind_Call struct {
	*mock.Call
}

// GetDisruptionKind is a helper method to define mock.On call
func (_e *InjectorMock_Expecter) GetDisruptionKind() *InjectorMock_GetDisruptionKind_Call {
	return &InjectorMock_GetDisruptionKind_Call{Call: _e.mock.On("GetDisruptionKind")}
}

func (_c *InjectorMock_GetDisruptionKind_Call) Run(run func()) *InjectorMock_GetDisruptionKind_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InjectorMock_GetDisruptionKind_Call) Return(_a0 types.DisruptionKindName) *InjectorMock_GetDisruptionKind_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InjectorMock_GetDisruptionKind_Call) RunAndReturn(run func() types.DisruptionKindName) *InjectorMock_GetDisruptionKind_Call {
	_c.Call.Return(run)
	return _c
}

// Inject provides a mock function with given fields:
func (_m *InjectorMock) Inject() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Inject")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InjectorMock_Inject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Inject'
type InjectorMock_Inject_Call struct {
	*mock.Call
}

// Inject is a helper method to define mock.On call
func (_e *InjectorMock_Expecter) Inject() *InjectorMock_Inject_Call {
	return &InjectorMock_Inject_Call{Call: _e.mock.On("Inject")}
}

func (_c *InjectorMock_Inject_Call) Run(run func()) *InjectorMock_Inject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InjectorMock_Inject_Call) Return(_a0 error) *InjectorMock_Inject_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InjectorMock_Inject_Call) RunAndReturn(run func() error) *InjectorMock_Inject_Call {
	_c.Call.Return(run)
	return _c
}

// TargetName provides a mock function with given fields:
func (_m *InjectorMock) TargetName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for TargetName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// InjectorMock_TargetName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'TargetName'
type InjectorMock_TargetName_Call struct {
	*mock.Call
}

// TargetName is a helper method to define mock.On call
func (_e *InjectorMock_Expecter) TargetName() *InjectorMock_TargetName_Call {
	return &InjectorMock_TargetName_Call{Call: _e.mock.On("TargetName")}
}

func (_c *InjectorMock_TargetName_Call) Run(run func()) *InjectorMock_TargetName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *InjectorMock_TargetName_Call) Return(_a0 string) *InjectorMock_TargetName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *InjectorMock_TargetName_Call) RunAndReturn(run func() string) *InjectorMock_TargetName_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateConfig provides a mock function with given fields: config
func (_m *InjectorMock) UpdateConfig(config Config) {
	_m.Called(config)
}

// InjectorMock_UpdateConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateConfig'
type InjectorMock_UpdateConfig_Call struct {
	*mock.Call
}

// UpdateConfig is a helper method to define mock.On call
//   - config Config
func (_e *InjectorMock_Expecter) UpdateConfig(config interface{}) *InjectorMock_UpdateConfig_Call {
	return &InjectorMock_UpdateConfig_Call{Call: _e.mock.On("UpdateConfig", config)}
}

func (_c *InjectorMock_UpdateConfig_Call) Run(run func(config Config)) *InjectorMock_UpdateConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(Config))
	})
	return _c
}

func (_c *InjectorMock_UpdateConfig_Call) Return() *InjectorMock_UpdateConfig_Call {
	_c.Call.Return()
	return _c
}

func (_c *InjectorMock_UpdateConfig_Call) RunAndReturn(run func(Config)) *InjectorMock_UpdateConfig_Call {
	_c.Call.Return(run)
	return _c
}

// NewInjectorMock creates a new instance of InjectorMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInjectorMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *InjectorMock {
	mock := &InjectorMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

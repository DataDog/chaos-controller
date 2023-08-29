// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers

import (
	mock "github.com/stretchr/testify/mock"
	source "sigs.k8s.io/controller-runtime/pkg/source"
)

// WatcherMock is an autogenerated mock type for the Watcher type
type WatcherMock struct {
	mock.Mock
}

type WatcherMock_Expecter struct {
	mock *mock.Mock
}

func (_m *WatcherMock) EXPECT() *WatcherMock_Expecter {
	return &WatcherMock_Expecter{mock: &_m.Mock}
}

// Clean provides a mock function with given fields:
func (_m *WatcherMock) Clean() {
	_m.Called()
}

// WatcherMock_Clean_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Clean'
type WatcherMock_Clean_Call struct {
	*mock.Call
}

// Clean is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) Clean() *WatcherMock_Clean_Call {
	return &WatcherMock_Clean_Call{Call: _e.mock.On("Clean")}
}

func (_c *WatcherMock_Clean_Call) Run(run func()) *WatcherMock_Clean_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_Clean_Call) Return() *WatcherMock_Clean_Call {
	_c.Call.Return()
	return _c
}

func (_c *WatcherMock_Clean_Call) RunAndReturn(run func()) *WatcherMock_Clean_Call {
	_c.Call.Return(run)
	return _c
}

// GetCacheSource provides a mock function with given fields:
func (_m *WatcherMock) GetCacheSource() (source.SyncingSource, error) {
	ret := _m.Called()

	var r0 source.SyncingSource
	var r1 error
	if rf, ok := ret.Get(0).(func() (source.SyncingSource, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() source.SyncingSource); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(source.SyncingSource)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WatcherMock_GetCacheSource_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCacheSource'
type WatcherMock_GetCacheSource_Call struct {
	*mock.Call
}

// GetCacheSource is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) GetCacheSource() *WatcherMock_GetCacheSource_Call {
	return &WatcherMock_GetCacheSource_Call{Call: _e.mock.On("GetCacheSource")}
}

func (_c *WatcherMock_GetCacheSource_Call) Run(run func()) *WatcherMock_GetCacheSource_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_GetCacheSource_Call) Return(_a0 source.SyncingSource, _a1 error) *WatcherMock_GetCacheSource_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *WatcherMock_GetCacheSource_Call) RunAndReturn(run func() (source.SyncingSource, error)) *WatcherMock_GetCacheSource_Call {
	_c.Call.Return(run)
	return _c
}

// GetConfig provides a mock function with given fields:
func (_m *WatcherMock) GetConfig() WatcherConfig {
	ret := _m.Called()

	var r0 WatcherConfig
	if rf, ok := ret.Get(0).(func() WatcherConfig); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(WatcherConfig)
	}

	return r0
}

// WatcherMock_GetConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConfig'
type WatcherMock_GetConfig_Call struct {
	*mock.Call
}

// GetConfig is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) GetConfig() *WatcherMock_GetConfig_Call {
	return &WatcherMock_GetConfig_Call{Call: _e.mock.On("GetConfig")}
}

func (_c *WatcherMock_GetConfig_Call) Run(run func()) *WatcherMock_GetConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_GetConfig_Call) Return(_a0 WatcherConfig) *WatcherMock_GetConfig_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WatcherMock_GetConfig_Call) RunAndReturn(run func() WatcherConfig) *WatcherMock_GetConfig_Call {
	_c.Call.Return(run)
	return _c
}

// GetContextTuple provides a mock function with given fields:
func (_m *WatcherMock) GetContextTuple() (CtxTuple, error) {
	ret := _m.Called()

	var r0 CtxTuple
	var r1 error
	if rf, ok := ret.Get(0).(func() (CtxTuple, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() CtxTuple); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(CtxTuple)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WatcherMock_GetContextTuple_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetContextTuple'
type WatcherMock_GetContextTuple_Call struct {
	*mock.Call
}

// GetContextTuple is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) GetContextTuple() *WatcherMock_GetContextTuple_Call {
	return &WatcherMock_GetContextTuple_Call{Call: _e.mock.On("GetContextTuple")}
}

func (_c *WatcherMock_GetContextTuple_Call) Run(run func()) *WatcherMock_GetContextTuple_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_GetContextTuple_Call) Return(_a0 CtxTuple, _a1 error) *WatcherMock_GetContextTuple_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *WatcherMock_GetContextTuple_Call) RunAndReturn(run func() (CtxTuple, error)) *WatcherMock_GetContextTuple_Call {
	_c.Call.Return(run)
	return _c
}

// GetName provides a mock function with given fields:
func (_m *WatcherMock) GetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// WatcherMock_GetName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetName'
type WatcherMock_GetName_Call struct {
	*mock.Call
}

// GetName is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) GetName() *WatcherMock_GetName_Call {
	return &WatcherMock_GetName_Call{Call: _e.mock.On("GetName")}
}

func (_c *WatcherMock_GetName_Call) Run(run func()) *WatcherMock_GetName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_GetName_Call) Return(_a0 string) *WatcherMock_GetName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WatcherMock_GetName_Call) RunAndReturn(run func() string) *WatcherMock_GetName_Call {
	_c.Call.Return(run)
	return _c
}

// IsExpired provides a mock function with given fields:
func (_m *WatcherMock) IsExpired() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// WatcherMock_IsExpired_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsExpired'
type WatcherMock_IsExpired_Call struct {
	*mock.Call
}

// IsExpired is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) IsExpired() *WatcherMock_IsExpired_Call {
	return &WatcherMock_IsExpired_Call{Call: _e.mock.On("IsExpired")}
}

func (_c *WatcherMock_IsExpired_Call) Run(run func()) *WatcherMock_IsExpired_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_IsExpired_Call) Return(_a0 bool) *WatcherMock_IsExpired_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WatcherMock_IsExpired_Call) RunAndReturn(run func() bool) *WatcherMock_IsExpired_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with given fields:
func (_m *WatcherMock) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WatcherMock_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type WatcherMock_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
func (_e *WatcherMock_Expecter) Start() *WatcherMock_Start_Call {
	return &WatcherMock_Start_Call{Call: _e.mock.On("Start")}
}

func (_c *WatcherMock_Start_Call) Run(run func()) *WatcherMock_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *WatcherMock_Start_Call) Return(_a0 error) *WatcherMock_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *WatcherMock_Start_Call) RunAndReturn(run func() error) *WatcherMock_Start_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewWatcherMock interface {
	mock.TestingT
	Cleanup(func())
}

// NewWatcherMock creates a new instance of WatcherMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewWatcherMock(t mockConstructorTestingTNewWatcherMock) *WatcherMock {
	mock := &WatcherMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

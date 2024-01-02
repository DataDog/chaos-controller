// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package mocks

import (
	mock "github.com/stretchr/testify/mock"
	cache "k8s.io/client-go/tools/cache"

	time "time"
)

// CacheInformerMock is an autogenerated mock type for the Informer type
type CacheInformerMock struct {
	mock.Mock
}

type CacheInformerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *CacheInformerMock) EXPECT() *CacheInformerMock_Expecter {
	return &CacheInformerMock_Expecter{mock: &_m.Mock}
}

// AddEventHandler provides a mock function with given fields: handler
func (_m *CacheInformerMock) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	ret := _m.Called(handler)

	if len(ret) == 0 {
		panic("no return value specified for AddEventHandler")
	}

	var r0 cache.ResourceEventHandlerRegistration
	var r1 error
	if rf, ok := ret.Get(0).(func(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)); ok {
		return rf(handler)
	}
	if rf, ok := ret.Get(0).(func(cache.ResourceEventHandler) cache.ResourceEventHandlerRegistration); ok {
		r0 = rf(handler)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cache.ResourceEventHandlerRegistration)
		}
	}

	if rf, ok := ret.Get(1).(func(cache.ResourceEventHandler) error); ok {
		r1 = rf(handler)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CacheInformerMock_AddEventHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddEventHandler'
type CacheInformerMock_AddEventHandler_Call struct {
	*mock.Call
}

// AddEventHandler is a helper method to define mock.On call
//   - handler cache.ResourceEventHandler
func (_e *CacheInformerMock_Expecter) AddEventHandler(handler interface{}) *CacheInformerMock_AddEventHandler_Call {
	return &CacheInformerMock_AddEventHandler_Call{Call: _e.mock.On("AddEventHandler", handler)}
}

func (_c *CacheInformerMock_AddEventHandler_Call) Run(run func(handler cache.ResourceEventHandler)) *CacheInformerMock_AddEventHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(cache.ResourceEventHandler))
	})
	return _c
}

func (_c *CacheInformerMock_AddEventHandler_Call) Return(_a0 cache.ResourceEventHandlerRegistration, _a1 error) *CacheInformerMock_AddEventHandler_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CacheInformerMock_AddEventHandler_Call) RunAndReturn(run func(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)) *CacheInformerMock_AddEventHandler_Call {
	_c.Call.Return(run)
	return _c
}

// AddEventHandlerWithResyncPeriod provides a mock function with given fields: handler, resyncPeriod
func (_m *CacheInformerMock) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	ret := _m.Called(handler, resyncPeriod)

	if len(ret) == 0 {
		panic("no return value specified for AddEventHandlerWithResyncPeriod")
	}

	var r0 cache.ResourceEventHandlerRegistration
	var r1 error
	if rf, ok := ret.Get(0).(func(cache.ResourceEventHandler, time.Duration) (cache.ResourceEventHandlerRegistration, error)); ok {
		return rf(handler, resyncPeriod)
	}
	if rf, ok := ret.Get(0).(func(cache.ResourceEventHandler, time.Duration) cache.ResourceEventHandlerRegistration); ok {
		r0 = rf(handler, resyncPeriod)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(cache.ResourceEventHandlerRegistration)
		}
	}

	if rf, ok := ret.Get(1).(func(cache.ResourceEventHandler, time.Duration) error); ok {
		r1 = rf(handler, resyncPeriod)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CacheInformerMock_AddEventHandlerWithResyncPeriod_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddEventHandlerWithResyncPeriod'
type CacheInformerMock_AddEventHandlerWithResyncPeriod_Call struct {
	*mock.Call
}

// AddEventHandlerWithResyncPeriod is a helper method to define mock.On call
//   - handler cache.ResourceEventHandler
//   - resyncPeriod time.Duration
func (_e *CacheInformerMock_Expecter) AddEventHandlerWithResyncPeriod(handler interface{}, resyncPeriod interface{}) *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call {
	return &CacheInformerMock_AddEventHandlerWithResyncPeriod_Call{Call: _e.mock.On("AddEventHandlerWithResyncPeriod", handler, resyncPeriod)}
}

func (_c *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call) Run(run func(handler cache.ResourceEventHandler, resyncPeriod time.Duration)) *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(cache.ResourceEventHandler), args[1].(time.Duration))
	})
	return _c
}

func (_c *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call) Return(_a0 cache.ResourceEventHandlerRegistration, _a1 error) *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call) RunAndReturn(run func(cache.ResourceEventHandler, time.Duration) (cache.ResourceEventHandlerRegistration, error)) *CacheInformerMock_AddEventHandlerWithResyncPeriod_Call {
	_c.Call.Return(run)
	return _c
}

// AddIndexers provides a mock function with given fields: indexers
func (_m *CacheInformerMock) AddIndexers(indexers cache.Indexers) error {
	ret := _m.Called(indexers)

	if len(ret) == 0 {
		panic("no return value specified for AddIndexers")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(cache.Indexers) error); ok {
		r0 = rf(indexers)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CacheInformerMock_AddIndexers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddIndexers'
type CacheInformerMock_AddIndexers_Call struct {
	*mock.Call
}

// AddIndexers is a helper method to define mock.On call
//   - indexers cache.Indexers
func (_e *CacheInformerMock_Expecter) AddIndexers(indexers interface{}) *CacheInformerMock_AddIndexers_Call {
	return &CacheInformerMock_AddIndexers_Call{Call: _e.mock.On("AddIndexers", indexers)}
}

func (_c *CacheInformerMock_AddIndexers_Call) Run(run func(indexers cache.Indexers)) *CacheInformerMock_AddIndexers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(cache.Indexers))
	})
	return _c
}

func (_c *CacheInformerMock_AddIndexers_Call) Return(_a0 error) *CacheInformerMock_AddIndexers_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *CacheInformerMock_AddIndexers_Call) RunAndReturn(run func(cache.Indexers) error) *CacheInformerMock_AddIndexers_Call {
	_c.Call.Return(run)
	return _c
}

// HasSynced provides a mock function with given fields:
func (_m *CacheInformerMock) HasSynced() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for HasSynced")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// CacheInformerMock_HasSynced_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HasSynced'
type CacheInformerMock_HasSynced_Call struct {
	*mock.Call
}

// HasSynced is a helper method to define mock.On call
func (_e *CacheInformerMock_Expecter) HasSynced() *CacheInformerMock_HasSynced_Call {
	return &CacheInformerMock_HasSynced_Call{Call: _e.mock.On("HasSynced")}
}

func (_c *CacheInformerMock_HasSynced_Call) Run(run func()) *CacheInformerMock_HasSynced_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *CacheInformerMock_HasSynced_Call) Return(_a0 bool) *CacheInformerMock_HasSynced_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *CacheInformerMock_HasSynced_Call) RunAndReturn(run func() bool) *CacheInformerMock_HasSynced_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveEventHandler provides a mock function with given fields: handle
func (_m *CacheInformerMock) RemoveEventHandler(handle cache.ResourceEventHandlerRegistration) error {
	ret := _m.Called(handle)

	if len(ret) == 0 {
		panic("no return value specified for RemoveEventHandler")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(cache.ResourceEventHandlerRegistration) error); ok {
		r0 = rf(handle)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CacheInformerMock_RemoveEventHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveEventHandler'
type CacheInformerMock_RemoveEventHandler_Call struct {
	*mock.Call
}

// RemoveEventHandler is a helper method to define mock.On call
//   - handle cache.ResourceEventHandlerRegistration
func (_e *CacheInformerMock_Expecter) RemoveEventHandler(handle interface{}) *CacheInformerMock_RemoveEventHandler_Call {
	return &CacheInformerMock_RemoveEventHandler_Call{Call: _e.mock.On("RemoveEventHandler", handle)}
}

func (_c *CacheInformerMock_RemoveEventHandler_Call) Run(run func(handle cache.ResourceEventHandlerRegistration)) *CacheInformerMock_RemoveEventHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(cache.ResourceEventHandlerRegistration))
	})
	return _c
}

func (_c *CacheInformerMock_RemoveEventHandler_Call) Return(_a0 error) *CacheInformerMock_RemoveEventHandler_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *CacheInformerMock_RemoveEventHandler_Call) RunAndReturn(run func(cache.ResourceEventHandlerRegistration) error) *CacheInformerMock_RemoveEventHandler_Call {
	_c.Call.Return(run)
	return _c
}

// NewCacheInformerMock creates a new instance of CacheInformerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCacheInformerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *CacheInformerMock {
	mock := &CacheInformerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

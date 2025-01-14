// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.
package watchers

import (
	mock "github.com/stretchr/testify/mock"
	cache "sigs.k8s.io/controller-runtime/pkg/cache"

	v1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
)

// DisruptionsWatchersManagerMock is an autogenerated mock type for the DisruptionsWatchersManager type
type DisruptionsWatchersManagerMock struct {
	mock.Mock
}

type DisruptionsWatchersManagerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *DisruptionsWatchersManagerMock) EXPECT() *DisruptionsWatchersManagerMock_Expecter {
	return &DisruptionsWatchersManagerMock_Expecter{mock: &_m.Mock}
}

// CreateAllWatchers provides a mock function with given fields: disruption, watcherManagerMock, cacheMock
func (_m *DisruptionsWatchersManagerMock) CreateAllWatchers(disruption *v1beta1.Disruption, watcherManagerMock Manager, cacheMock cache.Cache) error {
	ret := _m.Called(disruption, watcherManagerMock, cacheMock)

	if len(ret) == 0 {
		panic("no return value specified for CreateAllWatchers")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*v1beta1.Disruption, Manager, cache.Cache) error); ok {
		r0 = rf(disruption, watcherManagerMock, cacheMock)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DisruptionsWatchersManagerMock_CreateAllWatchers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateAllWatchers'
type DisruptionsWatchersManagerMock_CreateAllWatchers_Call struct {
	*mock.Call
}

// CreateAllWatchers is a helper method to define mock.On call
//   - disruption *v1beta1.Disruption
//   - watcherManagerMock Manager
//   - cacheMock cache.Cache
func (_e *DisruptionsWatchersManagerMock_Expecter) CreateAllWatchers(disruption interface{}, watcherManagerMock interface{}, cacheMock interface{}) *DisruptionsWatchersManagerMock_CreateAllWatchers_Call {
	return &DisruptionsWatchersManagerMock_CreateAllWatchers_Call{Call: _e.mock.On("CreateAllWatchers", disruption, watcherManagerMock, cacheMock)}
}

func (_c *DisruptionsWatchersManagerMock_CreateAllWatchers_Call) Run(run func(disruption *v1beta1.Disruption, watcherManagerMock Manager, cacheMock cache.Cache)) *DisruptionsWatchersManagerMock_CreateAllWatchers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1beta1.Disruption), args[1].(Manager), args[2].(cache.Cache))
	})
	return _c
}

func (_c *DisruptionsWatchersManagerMock_CreateAllWatchers_Call) Return(_a0 error) *DisruptionsWatchersManagerMock_CreateAllWatchers_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DisruptionsWatchersManagerMock_CreateAllWatchers_Call) RunAndReturn(run func(*v1beta1.Disruption, Manager, cache.Cache) error) *DisruptionsWatchersManagerMock_CreateAllWatchers_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveAllExpiredWatchers provides a mock function with given fields:
func (_m *DisruptionsWatchersManagerMock) RemoveAllExpiredWatchers() {
	_m.Called()
}

// DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAllExpiredWatchers'
type DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call struct {
	*mock.Call
}

// RemoveAllExpiredWatchers is a helper method to define mock.On call
func (_e *DisruptionsWatchersManagerMock_Expecter) RemoveAllExpiredWatchers() *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call {
	return &DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call{Call: _e.mock.On("RemoveAllExpiredWatchers")}
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call) Run(run func()) *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call) Return() *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call {
	_c.Call.Return()
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call) RunAndReturn(run func()) *DisruptionsWatchersManagerMock_RemoveAllExpiredWatchers_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveAllOrphanWatchers provides a mock function with given fields:
func (_m *DisruptionsWatchersManagerMock) RemoveAllOrphanWatchers() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for RemoveAllOrphanWatchers")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAllOrphanWatchers'
type DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call struct {
	*mock.Call
}

// RemoveAllOrphanWatchers is a helper method to define mock.On call
func (_e *DisruptionsWatchersManagerMock_Expecter) RemoveAllOrphanWatchers() *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call {
	return &DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call{Call: _e.mock.On("RemoveAllOrphanWatchers")}
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call) Run(run func()) *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call) Return(_a0 error) *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call) RunAndReturn(run func() error) *DisruptionsWatchersManagerMock_RemoveAllOrphanWatchers_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveAllWatchers provides a mock function with given fields: disruption
func (_m *DisruptionsWatchersManagerMock) RemoveAllWatchers(disruption *v1beta1.Disruption) {
	_m.Called(disruption)
}

// DisruptionsWatchersManagerMock_RemoveAllWatchers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAllWatchers'
type DisruptionsWatchersManagerMock_RemoveAllWatchers_Call struct {
	*mock.Call
}

// RemoveAllWatchers is a helper method to define mock.On call
//   - disruption *v1beta1.Disruption
func (_e *DisruptionsWatchersManagerMock_Expecter) RemoveAllWatchers(disruption interface{}) *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call {
	return &DisruptionsWatchersManagerMock_RemoveAllWatchers_Call{Call: _e.mock.On("RemoveAllWatchers", disruption)}
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call) Run(run func(disruption *v1beta1.Disruption)) *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*v1beta1.Disruption))
	})
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call) Return() *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call {
	_c.Call.Return()
	return _c
}

func (_c *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call) RunAndReturn(run func(*v1beta1.Disruption)) *DisruptionsWatchersManagerMock_RemoveAllWatchers_Call {
	_c.Call.Return(run)
	return _c
}

// NewDisruptionsWatchersManagerMock creates a new instance of DisruptionsWatchersManagerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDisruptionsWatchersManagerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *DisruptionsWatchersManagerMock {
	mock := &DisruptionsWatchersManagerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

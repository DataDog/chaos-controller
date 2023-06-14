// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers

import (
	mock "github.com/stretchr/testify/mock"
	cache "sigs.k8s.io/controller-runtime/pkg/cache"

	v1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
)

// FactoryMock is an autogenerated mock type for the Factory type
type FactoryMock struct {
	mock.Mock
}

type FactoryMock_Expecter struct {
	mock *mock.Mock
}

func (_m *FactoryMock) EXPECT() *FactoryMock_Expecter {
	return &FactoryMock_Expecter{mock: &_m.Mock}
}

// NewChaosPodWatcher provides a mock function with given fields: name, disruption, cacheMock
func (_m *FactoryMock) NewChaosPodWatcher(name string, disruption *v1beta1.Disruption, cacheMock cache.Cache) (Watcher, error) {
	ret := _m.Called(name, disruption, cacheMock)

	var r0 Watcher
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *v1beta1.Disruption, cache.Cache) (Watcher, error)); ok {
		return rf(name, disruption, cacheMock)
	}
	if rf, ok := ret.Get(0).(func(string, *v1beta1.Disruption, cache.Cache) Watcher); ok {
		r0 = rf(name, disruption, cacheMock)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Watcher)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *v1beta1.Disruption, cache.Cache) error); ok {
		r1 = rf(name, disruption, cacheMock)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FactoryMock_NewChaosPodWatcher_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NewChaosPodWatcher'
type FactoryMock_NewChaosPodWatcher_Call struct {
	*mock.Call
}

// NewChaosPodWatcher is a helper method to define mock.On call
//   - name string
//   - disruption *v1beta1.Disruption
//   - cacheMock cache.Cache
func (_e *FactoryMock_Expecter) NewChaosPodWatcher(name interface{}, disruption interface{}, cacheMock interface{}) *FactoryMock_NewChaosPodWatcher_Call {
	return &FactoryMock_NewChaosPodWatcher_Call{Call: _e.mock.On("NewChaosPodWatcher", name, disruption, cacheMock)}
}

func (_c *FactoryMock_NewChaosPodWatcher_Call) Run(run func(name string, disruption *v1beta1.Disruption, cacheMock cache.Cache)) *FactoryMock_NewChaosPodWatcher_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(*v1beta1.Disruption), args[2].(cache.Cache))
	})
	return _c
}

func (_c *FactoryMock_NewChaosPodWatcher_Call) Return(_a0 Watcher, _a1 error) *FactoryMock_NewChaosPodWatcher_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *FactoryMock_NewChaosPodWatcher_Call) RunAndReturn(run func(string, *v1beta1.Disruption, cache.Cache) (Watcher, error)) *FactoryMock_NewChaosPodWatcher_Call {
	_c.Call.Return(run)
	return _c
}

// NewDisruptionTargetWatcher provides a mock function with given fields: name, enableObserver, disruption, cacheMock
func (_m *FactoryMock) NewDisruptionTargetWatcher(name string, enableObserver bool, disruption *v1beta1.Disruption, cacheMock cache.Cache) (Watcher, error) {
	ret := _m.Called(name, enableObserver, disruption, cacheMock)

	var r0 Watcher
	var r1 error
	if rf, ok := ret.Get(0).(func(string, bool, *v1beta1.Disruption, cache.Cache) (Watcher, error)); ok {
		return rf(name, enableObserver, disruption, cacheMock)
	}
	if rf, ok := ret.Get(0).(func(string, bool, *v1beta1.Disruption, cache.Cache) Watcher); ok {
		r0 = rf(name, enableObserver, disruption, cacheMock)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Watcher)
		}
	}

	if rf, ok := ret.Get(1).(func(string, bool, *v1beta1.Disruption, cache.Cache) error); ok {
		r1 = rf(name, enableObserver, disruption, cacheMock)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FactoryMock_NewDisruptionTargetWatcher_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'NewDisruptionTargetWatcher'
type FactoryMock_NewDisruptionTargetWatcher_Call struct {
	*mock.Call
}

// NewDisruptionTargetWatcher is a helper method to define mock.On call
//   - name string
//   - enableObserver bool
//   - disruption *v1beta1.Disruption
//   - cacheMock cache.Cache
func (_e *FactoryMock_Expecter) NewDisruptionTargetWatcher(name interface{}, enableObserver interface{}, disruption interface{}, cacheMock interface{}) *FactoryMock_NewDisruptionTargetWatcher_Call {
	return &FactoryMock_NewDisruptionTargetWatcher_Call{Call: _e.mock.On("NewDisruptionTargetWatcher", name, enableObserver, disruption, cacheMock)}
}

func (_c *FactoryMock_NewDisruptionTargetWatcher_Call) Run(run func(name string, enableObserver bool, disruption *v1beta1.Disruption, cacheMock cache.Cache)) *FactoryMock_NewDisruptionTargetWatcher_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(bool), args[2].(*v1beta1.Disruption), args[3].(cache.Cache))
	})
	return _c
}

func (_c *FactoryMock_NewDisruptionTargetWatcher_Call) Return(_a0 Watcher, _a1 error) *FactoryMock_NewDisruptionTargetWatcher_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *FactoryMock_NewDisruptionTargetWatcher_Call) RunAndReturn(run func(string, bool, *v1beta1.Disruption, cache.Cache) (Watcher, error)) *FactoryMock_NewDisruptionTargetWatcher_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewFactoryMock interface {
	mock.TestingT
	Cleanup(func())
}

// NewFactoryMock creates a new instance of FactoryMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFactoryMock(t mockConstructorTestingTNewFactoryMock) *FactoryMock {
	mock := &FactoryMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

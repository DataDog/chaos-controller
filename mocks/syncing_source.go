// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	handler "sigs.k8s.io/controller-runtime/pkg/handler"

	predicate "sigs.k8s.io/controller-runtime/pkg/predicate"

	workqueue "k8s.io/client-go/util/workqueue"
)

// SyncingSourceMock is an autogenerated mock type for the SyncingSource type
type SyncingSourceMock struct {
	mock.Mock
}

type SyncingSourceMock_Expecter struct {
	mock *mock.Mock
}

func (_m *SyncingSourceMock) EXPECT() *SyncingSourceMock_Expecter {
	return &SyncingSourceMock_Expecter{mock: &_m.Mock}
}

// Start provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *SyncingSourceMock) Start(_a0 context.Context, _a1 handler.EventHandler, _a2 workqueue.RateLimitingInterface, _a3 ...predicate.Predicate) error {
	_va := make([]interface{}, len(_a3))
	for _i := range _a3 {
		_va[_i] = _a3[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _a0, _a1, _a2)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, handler.EventHandler, workqueue.RateLimitingInterface, ...predicate.Predicate) error); ok {
		r0 = rf(_a0, _a1, _a2, _a3...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SyncingSourceMock_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type SyncingSourceMock_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 handler.EventHandler
//   - _a2 workqueue.RateLimitingInterface
//   - _a3 ...predicate.Predicate
func (_e *SyncingSourceMock_Expecter) Start(_a0 interface{}, _a1 interface{}, _a2 interface{}, _a3 ...interface{}) *SyncingSourceMock_Start_Call {
	return &SyncingSourceMock_Start_Call{Call: _e.mock.On("Start",
		append([]interface{}{_a0, _a1, _a2}, _a3...)...)}
}

func (_c *SyncingSourceMock_Start_Call) Run(run func(_a0 context.Context, _a1 handler.EventHandler, _a2 workqueue.RateLimitingInterface, _a3 ...predicate.Predicate)) *SyncingSourceMock_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]predicate.Predicate, len(args)-3)
		for i, a := range args[3:] {
			if a != nil {
				variadicArgs[i] = a.(predicate.Predicate)
			}
		}
		run(args[0].(context.Context), args[1].(handler.EventHandler), args[2].(workqueue.RateLimitingInterface), variadicArgs...)
	})
	return _c
}

func (_c *SyncingSourceMock_Start_Call) Return(_a0 error) *SyncingSourceMock_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SyncingSourceMock_Start_Call) RunAndReturn(run func(context.Context, handler.EventHandler, workqueue.RateLimitingInterface, ...predicate.Predicate) error) *SyncingSourceMock_Start_Call {
	_c.Call.Return(run)
	return _c
}

// WaitForSync provides a mock function with given fields: ctx
func (_m *SyncingSourceMock) WaitForSync(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for WaitForSync")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SyncingSourceMock_WaitForSync_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WaitForSync'
type SyncingSourceMock_WaitForSync_Call struct {
	*mock.Call
}

// WaitForSync is a helper method to define mock.On call
//   - ctx context.Context
func (_e *SyncingSourceMock_Expecter) WaitForSync(ctx interface{}) *SyncingSourceMock_WaitForSync_Call {
	return &SyncingSourceMock_WaitForSync_Call{Call: _e.mock.On("WaitForSync", ctx)}
}

func (_c *SyncingSourceMock_WaitForSync_Call) Run(run func(ctx context.Context)) *SyncingSourceMock_WaitForSync_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *SyncingSourceMock_WaitForSync_Call) Return(_a0 error) *SyncingSourceMock_WaitForSync_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SyncingSourceMock_WaitForSync_Call) RunAndReturn(run func(context.Context) error) *SyncingSourceMock_WaitForSync_Call {
	_c.Call.Return(run)
	return _c
}

// NewSyncingSourceMock creates a new instance of SyncingSourceMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSyncingSourceMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *SyncingSourceMock {
	mock := &SyncingSourceMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

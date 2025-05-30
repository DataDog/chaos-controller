// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package chaosdogfood

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// ChaosDogfoodServerMock is an autogenerated mock type for the ChaosDogfoodServer type
type ChaosDogfoodServerMock struct {
	mock.Mock
}

type ChaosDogfoodServerMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ChaosDogfoodServerMock) EXPECT() *ChaosDogfoodServerMock_Expecter {
	return &ChaosDogfoodServerMock_Expecter{mock: &_m.Mock}
}

// GetCatalog provides a mock function with given fields: _a0, _a1
func (_m *ChaosDogfoodServerMock) GetCatalog(_a0 context.Context, _a1 *emptypb.Empty) (*CatalogReply, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetCatalog")
	}

	var r0 *CatalogReply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *emptypb.Empty) (*CatalogReply, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *emptypb.Empty) *CatalogReply); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*CatalogReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *emptypb.Empty) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChaosDogfoodServerMock_GetCatalog_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCatalog'
type ChaosDogfoodServerMock_GetCatalog_Call struct {
	*mock.Call
}

// GetCatalog is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *emptypb.Empty
func (_e *ChaosDogfoodServerMock_Expecter) GetCatalog(_a0 interface{}, _a1 interface{}) *ChaosDogfoodServerMock_GetCatalog_Call {
	return &ChaosDogfoodServerMock_GetCatalog_Call{Call: _e.mock.On("GetCatalog", _a0, _a1)}
}

func (_c *ChaosDogfoodServerMock_GetCatalog_Call) Run(run func(_a0 context.Context, _a1 *emptypb.Empty)) *ChaosDogfoodServerMock_GetCatalog_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*emptypb.Empty))
	})
	return _c
}

func (_c *ChaosDogfoodServerMock_GetCatalog_Call) Return(_a0 *CatalogReply, _a1 error) *ChaosDogfoodServerMock_GetCatalog_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ChaosDogfoodServerMock_GetCatalog_Call) RunAndReturn(run func(context.Context, *emptypb.Empty) (*CatalogReply, error)) *ChaosDogfoodServerMock_GetCatalog_Call {
	_c.Call.Return(run)
	return _c
}

// Order provides a mock function with given fields: _a0, _a1
func (_m *ChaosDogfoodServerMock) Order(_a0 context.Context, _a1 *FoodRequest) (*FoodReply, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Order")
	}

	var r0 *FoodReply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *FoodRequest) (*FoodReply, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *FoodRequest) *FoodReply); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*FoodReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *FoodRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChaosDogfoodServerMock_Order_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Order'
type ChaosDogfoodServerMock_Order_Call struct {
	*mock.Call
}

// Order is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *FoodRequest
func (_e *ChaosDogfoodServerMock_Expecter) Order(_a0 interface{}, _a1 interface{}) *ChaosDogfoodServerMock_Order_Call {
	return &ChaosDogfoodServerMock_Order_Call{Call: _e.mock.On("Order", _a0, _a1)}
}

func (_c *ChaosDogfoodServerMock_Order_Call) Run(run func(_a0 context.Context, _a1 *FoodRequest)) *ChaosDogfoodServerMock_Order_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*FoodRequest))
	})
	return _c
}

func (_c *ChaosDogfoodServerMock_Order_Call) Return(_a0 *FoodReply, _a1 error) *ChaosDogfoodServerMock_Order_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ChaosDogfoodServerMock_Order_Call) RunAndReturn(run func(context.Context, *FoodRequest) (*FoodReply, error)) *ChaosDogfoodServerMock_Order_Call {
	_c.Call.Return(run)
	return _c
}

// mustEmbedUnimplementedChaosDogfoodServer provides a mock function with no fields
func (_m *ChaosDogfoodServerMock) mustEmbedUnimplementedChaosDogfoodServer() {
	_m.Called()
}

// ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'mustEmbedUnimplementedChaosDogfoodServer'
type ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call struct {
	*mock.Call
}

// mustEmbedUnimplementedChaosDogfoodServer is a helper method to define mock.On call
func (_e *ChaosDogfoodServerMock_Expecter) mustEmbedUnimplementedChaosDogfoodServer() *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call {
	return &ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call{Call: _e.mock.On("mustEmbedUnimplementedChaosDogfoodServer")}
}

func (_c *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call) Run(run func()) *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call) Return() *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call {
	_c.Call.Return()
	return _c
}

func (_c *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call) RunAndReturn(run func()) *ChaosDogfoodServerMock_mustEmbedUnimplementedChaosDogfoodServer_Call {
	_c.Run(run)
	return _c
}

// NewChaosDogfoodServerMock creates a new instance of ChaosDogfoodServerMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewChaosDogfoodServerMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ChaosDogfoodServerMock {
	mock := &ChaosDogfoodServerMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

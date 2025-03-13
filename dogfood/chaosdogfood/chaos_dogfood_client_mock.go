// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package chaosdogfood

import (
	context "context"

	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	mock "github.com/stretchr/testify/mock"
)

// ChaosDogfoodClientMock is an autogenerated mock type for the ChaosDogfoodClient type
type ChaosDogfoodClientMock struct {
	mock.Mock
}

type ChaosDogfoodClientMock_Expecter struct {
	mock *mock.Mock
}

func (_m *ChaosDogfoodClientMock) EXPECT() *ChaosDogfoodClientMock_Expecter {
	return &ChaosDogfoodClientMock_Expecter{mock: &_m.Mock}
}

// GetCatalog provides a mock function with given fields: ctx, in, opts
func (_m *ChaosDogfoodClientMock) GetCatalog(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CatalogReply, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetCatalog")
	}

	var r0 *CatalogReply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*CatalogReply, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *emptypb.Empty, ...grpc.CallOption) *CatalogReply); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*CatalogReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *emptypb.Empty, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChaosDogfoodClientMock_GetCatalog_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCatalog'
type ChaosDogfoodClientMock_GetCatalog_Call struct {
	*mock.Call
}

// GetCatalog is a helper method to define mock.On call
//   - ctx context.Context
//   - in *emptypb.Empty
//   - opts ...grpc.CallOption
func (_e *ChaosDogfoodClientMock_Expecter) GetCatalog(ctx interface{}, in interface{}, opts ...interface{}) *ChaosDogfoodClientMock_GetCatalog_Call {
	return &ChaosDogfoodClientMock_GetCatalog_Call{Call: _e.mock.On("GetCatalog",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *ChaosDogfoodClientMock_GetCatalog_Call) Run(run func(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption)) *ChaosDogfoodClientMock_GetCatalog_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*emptypb.Empty), variadicArgs...)
	})
	return _c
}

func (_c *ChaosDogfoodClientMock_GetCatalog_Call) Return(_a0 *CatalogReply, _a1 error) *ChaosDogfoodClientMock_GetCatalog_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ChaosDogfoodClientMock_GetCatalog_Call) RunAndReturn(run func(context.Context, *emptypb.Empty, ...grpc.CallOption) (*CatalogReply, error)) *ChaosDogfoodClientMock_GetCatalog_Call {
	_c.Call.Return(run)
	return _c
}

// Order provides a mock function with given fields: ctx, in, opts
func (_m *ChaosDogfoodClientMock) Order(ctx context.Context, in *FoodRequest, opts ...grpc.CallOption) (*FoodReply, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Order")
	}

	var r0 *FoodReply
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *FoodRequest, ...grpc.CallOption) (*FoodReply, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *FoodRequest, ...grpc.CallOption) *FoodReply); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*FoodReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *FoodRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChaosDogfoodClientMock_Order_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Order'
type ChaosDogfoodClientMock_Order_Call struct {
	*mock.Call
}

// Order is a helper method to define mock.On call
//   - ctx context.Context
//   - in *FoodRequest
//   - opts ...grpc.CallOption
func (_e *ChaosDogfoodClientMock_Expecter) Order(ctx interface{}, in interface{}, opts ...interface{}) *ChaosDogfoodClientMock_Order_Call {
	return &ChaosDogfoodClientMock_Order_Call{Call: _e.mock.On("Order",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *ChaosDogfoodClientMock_Order_Call) Run(run func(ctx context.Context, in *FoodRequest, opts ...grpc.CallOption)) *ChaosDogfoodClientMock_Order_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*FoodRequest), variadicArgs...)
	})
	return _c
}

func (_c *ChaosDogfoodClientMock_Order_Call) Return(_a0 *FoodReply, _a1 error) *ChaosDogfoodClientMock_Order_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ChaosDogfoodClientMock_Order_Call) RunAndReturn(run func(context.Context, *FoodRequest, ...grpc.CallOption) (*FoodReply, error)) *ChaosDogfoodClientMock_Order_Call {
	_c.Call.Return(run)
	return _c
}

// NewChaosDogfoodClientMock creates a new instance of ChaosDogfoodClientMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewChaosDogfoodClientMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *ChaosDogfoodClientMock {
	mock := &ChaosDogfoodClientMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

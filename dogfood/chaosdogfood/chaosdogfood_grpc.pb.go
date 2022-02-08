// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package chaosdogfood

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ChaosDogfoodClient is the client API for ChaosDogfood service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ChaosDogfoodClient interface {
	Order(ctx context.Context, in *FoodRequest, opts ...grpc.CallOption) (*FoodReply, error)
	GetCatalog(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CatalogReply, error)
}

type chaosDogfoodClient struct {
	cc grpc.ClientConnInterface
}

func NewChaosDogfoodClient(cc grpc.ClientConnInterface) ChaosDogfoodClient {
	return &chaosDogfoodClient{cc}
}

func (c *chaosDogfoodClient) Order(ctx context.Context, in *FoodRequest, opts ...grpc.CallOption) (*FoodReply, error) {
	out := new(FoodReply)
	err := c.cc.Invoke(ctx, "/chaosdogfood.ChaosDogfood/order", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chaosDogfoodClient) GetCatalog(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*CatalogReply, error) {
	out := new(CatalogReply)
	err := c.cc.Invoke(ctx, "/chaosdogfood.ChaosDogfood/getCatalog", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ChaosDogfoodServer is the server API for ChaosDogfood service.
// All implementations must embed UnimplementedChaosDogfoodServer
// for forward compatibility
type ChaosDogfoodServer interface {
	Order(context.Context, *FoodRequest) (*FoodReply, error)
	GetCatalog(context.Context, *emptypb.Empty) (*CatalogReply, error)
	mustEmbedUnimplementedChaosDogfoodServer()
}

// UnimplementedChaosDogfoodServer must be embedded to have forward compatible implementations.
type UnimplementedChaosDogfoodServer struct {
}

func (UnimplementedChaosDogfoodServer) Order(context.Context, *FoodRequest) (*FoodReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Order not implemented")
}
func (UnimplementedChaosDogfoodServer) GetCatalog(context.Context, *emptypb.Empty) (*CatalogReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCatalog not implemented")
}
func (UnimplementedChaosDogfoodServer) mustEmbedUnimplementedChaosDogfoodServer() {}

// UnsafeChaosDogfoodServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ChaosDogfoodServer will
// result in compilation errors.
type UnsafeChaosDogfoodServer interface {
	mustEmbedUnimplementedChaosDogfoodServer()
}

func RegisterChaosDogfoodServer(s grpc.ServiceRegistrar, srv ChaosDogfoodServer) {
	s.RegisterService(&ChaosDogfood_ServiceDesc, srv)
}

func _ChaosDogfood_Order_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FoodRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChaosDogfoodServer).Order(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chaosdogfood.ChaosDogfood/order",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChaosDogfoodServer).Order(ctx, req.(*FoodRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ChaosDogfood_GetCatalog_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ChaosDogfoodServer).GetCatalog(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/chaosdogfood.ChaosDogfood/getCatalog",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ChaosDogfoodServer).GetCatalog(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// ChaosDogfood_ServiceDesc is the grpc.ServiceDesc for ChaosDogfood service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ChaosDogfood_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "chaosdogfood.ChaosDogfood",
	HandlerType: (*ChaosDogfoodServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "order",
			Handler:    _ChaosDogfood_Order_Handler,
		},
		{
			MethodName: "getCatalog",
			Handler:    _ChaosDogfood_GetCatalog_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "chaosdogfood.proto",
}

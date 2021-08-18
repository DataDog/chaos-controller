// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"context"

	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DisruptionListenerClientMock is a mock implementation of the DisruptionListenerClient interface
// used in unit tests
type DisruptionListenerClientMock struct {
	mock.Mock
}

//nolint:golint
func (d *DisruptionListenerClientMock) SendDisruption(ctx context.Context, spec *pb.DisruptionSpec, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	mockArgs := d.Called(ctx, spec)

	return mockArgs.Get(0).(*emptypb.Empty), mockArgs.Error(1)
}

//nolint:golint
func (d *DisruptionListenerClientMock) DisruptionStatus(ctx context.Context, empty *emptypb.Empty, opts ...grpc.CallOption) (*pb.DisruptionSpec, error) {
	mockArgs := d.Called(ctx, empty)

	return mockArgs.Get(0).(*pb.DisruptionSpec), mockArgs.Error(1)
}

//nolint:golint
func (d *DisruptionListenerClientMock) CleanDisruption(ctx context.Context, empty *emptypb.Empty, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	mockArgs := d.Called(ctx, empty)

	return mockArgs.Get(0).(*emptypb.Empty), mockArgs.Error(1)
}

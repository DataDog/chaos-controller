// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package grpc

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
)

func disruptSpec() *pb.DisruptionSpec {
	return &pb.DisruptionSpec{Endpoints: []*pb.EndpointSpec{{
		TargetEndpoint: "/service/Method",
		Alterations:    []*pb.AlterationSpec{{ErrorToReturn: "NOT_FOUND", QueryPercent: 100}},
	}}}
}

// The interceptor runs on every request of the server that embeds the
// listener, concurrently with the injector's Disrupt and ResetDisruptions
// calls. Run with -race: an unguarded read of the configuration is a data race.
func TestInterceptorConcurrentWithDisrupt(t *testing.T) {
	listener := NewDisruptionListener(zap.NewNop().Sugar())
	info := &grpc.UnaryServerInfo{FullMethod: "/service/Method"}
	handler := func(context.Context, interface{}) (interface{}, error) { return &emptypb.Empty{}, nil }

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for range 200 {
			_, _ = listener.ChaosServerInterceptor(context.Background(), nil, info, handler)
		}
	}()

	go func() {
		defer wg.Done()

		for range 100 {
			_, _ = listener.Disrupt(context.Background(), disruptSpec())
			_, _ = listener.ResetDisruptions(context.Background(), &emptypb.Empty{})
		}
	}()

	wg.Wait()
}

// Two injectors disrupting one unconfigured listener at once: exactly one may
// succeed, the other must be told the listener is already configured.
func TestConcurrentDisruptAcceptsOne(t *testing.T) {
	for range 200 {
		listener := NewDisruptionListener(zap.NewNop().Sugar())

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			successes int
		)

		for range 2 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				if _, err := listener.Disrupt(context.Background(), disruptSpec()); err == nil {
					mu.Lock()
					successes++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		if successes != 1 {
			t.Fatalf("%d concurrent Disrupt calls succeeded, want exactly 1", successes)
		}
	}
}

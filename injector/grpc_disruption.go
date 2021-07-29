// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaos_grpc "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/proto"
	"google.golang.org/grpc"
)

// GRPCDisruptionInjector describes a grpc disruption
type GRPCDisruptionInjector struct {
	spec   v1beta1.GRPCDisruptionSpec
	config GRPCDisruptionInjectorConfig
	client pb.DisruptionListenerClient
}

// GRPCDisruptionInjectorConfig contains all needed drivers to create a grpc disruption
type GRPCDisruptionInjectorConfig struct {
	Config
}

// NewGRPCDisruptionInjector creates a GRPCDisruptionInjector object with the given config,
// missing fields are initialized with the defaults
func NewGRPCDisruptionInjector(spec v1beta1.GRPCDisruptionSpec, config GRPCDisruptionInjectorConfig) (Injector, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	serverAddr := "10.244.0.53:50051" // TODO: make address / port of gRPC service
	config.Log.Infow("connecting to " + serverAddr + "...")

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		config.Log.Fatalf("fail to dial: %v", err)
	}

	client := pb.NewDisruptionListenerClient(conn)

	return GRPCDisruptionInjector{
		spec:   spec,
		config: config,
		client: client,
	}, err
}

// Inject injects the given dns disruption into the given container
func (i GRPCDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding grpc disruption", "spec", i.spec)

	chaos_grpc.ExecuteSendDisruption(i.client, i.spec)
	return nil
}

// Clean removes the injected disruption from the given container
func (i GRPCDisruptionInjector) Clean() error {
	i.config.Log.Infow("removing grpc disruption", "spec", i.spec)

	// defer i.client.conn.Close() TODO: figure out if this is necessary as I can't find a way to access i.client.cc
	chaos_grpc.ExecuteCleanDisruption(i.client)

	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package injector

import (
	"errors"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaos_grpc "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"github.com/DataDog/chaos-controller/types"
	"google.golang.org/grpc"
)

// GRPCDisruptionInjector describes a grpc disruption
type GRPCDisruptionInjector struct {
	spec       v1beta1.GRPCDisruptionSpec
	config     GRPCDisruptionInjectorConfig
	serverAddr string
}

// GRPCDisruptionInjectorConfig contains all needed drivers to create a grpc disruption
type GRPCDisruptionInjectorConfig struct {
	Config
}

// NewGRPCDisruptionInjector creates a GRPCDisruptionInjector object with the given config,
// missing fields are initialized with the defaults
func NewGRPCDisruptionInjector(spec v1beta1.GRPCDisruptionSpec, config GRPCDisruptionInjectorConfig) Injector {
	return &GRPCDisruptionInjector{
		spec:       spec,
		config:     config,
		serverAddr: config.TargetPodIP + ":" + strconv.Itoa(spec.Port),
	}
}

func (i *GRPCDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindGRPCDisruption
}

// Inject injects the given grpc disruption into the given container
func (i *GRPCDisruptionInjector) Inject() error {
	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	if i.config.DryRun {
		i.config.Log.Infow("adding dry run mode grpc disruption", "spec", i.spec)
		return nil
	}

	conn, err := connectToServer(i.serverAddr)
	if err != nil {
		i.config.Log.Errorf(err.Error())
		return err
	}

	i.config.Log.Infow("adding grpc disruption", "spec", i.spec)

	err = chaos_grpc.SendGrpcDisruption(pb.NewDisruptionListenerClient(conn), i.spec)

	if err != nil {
		i.config.Log.Error("Received an error: %v", err)
	}

	return conn.Close()
}

func (i *GRPCDisruptionInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes the injected disruption from the given container
func (i *GRPCDisruptionInjector) Clean() error {
	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	if i.config.DryRun {
		i.config.Log.Infow("removing dry run mode grpc disruption", "spec", i.spec)
		return nil
	}

	conn, err := connectToServer(i.serverAddr)
	if err != nil {
		i.config.Log.Errorf(err.Error())
		return err
	}

	i.config.Log.Infow("removing grpc disruption", "spec", i.spec)

	err = chaos_grpc.ClearGrpcDisruptions(pb.NewDisruptionListenerClient(conn))

	if err != nil {
		i.config.Log.Error("Received an error: %v", err)
	}

	return conn.Close()
}

func connectToServer(serverAddr string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure()) // Future Work: make secure
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, errors.New("fail to dial: " + serverAddr)
	}

	return conn, nil
}

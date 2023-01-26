// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaos_grpc "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"github.com/DataDog/chaos-controller/types"
	"google.golang.org/grpc"
)

// Five Seconds timeout before aborting the attempt to connect to server
// so that in turn, when user requests, the injector pod can be terminated
const connectionTimeout = time.Duration(5) * time.Second

// GRPCDisruptionInjector describes a grpc disruption
type GRPCDisruptionInjector struct {
	spec       v1beta1.GRPCDisruptionSpec
	config     GRPCDisruptionInjectorConfig
	serverAddr string
	timeout    time.Duration
}

// GRPCDisruptionInjectorConfig contains all needed drivers to create a grpc disruption
type GRPCDisruptionInjectorConfig struct {
	Config
	State InjectorState
}

// NewGRPCDisruptionInjector creates a GRPCDisruptionInjector object with the given config,
// missing fields are initialized with the defaults
func NewGRPCDisruptionInjector(spec v1beta1.GRPCDisruptionSpec, config GRPCDisruptionInjectorConfig) Injector {
	config.State = Created

	return &GRPCDisruptionInjector{
		spec:       spec,
		config:     config,
		serverAddr: config.TargetPodIP + ":" + strconv.Itoa(spec.Port),
		timeout:    connectionTimeout,
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

	conn, err := i.connectToServer()
	if err != nil {
		i.config.Log.Errorf(err.Error())
		return err
	}

	// as long as we managed to dial the server, then we have to assume we're injected
	i.config.State = Injected

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
	if i.config.State != Injected {
		i.config.Log.Infow("nothing to clean", "spec", i.spec, "state", i.config.State)
		return nil
	}

	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	if i.config.DryRun {
		i.config.Log.Infow("removing dry run mode grpc disruption", "spec", i.spec)
		return nil
	}

	conn, err := i.connectToServer()
	if err != nil {
		i.config.Log.Errorf(err.Error())
		return err
	}

	i.config.Log.Infow("removing grpc disruption", "spec", i.spec)

	err = chaos_grpc.ClearGrpcDisruptions(pb.NewDisruptionListenerClient(conn))

	if err != nil {
		i.config.Log.Error("Received an error: %v", err)
		return err
	}

	i.config.State = Cleaned

	return conn.Close()
}

func (i *GRPCDisruptionInjector) connectToServer() (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(), // Future Work: make secure
		grpc.WithBlock(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), i.timeout)

	defer cancel()

	conn, err := grpc.DialContext(ctx, i.serverAddr, opts...)

	if err != nil {
		return nil, errors.New("fail to dial: " + i.serverAddr)
	}

	return conn, nil
}

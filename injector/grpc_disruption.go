// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"errors"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaos_grpc "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruption_listener"
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
func NewGRPCDisruptionInjector(spec v1beta1.GRPCDisruptionSpec, config GRPCDisruptionInjectorConfig) (Injector, error) {
	return GRPCDisruptionInjector{
		spec:       spec,
		config:     config,
		serverAddr: config.TargetPodIP + ":" + strconv.Itoa(spec.Port),
	}, nil
}

// Inject injects the given dns disruption into the given container
func (i GRPCDisruptionInjector) Inject() error {
	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	conn, err := connectToServer(i.serverAddr)
	if err != nil {
		i.config.Log.Fatalf(err.Error())
		return err
	}

	defer func() {
		i.config.Log.Infow("closing connection...")
		conn.Close()
	}()

	client := pb.NewDisruptionListenerClient(conn)

	i.config.Log.Infow("adding grpc disruption", "spec", i.spec)

	chaos_grpc.ExecuteSendDisruption(client, i.spec)

	return nil
}

// Clean removes the injected disruption from the given container
func (i GRPCDisruptionInjector) Clean() error {
	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	conn, err := connectToServer(i.serverAddr)
	if err != nil {
		i.config.Log.Fatalf(err.Error())
		return err
	}

	defer func() {
		i.config.Log.Infow("closing connection...")
		conn.Close()
	}()

	client := pb.NewDisruptionListenerClient(conn)

	i.config.Log.Infow("removing grpc disruption", "spec", i.spec)

	chaos_grpc.ExecuteCleanDisruption(client)

	return nil
}

func connectToServer(serverAddr string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, errors.New("fail to dial: " + serverAddr)
	}

	return conn, nil
}

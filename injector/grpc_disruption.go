// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package injector

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaos_grpc "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
		serverAddr: config.Disruption.TargetPodIP + ":" + strconv.Itoa(spec.Port),
		timeout:    connectionTimeout,
	}
}

func (i *GRPCDisruptionInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *GRPCDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindGRPCDisruption
}

// Inject injects the given grpc disruption into the given container
func (i *GRPCDisruptionInjector) Inject() error {
	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("adding dry run mode grpc disruption", tags.SpecKey, i.spec)
		return nil
	}

	conn, err := i.connectToServer()
	if err != nil {
		return fmt.Errorf("an error occurred when connecting to server (inject): %w", err)
	}

	// as long as we managed to dial the server, then we have to assume we're injected
	i.config.State = Injected

	i.config.Log.Infow("adding grpc disruption", tags.SpecKey, i.spec)

	err = chaos_grpc.SendGrpcDisruption(pb.NewDisruptionListenerClient(conn), i.spec)

	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.Unimplemented {
				// error must have been --> code = Unimplemented desc = unknown service disruptionlistener.DisruptionListener
				// We dialed a grpc server, but it does not have the chaos-interceptor, no disruption is possible
				i.config.State = Created
				i.config.Log.Warnw("disruption attempted on grpc server without the chaos-interceptor", tags.SpecKey, i.spec)

				logConnClose := func() {
					connErr := conn.Close()
					i.config.Log.Errorw("could not close grpc connection", tags.ErrorKey, connErr)
				}

				defer logConnClose()

				return err
			}
		}

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
		i.config.Log.Infow("nothing to clean", tags.SpecKey, i.spec, tags.StateKey, i.config.State)
		return nil
	}

	i.config.Log.Infow("connecting to " + i.serverAddr + "...")

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("removing dry run mode grpc disruption", tags.SpecKey, i.spec)
		return nil
	}

	conn, err := i.connectToServer()
	if err != nil {
		return fmt.Errorf("an error occurred when connecting to server (clean): %w", err)
	}

	i.config.Log.Infow("removing grpc disruption", tags.SpecKey, i.spec)

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
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Future Work: make secure
	}

	conn, err := grpc.NewClient(i.serverAddr, opts...)
	if err != nil {
		return nil, errors.New("fail to dial: " + i.serverAddr)
	}

	return conn, nil
}

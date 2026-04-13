// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type InjectorState string

const (
	Created           InjectorState = "created"
	Injected          InjectorState = "injected"
	Cleaned           InjectorState = "cleaned"
	UnknownTargetName               = "unknown target name"
)

// Injector is an interface being able to inject or clean disruptions
type Injector interface {
	GetDisruptionKind() types.DisruptionKindName
	Inject() error
	TargetName() string
	UpdateConfig(config Config)
	Clean() error
}

// Config represents a generic injector config
type Config struct {
	Log                *zap.SugaredLogger
	MetricsSink        metrics.Sink
	TargetContainer    container.Container
	DisruptionDeadline time.Time
	Cgroup             cgroup.Manager
	Netns              netns.Manager
	K8sClient          kubernetes.Interface
	Disruption         chaosapi.DisruptionArgs
	InjectorCtx        context.Context
}

// TargetName returns the name of the target that relates to this configuration
// node name if the disruption if at the node level
// target container name if available
// UnknownTargetName otherwise
func (c Config) TargetName() string {
	if c.Disruption.Level == types.DisruptionLevelNode {
		return c.Disruption.TargetNodeName
	}

	if c.TargetContainer != nil {
		return c.TargetContainer.Name()
	}

	return UnknownTargetName
}

// stopAndWaitForBackgroundCmd stops a background command and waits for the process to fully exit
// before returning. This prevents cgroup race conditions during pulse mode re-injection.
func stopAndWaitForBackgroundCmd(log *zap.SugaredLogger, backgroundCmd command.BackgroundCmd, cancel context.CancelFunc) error {
	if backgroundCmd == nil {
		return nil
	}

	defer cancel()

	if err := backgroundCmd.Stop(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("unable to stop background process: %w", err)
	}

	select {
	case <-backgroundCmd.Done():
	case <-time.After(5 * time.Second):
		log.Warnw("timed out waiting for background process to exit")
	}

	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package injector

import (
	"context"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
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
	DNS                network.DNSConfig
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

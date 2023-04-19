// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
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
	Created  InjectorState = "created"
	Injected InjectorState = "injected"
	Cleaned  InjectorState = "cleaned"
)

// Injector is an interface being able to inject or clean disruptions
type Injector interface {
	GetDisruptionKind() types.DisruptionKindName
	Inject() error
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
}

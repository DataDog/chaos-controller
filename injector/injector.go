// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// Injector is an interface being able to inject or clean disruptions
type Injector interface {
	Inject() error
	Clean() error
}

// Config represents a generic injector config
type Config struct {
	DryRun      bool
	OnInit      bool
	Log         *zap.SugaredLogger
	MetricsSink metrics.Sink
	Kind        types.DisruptionKindName
	Level       types.DisruptionLevel
	Container   container.Container
	PodIP       string
	Cgroup      cgroup.Manager
	Netns       netns.Manager
	K8sClient   kubernetes.Interface
}

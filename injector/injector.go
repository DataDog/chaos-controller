// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// Injector is an interface being able to inject or clean disruptions
type Injector interface {
	Inject()
	Clean()
}

// injector represents a generic failure injector
type injector struct {
	log  *zap.SugaredLogger
	ms   metrics.Sink
	uid  string
	kind types.DisruptionKind
}

// containerInjector represents an injector for containers
type containerInjector struct {
	injector
	container container.Container
}

func (i injector) handleMetricSinkError(err error) {
	if err != nil {
		i.log.Errorw("error sending metric or event", "sink", i.ms.GetSinkName(), "error", err)
	}
}

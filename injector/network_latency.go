// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// networkLatencyInjector describes a network latency
type networkLatencyInjector struct {
	containerInjector
	spec   v1beta1.NetworkLatencySpec
	config NetworkDisruptionConfig
}

// NewNetworkLatencyInjector creates a NetworkLatencyInjector object with the default drivers
func NewNetworkLatencyInjector(uid string, spec v1beta1.NetworkLatencySpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	return NewNetworkLatencyInjectorWithConfig(uid, spec, ctn, log, ms, NewNetworkDisruptionConfig(log))
}

// NewNetworkLatencyInjectorWithConfig creates a NetworkLatencyInjector object with the given config,
// missing fields being initialized with the defaults
func NewNetworkLatencyInjectorWithConfig(uid string, spec v1beta1.NetworkLatencySpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkDisruptionConfig) Injector {
	return networkLatencyInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindNetworkLatency,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}
}

// Inject injects network latency according to the current spec
func (i networkLatencyInjector) Inject() {
	var err error

	i.log.Info("injecting latency")

	// handle metrics
	defer func() {
		i.handleMetricSinkError(i.ms.MetricInjected(i.container.ID(), i.uid, err == nil, i.kind, []string{}))
	}()

	// enter container network namespace
	err = i.container.EnterNetworkNamespace()
	if err != nil {
		i.log.Fatalw("unable to enter the given container network namespace", "error", err, "id", i.container.ID())
	}

	// defer the exit on return
	defer func() {
		err := i.container.ExitNetworkNamespace()
		if err != nil {
			i.log.Fatalw("unable to exit the given container network namespace", "error", err, "id", i.container.ID())
		}
	}()

	delay := time.Duration(i.spec.Delay) * time.Millisecond

	i.config.AddLatency(i.spec.Hosts, i.spec.Port, delay)
}

// Clean cleans the injected latency
func (i networkLatencyInjector) Clean() {
	var err error

	i.log.Info("cleaning latency")

	// handle metrics
	defer func() {
		i.handleMetricSinkError(i.ms.MetricCleaned(i.container.ID(), i.uid, err == nil, i.kind, []string{}))
	}()

	// enter container network namespace
	err = i.container.EnterNetworkNamespace()
	if err != nil {
		i.log.Fatalw("unable to enter the given container network namespace", "error", err, "id", i.container.ID())
	}

	// defer the exit on return
	defer func() {
		err := i.container.ExitNetworkNamespace()
		if err != nil {
			i.log.Fatalw("unable to exit the given container network namespace", "error", err, "id", i.container.ID())
		}
	}()

	i.config.ClearAllQdiscs(i.spec.Hosts)
}

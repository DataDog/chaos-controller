// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// networkLimitationInjector describes a network bandwidth limitation
type networkLimitationInjector struct {
	containerInjector
	spec   v1beta1.NetworkLimitationSpec
	config NetworkDisruptionConfig
}

// NewNetworkLimitationInjector creates a NetworkLimitationInjector object with the default drivers
func NewNetworkLimitationInjector(uid string, spec v1beta1.NetworkLimitationSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	return NewNetworkLimitationInjectorWithConfig(uid, spec, ctn, log, ms, NewNetworkDisruptionConfig(log))
}

// NewNetworkLimitationInjectorWithConfig creates a NetworkLimitationInjector object with the given config,
// missing fields being initialized with the defaults
func NewNetworkLimitationInjectorWithConfig(uid string, spec v1beta1.NetworkLimitationSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkDisruptionConfig) Injector {
	return networkLimitationInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindNetworkLimitation,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}
}

// Inject injects network bandwidth limitation according to the current spec
func (i networkLimitationInjector) Inject() {
	var err error

	i.log.Info("injecting bandwidth limitation: %s", i.spec)

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

	i.config.AddOutputLimit(i.spec.Hosts, i.spec.Port, i.spec.BytesPerSec)

	i.log.Info("successfully injected output bandwidth limit of %s bytes/sec to pod", i.spec.BytesPerSec)
}

// Clean cleans the injected bandwidth limitation
func (i networkLimitationInjector) Clean() {
	var err error

	i.log.Info("cleaning bandwidth limitation")

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

	i.log.Info("successfully cleared injected bandwidth limit")
}

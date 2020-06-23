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

// networkFailureInjector describes a network failure
type networkFailureInjector struct {
	containerInjector
	spec   v1beta1.NetworkFailureSpec
	config NetworkDisruptionConfig
}

// NewNetworkFailureInjector creates a NetworkFailureInjector object with default drivers
func NewNetworkFailureInjector(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	return NewNetworkFailureInjectorWithConfig(uid, spec, ctn, log, ms, NewNetworkDisruptionConfig(log))
}

// NewNetworkFailureInjectorWithConfig creates a NetworkFailureInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkFailureInjectorWithConfig(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkDisruptionConfig) Injector {
	return networkFailureInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindNetworkFailure,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}
}

// Inject injects the given network failure into the given container
func (i networkFailureInjector) Inject() {
	var err error

	i.log.Info("injecting network failure")

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

	drop := i.spec.Drop
	corrupt := i.spec.Corrupt

	if i.spec.Drop != 0 {
		// add drop rate
		i.log.Info("Adding Drop rate of ", i.spec.Drop)
		i.config.AddDrop(i.spec.Hosts, i.spec.Port, drop)
		i.log.Info("successfully injected drop of %s to pod", i.spec.Drop)
	}

	if i.spec.Corrupt != 0 {
		// add corruption
		i.log.Info("Adding Corruption rate of ", i.spec.Corrupt)
		i.config.AddCorrupt(i.spec.Hosts, i.spec.Port, corrupt)
		i.log.Info("successfully injected corruption of %s to pod", i.spec.Corrupt)
	}
}

// Clean removes all the injected failures in the given container
func (i networkFailureInjector) Clean() {
	var err error

	i.log.Info("cleaning failures")
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
	i.log.Info("successfully cleared injected network failure")

}

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

// networkDisruptionInjector describes a network disruption
type networkDisruptionInjector struct {
	containerInjector
	spec   v1beta1.NetworkDisruptionSpec
	config NetworkDisruptionConfig
}

// NewNetworkDisruptionInjector creates a NetworkDisruptionInjector object with default drivers
func NewNetworkDisruptionInjector(uid string, spec v1beta1.NetworkDisruptionSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	config := NewNetworkDisruptionConfigWithDefaults(log, spec.Hosts, spec.Port, spec.Protocol)

	return NewNetworkDisruptionInjectorWithConfig(uid, spec, ctn, log, ms, config)
}

// NewNetworkDisruptionInjectorWithConfig creates a NetworkDisruptionInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkDisruptionInjectorWithConfig(uid string, spec v1beta1.NetworkDisruptionSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkDisruptionConfig) Injector {
	return networkDisruptionInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindNetworkDisruption,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}
}

// Inject injects the given network disruption into the given container
func (i networkDisruptionInjector) Inject() {
	var err error

	i.log.Info("injecting network disruption")

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

	i.log.Infow("adding network disruptions", "drop", i.spec.Drop, "corrupt", i.spec.Corrupt)

	// add netem
	if i.spec.Delay > 0 || i.spec.Drop > 0 || i.spec.Corrupt > 0 {
		delay := time.Duration(i.spec.Delay) * time.Millisecond
		i.config.AddNetem(delay, i.spec.Drop, i.spec.Corrupt)
	}

	// add tbf
	if i.spec.BandwidthLimit > 0 {
		i.config.AddOutputLimit(uint(i.spec.BandwidthLimit))
	}

	if err := i.config.ApplyOperations(); err != nil {
		i.log.Fatalf("error applying tc operations", "error", err)
	}

	i.log.Info("operations applied successfully")
}

// Clean removes all the injected disruption in the given container
func (i networkDisruptionInjector) Clean() {
	var err error

	i.log.Info("cleaning disruptions")
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

	if err := i.config.ClearOperations(); err != nil {
		i.log.Fatalw("error clearing tc operations", "error", err)
	}

	i.log.Info("successfully cleared injected network disruption")
}

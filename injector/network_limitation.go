// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
    "github.com/DataDog/chaos-controller/api/v1beta1"
    "github.com/DataDog/chaos-controller/container"
    "github.com/DataDog/chaos-controller/metrics"
    "github.com/DataDog/chaos-controller/network"
    "github.com/DataDog/chaos-controller/types"
    "go.uber.org/zap"
)

// networkLimitationInjector describes a network bandwidth limitation
type networkLimitationInjector struct {
    containerInjector
    spec   v1beta1.NetworkLimitationSpec
    config NetworkLimitationInjectorConfig
}

// NetworkLimitationInjectorConfig contains needed drivers to create
// a NetworkLimitationInjector
type NetworkLimitationInjectorConfig struct {
    TrafficController network.TrafficController
    NetlinkAdapter    network.NetlinkAdapter
    DNSClient         network.DNSClient
}

// NewNetworkLimitationInjector creates a NetworkLimitationInjector object with the default drivers
func NewNetworkLimitationInjector(uid string, spec v1beta1.NetworkLimitationSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
    return NewNetworkLimitationInjectorWithConfig(uid, spec, ctn, log, ms, NetworkLimitationInjectorConfig{})
}

// NewNetworkLimitationInjectorWithConfig creates a NetworkLimitationInjector object with the given config,
// missing fields being initialized with the defaults
func NewNetworkLimitationInjectorWithConfig(uid string, spec v1beta1.NetworkLimitationSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkLimitationInjectorConfig) Injector {
    // traffic controller
    if config.TrafficController == nil {
        config.TrafficController = network.NewTrafficController(log)
    }

    // netlink adapter
    if config.NetlinkAdapter == nil {
        config.NetlinkAdapter = network.NewNetlinkAdapter()
    }

    // dns resolver
    if config.DNSClient == nil {
        config.DNSClient = network.NewDNSClient()
    }

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

    i.log.Info("Will inject bandwidth limitation to bytes per sec: ", i.spec.BytesPerSec)

}

// Clean cleans the injected bandwidth limitation
func (i networkLimitationInjector) Clean() {

    i.log.Info("Will clean bandwidth limitation")

}

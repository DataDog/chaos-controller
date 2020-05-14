// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"net"

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

func (i networkLimitationInjector) getInterfacesByIP() (map[string][]*net.IPNet, error) {
    linkByIP := map[string][]*net.IPNet{}

    // this is just copied from `network_limitation.go`
    // currently omitted support for specific IPs, Hosts for bandwidth limit
    // if we want to support that, probably should factor it out of `network_limitation.go`

    i.log.Info("no hosts specified, all interfaces will be impacted")

    // prepare links/IP association by pre-creating links
    links, err := i.config.NetlinkAdapter.LinkList()
    if err != nil {
        i.log.Fatalf("can't list links: %w", err)
    }
    for _, link := range links {
        i.log.Infof("adding interface %s", link.Name())
        linkByIP[link.Name()] = []*net.IPNet{}
    }

    return linkByIP, nil
}

// Inject injects network bandwidth limitation according to the current spec
func (i networkLimitationInjector) Inject() {
    var err error

    i.log.Info("injecting bandwidth limitation")

    parent := "root"

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

    i.log.Info("auto-detecting interfaces to apply bandwidth limitation to...")

    linkByIP, err := i.getInterfacesByIP()
    if err != nil {
        i.log.Fatalw("can't get interfaces per IP listing: %w", err)
    }

    // for each link/ip association, add bandwidth limitation
    for linkName, _ := range linkByIP {
        // retrieve link from name
        link, err := i.config.NetlinkAdapter.LinkByName(linkName)
        if err != nil {
            i.log.Fatalf("can't retrieve link %s: %w", linkName, err)
        }

        // currently omitted support for specific IPs, Hosts for bandwidth limit
        // if we want to support that, probably should factor it out of `network_limitation.go`

        i.log.Info("going to add bandwidth limit of %s bytes per sec now...", i.spec.BytesPerSec)

        // add limitation
        err2 := i.config.TrafficController.AddOutputLimit(link.Name(), parent, 0, i.spec.BytesPerSec)
        if err2 != nil {
            i.log.Fatalf("can't add bandwidth limit to the newly created qdisc for interface %s: %w", link.Name(), err2)
        }
    }
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

    linkByIP, err := i.getInterfacesByIP()
    if err != nil {
        i.log.Fatalf("can't get interfaces per IP map: %w", err)
    }

    for linkName := range linkByIP {
        i.log.Infof("clearing root qdisc for interface %s", linkName)

        // retrieve link from name
        link, err := i.config.NetlinkAdapter.LinkByName(linkName)
        if err != nil {
            i.log.Fatalf("can't retrieve link %s: %w", linkName, err)
        }

        // ensure qdisc isn't cleared before clearing it to avoid any tc error
        cleared, err := i.config.TrafficController.IsQdiscCleared(link.Name())
        if err != nil {
            i.log.Fatalf("can't ensure the %s link qdisc is cleared or not: %w", link.Name(), err)
        }

        // clear link qdisc if needed
        if !cleared {
            if err := i.config.TrafficController.ClearQdisc(link.Name()); err != nil {
                i.log.Fatalf("can't delete the %s link qdisc: %w", link.Name(), err)
            }
        } else {
            i.log.Infof("%s link qdisc is already cleared, skipping", link.Name())
        }
    }
}

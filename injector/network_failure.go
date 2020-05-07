// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
  "fmt"
	"net"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

// NetworkFailureInjectorConfig contains needed drivers to create
// a NetworkFailureInjector
type NetworkFailureInjectorConfig struct {
	TrafficController network.TrafficController
	NetlinkAdapter    network.NetlinkAdapter
	DNSClient         network.DNSClient
}

// networkFailureInjector describes a network failure
type networkFailureInjector struct {
	containerInjector
	config NetworkFailureInjectorConfig
	spec   v1beta1.NetworkFailureSpec
}

// NewNetworkFailureInjector creates a NetworkFailureInjector object with default drivers
func NewNetworkFailureInjector(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	return NewNetworkFailureInjectorWithConfig(uid, spec, ctn, log, ms, NetworkFailureInjectorConfig{})
}

// NewNetworkFailureInjectorWithConfig creates a NetworkFailureInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkFailureInjectorWithConfig(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config NetworkFailureInjectorConfig) Injector{
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

func (i networkFailureInjector) getInterfacesByIP() (map[string][]*net.IPNet, error) {
	linkByIP := map[string][]*net.IPNet{}

	if len(i.spec.Hosts) > 0 {
		i.log.Info("auto-detecting interfaces to apply network failure to...")
		// resolve hosts
		ips, err := resolveHosts(i.config.DNSClient, i.spec.Hosts)
		if err != nil {
			return nil, fmt.Errorf("can't resolve given hosts: %w", err)
		}

		// get the association between IP and interfaces to know
		// which interfaces we have to inject failure to
		for _, ip := range ips {
			// get routes for resolved destination IP
			routes, err := i.config.NetlinkAdapter.RoutesForIP(ip)
			if err != nil {
				return nil, fmt.Errorf("can't get route for IP %s: %w", ip.String(), err)
			}

			// for each route, get the related interface and add it to the association
			// between interfaces and IPs
			for _, route := range routes {
				i.log.Infof("IP %s belongs to interface %s", ip.String(), route.Link().Name())

				// store association, initialize the map entry if not present yet
				if _, ok := linkByIP[route.Link().Name()]; !ok {
					linkByIP[route.Link().Name()] = []*net.IPNet{}
				}

				linkByIP[route.Link().Name()] = append(linkByIP[route.Link().Name()], ip)
			}
		}
	} else {
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
	}

	return linkByIP, nil
}

// Inject injects the given network failure into the given container
func (i networkFailureInjector) Inject() {
  var err error

  i.log.Info("injecting network failure")

  // handle metrics
	defer func() {
		i.handleMetricSinkError(i.ms.MetricInjected(i.container.ID(), i.uid, err == nil, i.kind, []string{}))
	}()

  drop := i.spec.Drop
  corrupt := i.spec.Corrupt
  parent := "root"

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

  i.log.Info("auto-detecting interfaces to apply network failure to...")

  linkByIP, err := i.getInterfacesByIP()
	if err != nil {
		i.log.Fatalw("can't get interfaces per IP listing: %w", err)
	}

  // for each link/ip association, add latency
	for linkName, ips := range linkByIP {
    clearTxQlen := false

    // retrieve link from name
		link, err := i.config.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			i.log.Fatalf("can't retrieve link %s: %w", linkName, err)
		}

    // if at least one IP has been specified, we need to create a prio qdisc to be able to apply
		// a filter and a delay only on traffic going to those IP
		if len(ips) > 0 {
      // set the tx qlen if not already set as it is required to create a prio qdisc without dropping
			// all the outgoing traffic
			// this qlen will be removed once the injection is done if it was not present before
			if link.TxQLen() == 0 {
				i.log.Infof("setting tx qlen for interface %s", link.Name())

				clearTxQlen = true

				if err := link.SetTxQLen(1000); err != nil {
					i.log.Fatalf("can't set tx queue length on interface %s: %w", link.Name(), err)
				}
			}

      // create a new qdisc for the given interface of type prio with 4 bands instead of 3
			// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP/port
			// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to add network failure to
			// all the outgoing traffic
			parent = "1:4"
			priomap := [16]uint32{1, 2, 2, 2, 1, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1}

      if err := i.config.TrafficController.AddPrio(link.Name(), "root", 1, 4, priomap); err != nil {
				i.log.Fatalf("can't create a new qdisc for interface %s: %w", link.Name(), err)
			}

      if i.spec.Drop != 0 {
        // add drop rate
        if err := i.config.TrafficController.AddDrop(link.Name(), parent, 0, drop); err != nil {
    			i.log.Fatalf("can't add drop rate to the newly created qdisc for interface %s: %w", link.Name(), err)
    		}
      }

      if i.spec.Corrupt != 0 {
        // add corruption
        if err := i.config.TrafficController.AddCorrupt(link.Name(), parent, 0, corrupt); err != nil {
    			i.log.Fatalf("can't add corruption to the newly created qdisc for interface %s: %w", link.Name(), err)
    		}
      }

      // if only some hosts/ports are targeted, redirect the traffic to the extra band created earlier
  		// where the failures are applied
  		port := i.spec.Port

  		if len(ips) > 0 {
  			for _, ip := range ips {
  				if err := i.config.TrafficController.AddFilter(link.Name(), "1:0", 0, ip, port, "1:4"); err != nil {
  					i.log.Fatalf("can't add a filter to interface %s: %w", link.Name(), err)
  				}
  			}
  		}

      // reset the interface transmission queue length once filters have been created
  		if clearTxQlen {
  			i.log.Infof("clearing tx qlen for interface %s", link.Name())

  			if err := link.SetTxQLen(0); err != nil {
  				i.log.Fatalf("can't clear %s link transmission queue length: %w", link.Name(), err)
  			}
  		}
    }
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

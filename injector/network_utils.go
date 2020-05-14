// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"net"
	"fmt"
	"time"


	"github.com/DataDog/chaos-controller/network"
	"go.uber.org/zap"
)

// NetworkDisruptionConfig contains needed drivers to create a network disruption using `tc`
type NetworkDisruptionConfig struct {
	log               *zap.SugaredLogger
	TrafficController network.TrafficController
	NetlinkAdapter    network.NetlinkAdapter
	DNSClient         network.DNSClient
}

func (c NetworkDisruptionConfig) Initialize(logger *zap.SugaredLogger) {
	//logger
	c.log = logger
	// traffic controller
	if c.TrafficController == nil {
		c.TrafficController = network.NewTrafficController(logger)
	}
	// netlink adapter
	if c.NetlinkAdapter == nil {
		c.NetlinkAdapter = network.NewNetlinkAdapter()
	}
	// dns resolver
	if c.DNSClient == nil {
		c.DNSClient = network.NewDNSClient()
	}
}

func (c NetworkDisruptionConfig) getInterfacesByIP(hosts []string) (map[string][]*net.IPNet, error) {
	linkByIP := map[string][]*net.IPNet{}

	if len(hosts) > 0 {
		c.log.Info("auto-detecting interfaces to apply latency to...")
		// resolve hosts
		ips, err := resolveHosts(c.DNSClient, hosts)
		if err != nil {
			return nil, fmt.Errorf("can't resolve given hosts: %w", err)
		}

		// get the association between IP and interfaces to know
		// which interfaces we have to inject latency to
		for _, ip := range ips {
			// get routes for resolved destination IP
			routes, err := c.NetlinkAdapter.RoutesForIP(ip)
			if err != nil {
				return nil, fmt.Errorf("can't get route for IP %s: %w", ip.String(), err)
			}

			// for each route, get the related interface and add it to the association
			// between interfaces and IPs
			for _, route := range routes {
				c.log.Infof("IP %s belongs to interface %s", ip.String(), route.Link().Name())

				// store association, initialize the map entry if not present yet
				if _, ok := linkByIP[route.Link().Name()]; !ok {
					linkByIP[route.Link().Name()] = []*net.IPNet{}
				}

				linkByIP[route.Link().Name()] = append(linkByIP[route.Link().Name()], ip)
			}
		}
	} else {
		c.log.Info("no hosts specified, all interfaces will be impacted")

		// prepare links/IP association by pre-creating links
		links, err := c.NetlinkAdapter.LinkList()
		if err != nil {
			c.log.Fatalf("can't list links: %w", err)
		}
		for _, link := range links {
			c.log.Infof("adding interface %s", link.Name())
			linkByIP[link.Name()] = []*net.IPNet{}
		}
	}

	return linkByIP, nil
}

func (c NetworkDisruptionConfig) AddDelay(hosts []string, port int, delay time.Duration) error {
	c.log.Info("auto-detecting interfaces to apply latency to...")

	parent := "root"

	linkByIP, err := c.getInterfacesByIP(hosts)
	if err != nil {
		c.log.Fatalw("can't get interfaces per IP listing: %w", err)
	}

	// for each link/ip association, add latency
	for linkName, ips := range linkByIP {
		clearTxQlen := false

		// retrieve link from name
		link, err := c.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			c.log.Fatalf("can't retrieve link %s: %w", linkName, err)
		}

		// if at least one IP has been specified, we need to create a prio qdisc to be able to apply
		// a filter and a delay only on traffic going to those IP
		if len(ips) > 0 {
			// set the tx qlen if not already set as it is required to create a prio qdisc without dropping
			// all the outgoing traffic
			// this qlen will be removed once the injection is done if it was not present before
			if link.TxQLen() == 0 {
				c.log.Infof("setting tx qlen for interface %s", link.Name())

				clearTxQlen = true

				if err := link.SetTxQLen(1000); err != nil {
					c.log.Fatalf("can't set tx queue length on interface %s: %w", link.Name(), err)
				}
			}

			// create a new qdisc for the given interface of type prio with 4 bands instead of 3
			// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP
			// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to add delay to
			// all the outgoing traffic
			parent = "1:4"
			priomap := [16]uint32{1, 2, 2, 2, 1, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1}

			if err := c.TrafficController.AddPrio(link.Name(), "root", 1, 4, priomap); err != nil {
				c.log.Fatalf("can't create a new qdisc for interface %s: %w", link.Name(), err)
			}
		}

		// add delay
		if err := c.TrafficController.AddDelay(link.Name(), parent, 0, delay); err != nil {
			c.log.Fatalf("can't add delay to the newly created qdisc for interface %s: %w", link.Name(), err)
		}

		// if only some hosts/ports are targeted, redirect the traffic to the extra band created earlier
		// where the delay is applied
		if len(ips) > 0 {
			for _, ip := range ips {
				if err := c.TrafficController.AddFilter(link.Name(), "1:0", 0, ip, port, "1:4"); err != nil {
					c.log.Fatalf("can't add a filter to interface %s: %w", link.Name(), err)
				}
			}
		}

		// reset the interface transmission queue length once filters have been created
		if clearTxQlen {
			c.log.Infof("clearing tx qlen for interface %s", link.Name())

			if err := link.SetTxQLen(0); err != nil {
				c.log.Fatalf("can't clear %s link transmission queue length: %w", link.Name(), err)
			}
		}
	}

	return nil
}

func (c NetworkDisruptionConfig) AddOutputLimit(hosts []string, port int, bytesPerSec uint) error {
	return nil
}

func (c NetworkDisruptionConfig) ClearAllQdiscs(hosts []string) error {
	linkByIP, err := c.getInterfacesByIP(hosts)
	if err != nil {
		c.log.Fatalf("can't get interfaces per IP map: %w", err)
	}

	for linkName := range linkByIP {
		c.log.Infof("clearing root qdisc for interface %s", linkName)

		// retrieve link from name
		link, err := c.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			c.log.Fatalf("can't retrieve link %s: %w", linkName, err)
		}

		// ensure qdisc isn't cleared before clearing it to avoid any tc error
		cleared, err := c.TrafficController.IsQdiscCleared(link.Name())
		if err != nil {
			c.log.Fatalf("can't ensure the %s link qdisc is cleared or not: %w", link.Name(), err)
		}

		// clear link qdisc if needed
		if !cleared {
			if err := c.TrafficController.ClearQdisc(link.Name()); err != nil {
				c.log.Fatalf("can't delete the %s link qdisc: %w", link.Name(), err)
			}
		} else {
			c.log.Infof("%s link qdisc is already cleared, skipping", link.Name())
		}
	}

	return nil
}

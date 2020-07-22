// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"fmt"
	"net"
	"time"

	"github.com/DataDog/chaos-controller/network"
	"go.uber.org/zap"
)

// linkOperation represents a tc operation on a single network interface combined with the parent to bind to and the handle identifier to use
type linkOperation func(network.NetlinkLink, string, uint32) error

// NetworkDisruptionConfig provides an interface for using the network traffic controller for new disruptions
type NetworkDisruptionConfig interface {
	AddNetem(delay time.Duration, drop int, corrupt int)
	AddOutputLimit(bytesPerSec uint)
	ApplyOperations() error
	ClearOperations() error
}

// NetworkDisruptionConfigStruct contains all needed drivers to create a network disruption using `tc`
type NetworkDisruptionConfigStruct struct {
	Log               *zap.SugaredLogger
	TrafficController network.TrafficController
	NetlinkAdapter    network.NetlinkAdapter
	DNSClient         network.DNSClient
	hosts             []string
	port              int
	operations        []linkOperation
}

// NewNetworkDisruptionConfig creates a new network disruption object using the given netlink, dns, etc.
func NewNetworkDisruptionConfig(logger *zap.SugaredLogger, tc network.TrafficController, netlink network.NetlinkAdapter, dns network.DNSClient, hosts []string, port int) NetworkDisruptionConfig {
	return &NetworkDisruptionConfigStruct{
		Log:               logger,
		TrafficController: tc,
		NetlinkAdapter:    netlink,
		DNSClient:         dns,
		hosts:             hosts,
		port:              port,
		operations:        []linkOperation{},
	}
}

// NewNetworkDisruptionConfigWithDefaults creates a new network disruption object using default netlink, dns, etc.
func NewNetworkDisruptionConfigWithDefaults(logger *zap.SugaredLogger, hosts []string, port int) NetworkDisruptionConfig {
	return NewNetworkDisruptionConfig(logger, network.NewTrafficController(logger), network.NewNetlinkAdapter(), network.NewDNSClient(), hosts, port)
}

// getInterfacesByIP returns the interfaces used to reach the given hosts
// if hosts is empty, all interfaces are returned
func (c *NetworkDisruptionConfigStruct) getInterfacesByIP(hosts []string) (map[string][]*net.IPNet, error) {
	linkByIP := map[string][]*net.IPNet{}

	if len(hosts) > 0 {
		c.Log.Info("auto-detecting interfaces to apply disruption to...")
		// resolve hosts
		ips, err := resolveHosts(c.DNSClient, hosts)
		if err != nil {
			return nil, fmt.Errorf("can't resolve given hosts: %w", err)
		}

		// get the association between IP and interfaces to know
		// which interfaces we have to inject disruption to
		for _, ip := range ips {
			// get routes for resolved destination IP
			routes, err := c.NetlinkAdapter.RoutesForIP(ip)
			if err != nil {
				return nil, fmt.Errorf("can't get route for IP %s: %w", ip.String(), err)
			}

			// for each route, get the related interface and add it to the association
			// between interfaces and IPs
			for _, route := range routes {
				c.Log.Infof("IP %s belongs to interface %s", ip.String(), route.Link().Name())

				// store association, initialize the map entry if not present yet
				if _, ok := linkByIP[route.Link().Name()]; !ok {
					linkByIP[route.Link().Name()] = []*net.IPNet{}
				}

				linkByIP[route.Link().Name()] = append(linkByIP[route.Link().Name()], ip)
			}
		}
	} else {
		c.Log.Info("no hosts specified, all interfaces will be impacted")

		// prepare links/IP association by pre-creating links
		links, err := c.NetlinkAdapter.LinkList()
		if err != nil {
			c.Log.Fatalf("can't list links: %w", err)
		}
		for _, link := range links {
			c.Log.Infof("adding interface %s", link.Name())
			linkByIP[link.Name()] = []*net.IPNet{}
		}
	}

	return linkByIP, nil
}

// ApplyOperations applies the added operations
// Depending on if hosts have been specified or not, this method will behave in a different way
// If no host is specified:
//  - the first operation will be attached to root
//  - other operations will be chained
// If at least one host is specified:
//  - a prio qdisc will be created and attached to root
//  - first operation will be attached to the last band of the prio qdisc
//  - other operations will be chained
//  - a filter will be created to redirect traffic related to the specified host(s) through the last prio band
func (c *NetworkDisruptionConfigStruct) ApplyOperations() error {
	c.Log.Info("auto-detecting interfaces to apply disruption to...")

	linkByIP, err := c.getInterfacesByIP(c.hosts)
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP listing: %w", err)
	}

	// for each link/ip association, add disruption
	for linkName, ips := range linkByIP {
		// default parent to root
		parent := "root"
		clearTxQlen := false

		// retrieve link from name
		link, err := c.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			return fmt.Errorf("can't retrieve link %s: %w", linkName, err)
		}

		// if at least one IP has been specified, we need to create a prio qdisc to be able to apply
		// a filter and a delay only on traffic going to those IP
		if len(ips) > 0 {
			// set the tx qlen if not already set as it is required to create a prio qdisc without dropping
			// all the outgoing traffic
			// this qlen will be removed once the injection is done if it was not present before
			if link.TxQLen() == 0 {
				c.Log.Infof("setting tx qlen for interface %s", link.Name())

				// set clear flag to true so we can clean up this once qdiscs are created
				clearTxQlen = true

				// set qlen
				if err := link.SetTxQLen(1000); err != nil {
					return fmt.Errorf("can't set tx queue length on interface %s: %w", link.Name(), err)
				}
			}

			// create a new qdisc for the given interface of type prio with 4 bands instead of 3
			// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP
			// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to apply disruptions to all the traffic for a bit of time
			priomap := [16]uint32{1, 2, 2, 2, 1, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1}

			if err := c.TrafficController.AddPrio(link.Name(), "root", 1, 4, priomap); err != nil {
				return fmt.Errorf("can't create a new qdisc for interface %s: %w", link.Name(), err)
			}

			// update parent reference since we created the prio qdisc
			// this parent points to the last prio band (each band having a separate class)
			parent = "1:4"
		}

		// set handle identifier for the first operation to apply
		// handle is 1 if no qdisc has been created yet, or 2 otherwise
		handle := uint32(1)
		if parent != "root" {
			handle++
		}

		// add operations
		for _, operation := range c.operations {
			if err := operation(link, parent, handle); err != nil {
				return fmt.Errorf("could not perform operation on newly created qdisc for interface %s: %w", link.Name(), err)
			}

			// update parent reference and handle identifier for the next operation
			// the next operation parent will be the current handle identifier
			// the next handle identifier is just an increment of the actual one
			parent = fmt.Sprintf("%d:", handle)
			handle++
		}

		// if only some hosts/ports are targeted, create a filter to redirect the traffic to the extra band created earlier
		if len(ips) > 0 {
			for _, ip := range ips {
				if err := c.TrafficController.AddFilter(link.Name(), "1:0", 0, ip, c.port, "1:4"); err != nil {
					return fmt.Errorf("can't add a filter to interface %s: %w", link.Name(), err)
				}
			}
		}

		// reset the interface transmission queue length once filters have been created (only if qlen has been set earlier)
		if clearTxQlen {
			c.Log.Infof("clearing tx qlen for interface %s", link.Name())

			if err := link.SetTxQLen(0); err != nil {
				return fmt.Errorf("can't clear %s link transmission queue length: %w", link.Name(), err)
			}
		}
	}

	return nil
}

// AddNetem adds network disruptions using the drivers in the NetworkDisruptionConfigStruct
func (c *NetworkDisruptionConfigStruct) AddNetem(delay time.Duration, drop int, corrupt int) {
	// closure which adds netem disruptions
	operation := func(link network.NetlinkLink, parent string, handle uint32) error {
		return c.TrafficController.AddNetem(link.Name(), parent, handle, delay, drop, corrupt)
	}

	c.operations = append(c.operations, operation)
}

// AddOutputLimit adds a network bandwidth disruption using the drivers in the NetworkDisruptionConfigStruct
func (c *NetworkDisruptionConfigStruct) AddOutputLimit(bytesPerSec uint) {
	// closure which adds a bandwidth limit
	operation := func(link network.NetlinkLink, parent string, handle uint32) error {
		return c.TrafficController.AddOutputLimit(link.Name(), parent, handle, bytesPerSec)
	}

	c.operations = append(c.operations, operation)
}

// ClearOperations removes all disruptions by clearing all custom qdiscs created for the given config struct (filters will be deleted as well)
func (c *NetworkDisruptionConfigStruct) ClearOperations() error {
	linkByIP, err := c.getInterfacesByIP(c.hosts)
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP map: %w", err)
	}

	for linkName := range linkByIP {
		c.Log.Infof("clearing root qdisc for interface %s", linkName)

		// retrieve link from name
		link, err := c.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			return fmt.Errorf("can't retrieve link %s: %w", linkName, err)
		}

		// ensure qdisc isn't cleared before clearing it to avoid any tc error
		cleared, err := c.TrafficController.IsQdiscCleared(link.Name())
		if err != nil {
			return fmt.Errorf("can't ensure the %s link qdisc is cleared or not: %w", link.Name(), err)
		}

		// clear link qdisc if needed
		if !cleared {
			if err := c.TrafficController.ClearQdisc(link.Name()); err != nil {
				return fmt.Errorf("can't delete the %s link qdisc: %w", link.Name(), err)
			}
		} else {
			c.Log.Infof("%s link qdisc is already cleared, skipping", link.Name())
		}
	}

	return nil
}

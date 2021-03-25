// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	flowEgress  = "egress"
	flowIngress = "ingress"
)

// linkOperation represents a tc operation on a single network interface combined with the parent to bind to and the handle identifier to use
type linkOperation func(network.NetlinkLink, string, uint32) error

// networkDisruptionInjector describes a network disruption
type networkDisruptionInjector struct {
	spec       v1beta1.NetworkDisruptionSpec
	config     NetworkDisruptionInjectorConfig
	operations []linkOperation
}

// NetworkDisruptionInjectorConfig contains all needed drivers to create a network disruption using `tc`
type NetworkDisruptionInjectorConfig struct {
	Config
	TrafficController network.TrafficController
	NetlinkAdapter    network.NetlinkAdapter
	DNSClient         network.DNSClient
}

// NewNetworkDisruptionInjector creates a NetworkDisruptionInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkDisruptionInjector(spec v1beta1.NetworkDisruptionSpec, config NetworkDisruptionInjectorConfig) Injector {
	if config.TrafficController == nil {
		config.TrafficController = network.NewTrafficController(config.Log, config.DryRun)
	}

	if config.NetlinkAdapter == nil {
		config.NetlinkAdapter = network.NewNetlinkAdapter()
	}

	if config.DNSClient == nil {
		config.DNSClient = network.NewDNSClient()
	}

	return networkDisruptionInjector{
		spec:       spec,
		config:     config,
		operations: []linkOperation{},
	}
}

// Inject injects the given network disruption into the given container
func (i networkDisruptionInjector) Inject() error {
	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	i.config.Log.Infow("adding network disruptions", "drop", i.spec.Drop, "duplicate", i.spec.Duplicate, "corrupt", i.spec.Corrupt, "delay", i.spec.Delay, "delayJitter", i.spec.DelayJitter, "bandwidthLimit", i.spec.BandwidthLimit)

	// add netem
	if i.spec.Delay > 0 || i.spec.Drop > 0 || i.spec.Corrupt > 0 || i.spec.Duplicate > 0 {
		delay := time.Duration(i.spec.Delay) * time.Millisecond
		delayJitter := time.Duration((float64(i.spec.DelayJitter)/100.0)*float64(i.spec.Delay)) * time.Millisecond
		i.addNetemOperation(delay, delayJitter, i.spec.Drop, i.spec.Corrupt, i.spec.Duplicate)
	}

	// add tbf
	if i.spec.BandwidthLimit > 0 {
		i.addOutputLimitOperation(uint(i.spec.BandwidthLimit))
	}

	// apply operations if any
	if len(i.operations) > 0 {
		if err := i.applyOperations(); err != nil {
			return fmt.Errorf("error applying tc operations: %w", err)
		}

		i.config.Log.Info("operations applied successfully")
	}

	i.config.Log.Info("editing pod net_cls cgroup to apply a classid to target container packets")

	// write classid to pod net_cls cgroup
	if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0x00020002"); err != nil {
		return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// Clean removes all the injected disruption in the given container
func (i networkDisruptionInjector) Clean() error {
	// enter container network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// defer the exit on return
	defer func() {
	}()

	if err := i.clearOperations(); err != nil {
		return fmt.Errorf("error clearing tc operations: %w", err)
	}

	// write default classid to pod net_cls cgroup if it still exists
	exists, err := i.config.Cgroup.Exists("net_cls")
	if err != nil {
		return fmt.Errorf("error checking if pod net_cls cgroup still exists: %w", err)
	}

	if exists {
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0x0"); err != nil {
			return fmt.Errorf("error reseting classid of pod net_cls cgroup: %w", err)
		}
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// getInterfacesByIP returns the interfaces used to reach the given hosts
// if hosts is empty, all interfaces are returned
func (i *networkDisruptionInjector) getInterfacesByIP(hosts []string) (map[string][]*net.IPNet, error) {
	linkByIP := map[string][]*net.IPNet{}

	if len(hosts) > 0 {
		i.config.Log.Infow("auto-detecting used interfaces to reach the given hosts", "hosts", hosts)

		// resolve hosts
		ips, err := resolveHosts(i.config.DNSClient, hosts)
		if err != nil {
			return nil, fmt.Errorf("can't resolve given hosts: %w", err)
		}

		// get the association between IP and interfaces to know
		// which interfaces we have to inject disruption to
		for _, ip := range ips {
			// get routes for resolved destination IP
			routes, err := i.config.NetlinkAdapter.RoutesForIP(ip)
			if err != nil {
				return nil, fmt.Errorf("can't get route for IP %s: %w", ip.String(), err)
			}

			// for each route, get the related interface and add it to the association
			// between interfaces and IPs
			for _, route := range routes {
				i.config.Log.Infof("IP %s belongs to interface %s", ip.String(), route.Link().Name())

				// store association, initialize the map entry if not present yet
				if _, ok := linkByIP[route.Link().Name()]; !ok {
					linkByIP[route.Link().Name()] = []*net.IPNet{}
				}

				linkByIP[route.Link().Name()] = append(linkByIP[route.Link().Name()], ip)
			}
		}
	} else {
		i.config.Log.Info("no hosts specified, all interfaces will be impacted")

		// prepare links/IP association by pre-creating links
		links, err := i.config.NetlinkAdapter.LinkList()
		if err != nil {
			return nil, fmt.Errorf("can't list links: %w", err)
		}

		for _, link := range links {
			i.config.Log.Infof("adding interface %s", link.Name())
			linkByIP[link.Name()] = []*net.IPNet{}
		}

		// explicitly add loopback interface
		i.config.Log.Infof("adding loopback interface")
		linkByIP["lo"] = []*net.IPNet{}
	}

	return linkByIP, nil
}

// ApplyOperations applies the added operations by building a tc tree
// Here's what happen on tc side:
//  - a first prio qdisc will be created and attached to root
//    - it'll be used to apply the first filter, filtering on packet IP destination, source/destination ports and protocol
//  - a second prio qdisc will be created and attached to the first one
//    - it'll be used to apply the second filter, filtering on packet classid to identify packets coming from the targeted process
//  - operations will be chained to the second band of the second prio qdisc
//  - a cgroup filter will be created to classify packets according to their classid (if any)
//  - a filter will be created to redirect traffic related to the specified host(s) through the last prio band
//    - if no host, port or protocol is specified, a filter redirecting all the traffic (0.0.0.0/0) to the disrupted band will be created
//  - a last filter will be created to redirect traffic related to the local node through a not disrupted band
//
// Here's the tc tree representation:
// root (1:) <-- prio qdisc with 4 bands with a filter classifying packets matching the given dst ip, src/dst ports and protocol with class 1:4
//   |- (1:1) <-- first band
//   |- (1:2) <-- second band
//   |- (1:3) <-- third band
//   |- (1:4) <-- fourth band
//     |- (2:) <-- prio qdisc with 2 bands with a cgroup filter to classify packets according to their classid (packets with classid 2:2 will be affected by operations)
//       |- (2:1) <-- first band
//       |- (2:2) <-- second band
//         |- (3:) <-- first operation
//           |- (4:) <-- second operation
//             ...
func (i *networkDisruptionInjector) applyOperations() error {
	// retrieve the default route information
	defaultRoute, err := i.config.NetlinkAdapter.DefaultRoute()
	if err != nil {
		return fmt.Errorf("error getting the default route: %w", err)
	}

	i.config.Log.Infof("detected default gateway IP %s on interface %s", defaultRoute.Gateway().String(), defaultRoute.Link().Name())

	// get the targeted pod node IP from the environment variable
	hostIP, ok := os.LookupEnv(env.InjectorTargetPodHostIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the target pod node IP", env.InjectorTargetPodHostIP)
	}

	i.config.Log.Infof("target pod node IP is %s", hostIP)

	hostIPNet := &net.IPNet{
		IP:   net.ParseIP(hostIP),
		Mask: net.CIDRMask(32, 32),
	}

	// get routes going to this node IP to add a filter excluding this IP from the disruptions later
	// it is used to allow the node to reach the pod even with disruptions applied
	hostIPRoutes, err := i.config.NetlinkAdapter.RoutesForIP(hostIPNet)
	if err != nil {
		return fmt.Errorf("error getting target pod node IP routes: %w", err)
	}

	// get the interfaces per IP map looking like:
	// "eth0" => [10.0.0.0/8, 192.168.0.1/32]
	// "eth1" => [172.16.0.0/16, ...]
	linkByIP, err := i.getInterfacesByIP(i.spec.Hosts)
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP listing: %w", err)
	}

	// allow kubelet -> apiserver communications
	// resolve the kubernetes.default service created at cluster bootstrap and owning the apiserver cluster IP
	apiservers, err := i.getInterfacesByIP([]string{"kubernetes.default"})
	if err != nil {
		return fmt.Errorf("error resolving apiservers service IP: %w", err)
	}

	if len(apiservers) == 0 {
		return fmt.Errorf("could not resolve kubernetes.default service IP")
	}

	// for each link/ip association, add disruption
	for linkName, ips := range linkByIP {
		// retrieve link from name
		link, err := i.config.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			return fmt.Errorf("can't retrieve link %s: %w", linkName, err)
		}

		// set the tx qlen if not already set as it is required to create a prio qdisc without dropping
		// all the outgoing traffic
		// this qlen will be removed once the injection is done if it was not present before
		if link.TxQLen() == 0 {
			i.config.Log.Infof("setting tx qlen for interface %s", link.Name())

			// set qlen
			if err := link.SetTxQLen(1000); err != nil {
				return fmt.Errorf("can't set tx queue length on interface %s: %w", link.Name(), err)
			}

			// defer the tx qlen clear
			defer func(link network.NetlinkLink) {
				i.config.Log.Infof("clearing tx qlen for interface %s", link.Name())

				if err := link.SetTxQLen(0); err != nil {
					i.config.Log.Errorw("can't clear %s link transmission queue length: %w", link.Name(), err)
				}
			}(link)
		}

		// create a new qdisc for the given interface of type prio with 4 bands instead of 3
		// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP
		// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to apply disruptions to all the traffic for a bit of time
		priomap := [16]uint32{1, 2, 2, 2, 1, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1}

		if err := i.config.TrafficController.AddPrio(link.Name(), "root", 1, 4, priomap); err != nil {
			return fmt.Errorf("can't create a new qdisc for interface %s: %w", link.Name(), err)
		}

		// parent 1:4 refers to the 4th band of the prio qdisc
		// handle starts from 2 because 1 is used by the prio qdisc
		parent := "1:4"
		handle := uint32(2)

		// if the disruption is at pod level, create a second qdisc to filter packets coming from
		// this specific pod processes only
		if i.config.Level == chaostypes.DisruptionLevelPod {
			// create second prio with only 2 bands to filter traffic with a specific classid
			if err := i.config.TrafficController.AddPrio(link.Name(), "1:4", 2, 2, [16]uint32{}); err != nil {
				return fmt.Errorf("can't create a new qdisc for interface %s: %w", link.Name(), err)
			}

			// create cgroup filter
			if err := i.config.TrafficController.AddCgroupFilter(link.Name(), "2:0", 2); err != nil {
				return fmt.Errorf("can't create the cgroup filter for interface %s: %w", link.Name(), err)
			}
			// parent 2:2 refers to the 2nd band of the 2nd prio qdisc
			// handle starts from 3 because 1 and 2 are used by the 2 prio qdiscs
			parent = "2:2"
			handle = uint32(3)
		}

		// add operations
		for _, operation := range i.operations {
			if err := operation(link, parent, handle); err != nil {
				return fmt.Errorf("could not perform operation on newly created qdisc for interface %s: %w", link.Name(), err)
			}

			// update parent reference and handle identifier for the next operation
			// the next operation parent will be the current handle identifier
			// the next handle identifier is just an increment of the actual one
			parent = fmt.Sprintf("%d:", handle)
			handle++
		}

		// handle flow direction
		// if flow is egress, filter will be on destination port
		// if flow is ingress, filter will be on source port
		var srcPort, dstPort int

		switch i.spec.Flow {
		case flowEgress:
			dstPort = i.spec.Port
		case flowIngress:
			srcPort = i.spec.Port
		default:
			return fmt.Errorf("unsupported flow: %s", i.spec.Flow)
		}

		// if some hosts are targeted, create one filter per host to redirect the traffic to the disrupted band
		// otherwise, create a filter redirecting all the traffic (0.0.0.0/0) using the given port and protocol to the disrupted band
		if len(ips) > 0 {
			for _, ip := range ips {
				if err := i.config.TrafficController.AddFilter(link.Name(), "1:0", 0, nil, ip, srcPort, dstPort, i.spec.Protocol, "1:4"); err != nil {
					return fmt.Errorf("can't add a filter to interface %s: %w", link.Name(), err)
				}
			}
		} else {
			_, nullIP, _ := net.ParseCIDR("0.0.0.0/0")
			if err := i.config.TrafficController.AddFilter(link.Name(), "1:0", 0, nil, nullIP, srcPort, dstPort, i.spec.Protocol, "1:4"); err != nil {
				return fmt.Errorf("can't add a filter to interface %s: %w", link.Name(), err)
			}
		}
	}

	// the following lines are used to exclude some critical packets from any disruption such as health check probes
	// depending on the network configuration, only one of those filters can be useful but we must add all of them
	// those filters are only added if the related interface has been impacted by a disruption so far
	// NOTE: those filters must be added after every other filters applied to the interface so they are used first
	if i.config.Level == chaostypes.DisruptionLevelPod {
		// this filter allows the pod to communicate with the default route gateway IP
		gatewayIP := &net.IPNet{
			IP:   defaultRoute.Gateway(),
			Mask: net.CIDRMask(32, 32),
		}

		if _, found := linkByIP[defaultRoute.Link().Name()]; found {
			if err := i.config.TrafficController.AddFilter(defaultRoute.Link().Name(), "1:0", 0, nil, gatewayIP, 0, 0, "", "1:1"); err != nil {
				return fmt.Errorf("can't add the default route gateway IP filter: %w", err)
			}
		}

		// this filter allows the pod to communicate with the node IP
		for _, hostIPRoute := range hostIPRoutes {
			if _, found := linkByIP[hostIPRoute.Link().Name()]; found {
				if err := i.config.TrafficController.AddFilter(hostIPRoute.Link().Name(), "1:0", 0, nil, hostIPNet, 0, 0, "", "1:1"); err != nil {
					return fmt.Errorf("can't add the target pod node IP filter: %w", err)
				}
			}
		}
	} else if i.config.Level == chaostypes.DisruptionLevelNode {
		if _, found := linkByIP[defaultRoute.Link().Name()]; found {
			// allow SSH connections (port 22/tcp)
			if err := i.config.TrafficController.AddFilter(defaultRoute.Link().Name(), "1:0", 0, nil, nil, 22, 0, "tcp", "1:1"); err != nil {
				return fmt.Errorf("error adding filter allowing SSH connections: %w", err)
			}

			// allow cloud provider health checks (arp)
			if err := i.config.TrafficController.AddFilter(defaultRoute.Link().Name(), "1:0", 0, nil, nil, 0, 0, "arp", "1:1"); err != nil {
				return fmt.Errorf("error adding filter allowing cloud providers health checks (ARP packets): %w", err)
			}
		}

		// allow all communications to this (eventually these) IP
		for linkName, apiserverIPs := range apiservers {
			if _, found := linkByIP[linkName]; found {
				link, err := i.config.NetlinkAdapter.LinkByName(linkName)
				if err != nil {
					return fmt.Errorf("error getting %s link: %w", linkName, err)
				}

				for _, ip := range apiserverIPs {
					if err := i.config.TrafficController.AddFilter(link.Name(), "1:0", 0, nil, ip, 0, 0, "", "1:1"); err != nil {
						return fmt.Errorf("error adding filter allowing apiserver communications: %w", err)
					}
				}
			}
		}
	}

	return nil
}

// AddNetem adds network disruptions using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addNetemOperation(delay, delayJitter time.Duration, drop int, corrupt int, duplicate int) {
	// closure which adds netem disruptions
	operation := func(link network.NetlinkLink, parent string, handle uint32) error {
		return i.config.TrafficController.AddNetem(link.Name(), parent, handle, delay, delayJitter, drop, corrupt, duplicate)
	}

	i.operations = append(i.operations, operation)
}

// AddOutputLimit adds a network bandwidth disruption using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addOutputLimitOperation(bytesPerSec uint) {
	// closure which adds a bandwidth limit
	operation := func(link network.NetlinkLink, parent string, handle uint32) error {
		return i.config.TrafficController.AddOutputLimit(link.Name(), parent, handle, bytesPerSec)
	}

	i.operations = append(i.operations, operation)
}

// clearOperations removes all disruptions by clearing all custom qdiscs created for the given config struct (filters will be deleted as well)
func (i *networkDisruptionInjector) clearOperations() error {
	linkByIP, err := i.getInterfacesByIP(i.spec.Hosts)
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP map: %w", err)
	}

	for linkName := range linkByIP {
		i.config.Log.Infof("clearing root qdisc for interface %s", linkName)

		// retrieve link from name
		link, err := i.config.NetlinkAdapter.LinkByName(linkName)
		if err != nil {
			return fmt.Errorf("can't retrieve link %s: %w", linkName, err)
		}

		// clear link qdisc if needed
		if err := i.config.TrafficController.ClearQdisc(link.Name()); err != nil {
			return fmt.Errorf("can't delete the %s link qdisc: %w", link.Name(), err)
		}
	}

	return nil
}

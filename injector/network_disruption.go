// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// linkOperation represents a tc operation on a set of network interfaces combined with the parent to bind to and the handle identifier to use
type linkOperation func([]string, string, uint32) error

// networkDisruptionService describes a parsed Kubernetes service, representing an (ip, port, protocol) tuple
type networkDisruptionService struct {
	ip       *net.IPNet
	port     int
	protocol string
}

func (n networkDisruptionService) String() string {
	ip := ""
	if n.ip != nil {
		ip = n.ip.String()
	}
	return fmt.Sprintf("ip=%s; port=%d; protocol=%s", ip, n.port, n.protocol)
}

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

		var delayJitter time.Duration

		// add a 10% delayJitter to delay by default if not specified
		if i.spec.DelayJitter == 0 {
			delayJitter = time.Duration(float64(i.spec.Delay)*0.1) * time.Millisecond
		} else {
			// convert delayJitter into a percentage then multiply that with delay to get correct percentage of delay
			delayJitter = time.Duration((float64(i.spec.DelayJitter)/100.0)*float64(i.spec.Delay)) * time.Millisecond
		}

		delayJitter = time.Duration(math.Max(float64(delayJitter), float64(time.Millisecond)))

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

// applyOperations applies the added operations by building a tc tree
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
	// get interfaces
	links, err := i.config.NetlinkAdapter.LinkList()
	if err != nil {
		return fmt.Errorf("error listing interfaces: %w", err)
	}

	// build a map of link name and link interface
	interfaces := []string{}
	for _, link := range links {
		interfaces = append(interfaces, link.Name())
	}

	// retrieve the default route information
	defaultRoutes, err := i.config.NetlinkAdapter.DefaultRoutes()
	if err != nil {
		return fmt.Errorf("error getting the default route: %w", err)
	}

	i.config.Log.Infof("detected default gateway IPs %s", defaultRoutes)

	// get the targeted pod node IP from the environment variable
	nodeIP, ok := os.LookupEnv(env.InjectorTargetPodHostIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the target pod node IP", env.InjectorTargetPodHostIP)
	}

	i.config.Log.Infof("target pod node IP is %s", nodeIP)

	nodeIPNet := &net.IPNet{
		IP:   net.ParseIP(nodeIP),
		Mask: net.CIDRMask(32, 32),
	}

	// create cloud provider metadata service ipnet
	metadataIPNet := &net.IPNet{
		IP:   net.ParseIP("169.254.169.254"),
		Mask: net.CIDRMask(32, 32),
	}

	// set the tx qlen if not already set as it is required to create a prio qdisc without dropping
	// all the outgoing traffic
	// this qlen will be removed once the injection is done if it was not present before
	for _, link := range links {
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
	}

	// create a new qdisc for the given interface of type prio with 4 bands instead of 3
	// we keep the default priomap, the extra band will be used to filter traffic going to the specified IP
	// we only create this qdisc if we want to target traffic going to some hosts only, it avoids to apply disruptions to all the traffic for a bit of time
	priomap := [16]uint32{1, 2, 2, 2, 1, 2, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1}

	if err := i.config.TrafficController.AddPrio(interfaces, "root", 1, 4, priomap); err != nil {
		return fmt.Errorf("can't create a new qdisc: %w", err)
	}

	// parent 1:4 refers to the 4th band of the prio qdisc
	// handle starts from 2 because 1 is used by the prio qdisc
	parent := "1:4"
	handle := uint32(2)

	// if the disruption is at pod level and there's no handler to notify,
	// create a second qdisc to filter packets coming from this specific pod processes only
	// if the disruption is applied on init, we consider that some more containers may be created within
	// the pod so we can't scope the disruption to a specific set of containers
	if i.config.Level == chaostypes.DisruptionLevelPod && !i.config.OnInit {
		// create second prio with only 2 bands to filter traffic with a specific classid
		if err := i.config.TrafficController.AddPrio(interfaces, "1:4", 2, 2, [16]uint32{}); err != nil {
			return fmt.Errorf("can't create a new qdisc: %w", err)
		}

		// create cgroup filter
		if err := i.config.TrafficController.AddCgroupFilter(interfaces, "2:0", 2); err != nil {
			return fmt.Errorf("can't create the cgroup filter: %w", err)
		}
		// parent 2:2 refers to the 2nd band of the 2nd prio qdisc
		// handle starts from 3 because 1 and 2 are used by the 2 prio qdiscs
		parent = "2:2"
		handle = uint32(3)
	}

	// add operations
	for _, operation := range i.operations {
		if err := operation(interfaces, parent, handle); err != nil {
			return fmt.Errorf("could not perform operation on newly created qdisc: %w", err)
		}

		// update parent reference and handle identifier for the next operation
		// the next operation parent will be the current handle identifier
		// the next handle identifier is just an increment of the actual one
		parent = fmt.Sprintf("%d:", handle)
		handle++
	}

	// create tc filters depending on the given hosts to match
	// redirect all packets of all interfaces if no host is given
	if len(i.spec.Hosts) == 0 && len(i.spec.Services) == 0 {
		_, nullIP, _ := net.ParseCIDR("0.0.0.0/0")
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, nil, nullIP, 0, 0, "", "1:4"); err != nil {
			return fmt.Errorf("can't add a filter: %w", err)
		}
	} else {
		// apply filters for given hosts
		if err := i.addFiltersForHosts(interfaces, i.spec.Hosts, "1:4"); err != nil {
			return fmt.Errorf("error adding filters for given hosts: %w", err)
		}

		// apply filters for given services
		if err := i.addFiltersForServices(interfaces, "1:4"); err != nil {
			return fmt.Errorf("error adding filters for given services: %w", err)
		}
	}

	// the following lines are used to exclude some critical packets from any disruption such as health check probes
	// depending on the network configuration, only one of those filters can be useful but we must add all of them
	// those filters are only added if the related interface has been impacted by a disruption so far
	// NOTE: those filters must be added after every other filters applied to the interface so they are used first
	if i.config.Level == chaostypes.DisruptionLevelPod {
		// this filter allows the pod to communicate with the default route gateway IP
		for _, defaultRoute := range defaultRoutes {
			gatewayIP := &net.IPNet{
				IP:   defaultRoute.Gateway(),
				Mask: net.CIDRMask(32, 32),
			}

			if err := i.config.TrafficController.AddFilter([]string{defaultRoute.Link().Name()}, "1:0", 0, nil, gatewayIP, 0, 0, "", "1:1"); err != nil {
				return fmt.Errorf("can't add the default route gateway IP filter: %w", err)
			}
		}

		// this filter allows the pod to communicate with the node IP
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, nil, nodeIPNet, 0, 0, "", "1:1"); err != nil {
			return fmt.Errorf("can't add the target pod node IP filter: %w", err)
		}
	} else if i.config.Level == chaostypes.DisruptionLevelNode {
		// GENERIC SAFEGUARDS
		// allow SSH connections on all interfaces (port 22/tcp)
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, nil, nil, 22, 0, "tcp", "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing SSH connections: %w", err)
		}

		// CLOUD PROVIDER SPECIFIC SAFEGUARDS
		// allow cloud provider health checks on all interfaces(arp)
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, nil, nil, 0, 0, "arp", "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing cloud providers health checks (ARP packets): %w", err)
		}

		// allow cloud provider metadata service communication
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, nil, metadataIPNet, 0, 0, "", "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing cloud providers health checks (ARP packets): %w", err)
		}
	}

	// add filters for allowed hosts
	if err := i.addFiltersForHosts(interfaces, i.spec.AllowedHosts, "1:1"); err != nil {
		return fmt.Errorf("error adding filter for allowed hosts: %w", err)
	}

	return nil
}

// getServices parses the Kubernetes services in the disruption spec and returns a set of (ip, port, protocol) tuples
func (i *networkDisruptionInjector) getServices() ([]networkDisruptionService, error) {
	allServices := []networkDisruptionService{}

	for _, serviceSpec := range i.spec.Services {
		// retrieve serviceSpec
		k8sService, err := i.config.K8sClient.CoreV1().Services(serviceSpec.Namespace).Get(context.Background(), serviceSpec.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting the given kubernetes serviceSpec (%s/%s): %w", serviceSpec.Namespace, serviceSpec.Name, err)
		}

		// retrieve endpoints from selector
		endpoints, err := i.config.K8sClient.CoreV1().Pods(serviceSpec.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromValidatedSet(k8sService.Spec.Selector).String(),
		})
		if err != nil {
			return nil, fmt.Errorf("error getting the given kubernetes serviceSpec (%s/%s) endpoints: %w", serviceSpec.Namespace, serviceSpec.Name, err)
		}

		services := []networkDisruptionService{}

		// retrieve endpoints IPs
		for _, endpoint := range endpoints.Items {
			// compute endpoint IP (pod IP)
			_, endpointIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", endpoint.Status.PodIP))

			for _, port := range k8sService.Spec.Ports {
				services = append(services, networkDisruptionService{
					ip:       endpointIP,
					port:     int(port.TargetPort.IntVal),
					protocol: string(port.Protocol),
				})
			}
		}

		// compute service IP
		_, serviceIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", k8sService.Spec.ClusterIP))

		for _, port := range k8sService.Spec.Ports {
			services = append(services, networkDisruptionService{
				ip:       serviceIP,
				port:     int(port.Port),
				protocol: string(port.Protocol),
			})
		}

		endpointInfo := ""
		for _, service := range services {
			allServices = append(allServices, service)
			endpointInfo = fmt.Sprintf("%s{%s}, ", endpointInfo, service)
		}

		i.config.Log.Infow("found serviceSpec endpoints", "serviceSpec", serviceSpec.Name, "endpoints", endpointInfo)
	}

	return allServices, nil
}

// addFiltersForServices creates tc filters on given interfaces for services in disruption spec classifying matching packets in the given flowid
func (i *networkDisruptionInjector) addFiltersForServices(interfaces []string, flowid string) error {
	// apply filters for given services
	services, err := i.getServices()
	if err != nil {
		return fmt.Errorf("error getting services IPs and ports: %w", err)
	}

	for _, service := range services {
		// handle flow direction
		var (
			srcPort, dstPort int
			srcIP, dstIP     *net.IPNet
		)

		switch i.spec.Flow {
		case v1beta1.FlowEgress:
			dstPort = service.port
			dstIP = service.ip
		case v1beta1.FlowIngress:
			srcPort = service.port
			srcIP = service.ip
		}

		// create tc filter
		if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, srcIP, dstIP, srcPort, dstPort, service.protocol, flowid); err != nil {
			return fmt.Errorf("can't add a filter: %w", err)
		}
	}

	return nil
}

// addFiltersForHosts creates tc filters on given interfaces for given hosts classifying matching packets in the given flowid
func (i *networkDisruptionInjector) addFiltersForHosts(interfaces []string, hosts []v1beta1.NetworkDisruptionHostSpec, flowid string) error {
	for _, host := range hosts {
		// resolve given hosts if needed
		ips, err := resolveHost(i.config.DNSClient, host.Host)
		if err != nil {
			return fmt.Errorf("error resolving given host %s: %w", host.Host, err)
		}

		for _, ip := range ips {
			// handle flow direction
			var (
				srcPort, dstPort int
				srcIP, dstIP     *net.IPNet
			)

			switch i.spec.Flow {
			case v1beta1.FlowEgress:
				dstPort = host.Port
				dstIP = ip
			case v1beta1.FlowIngress:
				srcPort = host.Port
				srcIP = ip
			}

			// create tc filter
			if err := i.config.TrafficController.AddFilter(interfaces, "1:0", 0, srcIP, dstIP, srcPort, dstPort, host.Protocol, flowid); err != nil {
				return fmt.Errorf("error adding filte for host %s: %w", host.Host, err)
			}
		}
	}

	return nil
}

// AddNetem adds network disruptions using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addNetemOperation(delay, delayJitter time.Duration, drop int, corrupt int, duplicate int) {
	// closure which adds netem disruptions
	operation := func(interfaces []string, parent string, handle uint32) error {
		return i.config.TrafficController.AddNetem(interfaces, parent, handle, delay, delayJitter, drop, corrupt, duplicate)
	}

	i.operations = append(i.operations, operation)
}

// AddOutputLimit adds a network bandwidth disruption using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addOutputLimitOperation(bytesPerSec uint) {
	// closure which adds a bandwidth limit
	operation := func(interfaces []string, parent string, handle uint32) error {
		return i.config.TrafficController.AddOutputLimit(interfaces, parent, handle, bytesPerSec)
	}

	i.operations = append(i.operations, operation)
}

// clearOperations removes all disruptions by clearing all custom qdiscs created for the given config struct (filters will be deleted as well)
func (i *networkDisruptionInjector) clearOperations() error {
	i.config.Log.Infof("clearing root qdiscs")

	// get all interfaces
	links, err := i.config.NetlinkAdapter.LinkList()
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP map: %w", err)
	}

	// clear all interfaces root qdisc so it gets back to default
	interfaces := []string{}
	for _, link := range links {
		interfaces = append(interfaces, link.Name())
	}

	// clear link qdisc if needed
	if err := i.config.TrafficController.ClearQdisc(interfaces); err != nil {
		return fmt.Errorf("error deleting root qdisc: %w", err)
	}

	return nil
}

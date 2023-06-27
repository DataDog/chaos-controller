// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

// linkOperation represents a tc operation on a set of network interfaces combined with the parent to bind to and the handle identifier to use
type linkOperation func([]string, string, string) error

// networkDisruptionService describes a parsed Kubernetes service, representing an (ip, port, protocol) tuple
type networkDisruptionService struct {
	ip       *net.IPNet
	port     int
	protocol v1.Protocol
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
	cancel     []context.CancelFunc
}

// NetworkDisruptionInjectorConfig contains all needed drivers to create a network disruption using `tc`
type NetworkDisruptionInjectorConfig struct {
	Config
	TrafficController   network.TrafficController
	IPTables            network.IPTables
	NetlinkAdapter      network.NetlinkAdapter
	DNSClient           network.DNSClient
	HostResolveInterval time.Duration
}

// tcServiceFilter describes a tc filter, representing the service filtered and its priority
type tcServiceFilter struct {
	service  networkDisruptionService
	priority uint32 // one priority per tc filters applied, the priority is the same for all interfaces
}

// tcFilter describes a tc filter
type tcFilter struct {
	ip       *net.IPNet
	priority uint32 // one priority per tc filters applied, the priority is the same for all interfaces
}

// serviceWatcher
type serviceWatcher struct {
	// information about the service watched
	watchedServiceSpec   v1beta1.NetworkDisruptionServiceSpec
	servicePorts         []v1.ServicePort
	labelServiceSelector string

	// filters and watcher for the pods related to the service watched
	kubernetesPodEndpointsWatcher <-chan watch.Event
	tcFiltersFromPodEndpoints     []tcServiceFilter
	podsWithoutIPs                []string
	podsResourceVersion           string

	// filters and watcher for the kubernetes service watched
	kubernetesServiceWatcher       <-chan watch.Event
	tcFiltersFromNamespaceServices []tcServiceFilter
	servicesResourceVersion        string
}

type hostsWatcher struct {
	// The only identifying info we need are the ip and filter priority
	hostFilterMap map[v1beta1.NetworkDisruptionHostSpec][]*tcFilter
}

// NewNetworkDisruptionInjector creates a NetworkDisruptionInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkDisruptionInjector(spec v1beta1.NetworkDisruptionSpec, config NetworkDisruptionInjectorConfig) (Injector, error) {
	var err error

	if config.IPTables == nil {
		config.IPTables, err = network.NewIPTables(config.Log, config.Disruption.DryRun)
		if err != nil {
			return nil, err
		}
	}

	if config.TrafficController == nil {
		config.TrafficController = network.NewTrafficController(config.Log, config.Disruption.DryRun)
	}

	if config.NetlinkAdapter == nil {
		config.NetlinkAdapter = network.NewNetlinkAdapter()
	}

	if config.DNSClient == nil {
		config.DNSClient = network.NewDNSClient()
	}

	return &networkDisruptionInjector{
		spec:       spec,
		config:     config,
		operations: []linkOperation{},
	}, nil
}

func (i *networkDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindNetworkDisruption
}

// Inject injects the given network disruption into the given container
func (i *networkDisruptionInjector) Inject() error {
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

		i.config.Log.Debug("operations applied successfully")
	}

	// add a conntrack reference to enable it
	// it consists of adding a noop iptables rule loading the conntrack module so it enables connection tracking in the targeted network namespace
	// cf. https://thermalcircle.de/doku.php?id=blog:linux:connection_tracking_1_modules_and_hooks for more information on how conntrack works outside of the main network namespace
	if err := i.config.IPTables.LogConntrack(); err != nil {
		return fmt.Errorf("error injecting the conntrack reference iptables rule: %w", err)
	}

	// mark all packets created by the targeted container with the classifying mark
	if i.config.Disruption.Level == types.DisruptionLevelPod && !i.config.Disruption.OnInit {
		if i.config.Cgroup.IsCgroupV2() { // cgroup v2 can rely on the single cgroup hierarchy relative path to mark packets
			if err := i.config.IPTables.MarkCgroupPath(i.config.Cgroup.RelativePath(""), types.InjectorCgroupClassID); err != nil {
				return fmt.Errorf("error injecting packet marking iptables rule: %w", err)
			}
		} else { // cgroup v1 needs to mark packets through the net_cls cgroup controller of the container
			if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", types.InjectorCgroupClassID); err != nil {
				return fmt.Errorf("error injecting packet marking in net_cls cgroup: %w", err)
			}

			if err := i.config.IPTables.MarkClassID(types.InjectorCgroupClassID, types.InjectorCgroupClassID); err != nil {
				return fmt.Errorf("error injecting packet marking iptables rule: %w", err)
			}
		}
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

func (i *networkDisruptionInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes all the injected disruption in the given container
func (i *networkDisruptionInjector) Clean() error {
	// stop all background watchers now
	if i.cancel != nil {
		for _, cancelFunc := range i.cancel {
			cancelFunc()
		}
		i.cancel = nil
	}

	// enter container network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	if err := i.clearOperations(); err != nil {
		return fmt.Errorf("error clearing tc operations: %w", err)
	}

	// remove the conntrack reference to disable conntrack in the network namespace
	if err := i.config.IPTables.Clear(); err != nil {
		return fmt.Errorf("error cleaning iptables rules and chain: %w", err)
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	// remove the net_cls classid used for cgroup v1
	if !i.config.Cgroup.IsCgroupV2() {
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0"); err != nil {
			if os.IsNotExist(err) {
				i.config.Log.Warnw("unable to find target container's net_cls.classid file, we will assume we cannot find the cgroup path because it is gone", "targetContainerID", i.config.TargetContainer.ID(), "error", err)
				return nil
			}

			return fmt.Errorf("error cleaning net_cls classid: %w", err)
		}
	}

	return nil
}

// applyOperations applies the added operations by building a tc tree
// Here's what happen on tc side:
//   - a first prio qdisc will be created and attached to root
//     it'll be used to apply the first filter, filtering on packet IP destination, source/destination ports and protocol
//   - a second prio qdisc will be created and attached to the first one
//     it'll be used to apply the second filter, filtering on packet mark to identify packets coming from the targeted process
//   - operations will be chained to the second band of the second prio qdisc
//   - an fw filter will be created to classify packets according to their mark (if any)
//   - a filter will be created to redirect traffic related to the specified host(s) through the last prio band
//     if no host, port or protocol is specified, a filter redirecting all the traffic (0.0.0.0/0) to the disrupted band will be created
//   - a last filter will be created to redirect traffic related to the local node through a not disrupted band
//
// Here's the tc tree representation:
// root (1:) <-- prio qdisc with 4 bands with a filter classifying packets matching the given dst ip, src/dst ports and protocol with class 1:4
//
//	|- (1:1) <-- first band
//	|- (1:2) <-- second band
//	|- (1:3) <-- third band
//	|- (1:4) <-- fourth band
//	  |- (2:) <-- prio qdisc with 2 bands with an fw filter to classify packets according to their mark (packets with mark 2:2 will be affected by operations)
//	    |- (2:1) <-- first band
//	    |- (2:2) <-- second band
//	      |- (3:) <-- first operation
//	        |- (4:) <-- second operation
//	          ...
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

	if err := i.config.TrafficController.AddPrio(interfaces, "root", "1:", 4, priomap); err != nil {
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
	if i.config.Disruption.Level == types.DisruptionLevelPod && !i.config.Disruption.OnInit {
		// create second prio with only 2 bands to filter traffic with a specific mark
		if err := i.config.TrafficController.AddPrio(interfaces, "1:4", "2:", 2, [16]uint32{}); err != nil {
			return fmt.Errorf("can't create a new qdisc: %w", err)
		}

		// create fw filter to classify packets based on their mark
		if err := i.config.TrafficController.AddFwFilter(interfaces, "2:0", types.InjectorCgroupClassID, "2:2"); err != nil {
			return fmt.Errorf("can't create the fw filter: %w", err)
		}
		// parent 2:2 refers to the 2nd band of the 2nd prio qdisc
		// handle starts from 3 because 1 and 2 are used by the 2 prio qdiscs
		parent = "2:2"
		handle = uint32(3)
	}

	// add operations
	for _, operation := range i.operations {
		if err := operation(interfaces, parent, fmt.Sprintf("%d:", handle)); err != nil {
			return fmt.Errorf("could not perform operation on newly created qdisc: %w", err)
		}

		// update parent reference and handle identifier for the next operation
		// the next operation parent will be the current handle identifier
		// the next handle identifier is just an increment of the actual one
		parent = fmt.Sprintf("%d:", handle)
		handle++
	}

	// the following lines are used to exclude some critical packets from any disruption such as health check probes
	// depending on the network configuration, only one of those filters can be useful but we must add all of them
	// those filters are only added if the related interface has been impacted by a disruption so far
	// NOTE: those filters must be added after every other filters applied to the interface so they are used first
	if i.config.Disruption.Level == types.DisruptionLevelPod {
		// this filter allows the pod to communicate with the default route gateway IP
		for _, defaultRoute := range defaultRoutes {
			gatewayIP := &net.IPNet{
				IP:   defaultRoute.Gateway(),
				Mask: net.CIDRMask(32, 32),
			}

			if _, err := i.config.TrafficController.AddFilter([]string{defaultRoute.Link().Name()}, "1:0", "", nil, gatewayIP, 0, 0, network.TCP, network.ConnStateUndefined, "1:1"); err != nil {
				return fmt.Errorf("can't add the default route gateway IP filter: %w", err)
			}
		}

		// this filter allows the pod to communicate with the node IP
		if _, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, nodeIPNet, 0, 0, network.TCP, network.ConnStateUndefined, "1:1"); err != nil {
			return fmt.Errorf("can't add the target pod node IP filter: %w", err)
		}
	} else if i.config.Disruption.Level == types.DisruptionLevelNode {
		// GENERIC SAFEGUARDS
		// allow SSH connections on all interfaces (port 22/tcp)
		if _, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, nil, 22, 0, network.TCP, network.ConnStateUndefined, "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing SSH connections: %w", err)
		}

		// CLOUD PROVIDER SPECIFIC SAFEGUARDS
		// allow cloud provider health checks on all interfaces(arp)
		if _, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, nil, 0, 0, network.ARP, network.ConnStateUndefined, "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing cloud providers health checks (ARP packets): %w", err)
		}

		// allow cloud provider metadata service communication
		if _, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, metadataIPNet, 0, 0, network.TCP, network.ConnStateUndefined, "1:1"); err != nil {
			return fmt.Errorf("error adding filter allowing cloud providers metadata service requests: %w", err)
		}
	}

	// add filters for allowed hosts
	if _, err := i.addFiltersForHosts(interfaces, i.spec.AllowedHosts, "1:1"); err != nil {
		return fmt.Errorf("error adding filter for allowed hosts: %w", err)
	}

	// create tc filters depending on the given hosts to match
	// redirect all packets of all interfaces if no host is given
	if len(i.spec.Hosts) == 0 && len(i.spec.Services) == 0 {
		_, nullIP, _ := net.ParseCIDR("0.0.0.0/0")

		for _, protocol := range network.AllProtocols(network.ALL) {
			if _, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, nullIP, 0, 0, protocol, network.ConnStateUndefined, "1:4"); err != nil {
				return fmt.Errorf("can't add a filter: %w", err)
			}
		}
	} else {
		// apply filters for given hosts, re-resolving on a given interval and adding/deleting filters as needed
		if err := i.handleFiltersForHosts(interfaces, "1:4"); err != nil {
			return fmt.Errorf("error adding filters for given hosts: %w", err)
		}

		// add or delete filters for given services depending on changes on the destination kubernetes services and associated pods
		if err := i.handleFiltersForServices(interfaces, "1:4"); err != nil {
			return fmt.Errorf("error adding filters for given services: %w", err)
		}
	}

	return nil
}

// addServiceFilters adds a list of service tc filters on a list of interfaces
func (i *networkDisruptionInjector) addServiceFilters(serviceName string, filters []tcServiceFilter, interfaces []string, flowid string) ([]tcServiceFilter, error) {
	var err error

	builtServices := []tcServiceFilter{}

	for _, filter := range filters {
		i.config.Log.Infow("found service endpoint", "resolvedEndpoint", filter.service.String(), "resolvedService", serviceName)

		for _, protocol := range network.AllProtocols(filter.service.protocol) {
			filter.priority, err = i.config.TrafficController.AddFilter(interfaces, "1:0", "", nil, filter.service.ip, 0, filter.service.port, protocol, network.ConnStateUndefined, flowid)
			if err != nil {
				return nil, err
			}
		}

		i.config.Log.Infow(fmt.Sprintf("added a tc filter for service %s-%s with priority %d", serviceName, filter.service, filter.priority), "interfaces", interfaces)

		builtServices = append(builtServices, filter)
	}

	return builtServices, nil
}

// removeServiceFilter delete tc filters for a k8s service
func (i *networkDisruptionInjector) removeServiceFilter(interfaces []string, tcFilter tcServiceFilter) error {
	if err := i.removeTcFilter(interfaces, tcFilter.priority); err != nil {
		return err
	}

	i.config.Log.Infow("tc filter deleted for all interfaces", "tcServiceFilter", tcFilter, "interfaces", interfaces)

	return nil
}

// delete tc filters using only its priority
func (i *networkDisruptionInjector) removeTcFilter(interfaces []string, priority uint32) error {
	for _, iface := range interfaces {
		if err := i.config.TrafficController.DeleteFilter(iface, priority); err != nil {
			return err
		}
	}

	return nil
}

// removeServiceFiltersInList delete a list of tc filters inside of another list of tc filters
func (i *networkDisruptionInjector) removeServiceFiltersInList(interfaces []string, tcFilters []tcServiceFilter, tcFiltersToRemove []tcServiceFilter) ([]tcServiceFilter, error) {
	for _, serviceToRemove := range tcFiltersToRemove {
		if deletedIdx := i.findServiceFilter(tcFilters, serviceToRemove); deletedIdx >= 0 {
			if err := i.removeServiceFilter(interfaces, tcFilters[deletedIdx]); err != nil {
				return nil, err
			}

			tcFilters = append(tcFilters[:deletedIdx], tcFilters[deletedIdx+1:]...)
		}
	}

	return tcFilters, nil
}

// buildServiceFiltersFromPod builds a list of tc filters per pod endpoint using the service ports
func (i *networkDisruptionInjector) buildServiceFiltersFromPod(pod v1.Pod, servicePorts []v1.ServicePort) []tcServiceFilter {
	// compute endpoint IP (pod IP)
	_, endpointIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", pod.Status.PodIP))

	endpointsToWatch := []tcServiceFilter{}

	for _, port := range servicePorts {
		filter := tcServiceFilter{
			service: networkDisruptionService{
				ip:       endpointIP,
				port:     int(port.TargetPort.IntVal),
				protocol: port.Protocol,
			},
		}

		if i.findServiceFilter(endpointsToWatch, filter) == -1 { // forbid duplication
			endpointsToWatch = append(endpointsToWatch, filter)
		}
	}

	return endpointsToWatch
}

// buildServiceFiltersFromService builds a list of tc filters per service using the service ports
func (i *networkDisruptionInjector) buildServiceFiltersFromService(service v1.Service, servicePorts []v1.ServicePort) []tcServiceFilter {
	// compute service IP (cluster IP)
	_, serviceIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", service.Spec.ClusterIP))

	endpointsToWatch := []tcServiceFilter{}

	if isHeadless(service) {
		return endpointsToWatch
	}

	for _, port := range servicePorts {
		filter := tcServiceFilter{
			service: networkDisruptionService{
				ip:       serviceIP,
				port:     int(port.Port),
				protocol: port.Protocol,
			},
		}

		if i.findServiceFilter(endpointsToWatch, filter) == -1 { // forbid duplication
			endpointsToWatch = append(endpointsToWatch, filter)
		}
	}

	return endpointsToWatch
}

func (i *networkDisruptionInjector) handleWatchError(event watch.Event) error {
	err, ok := event.Object.(*metav1.Status)
	if ok {
		return fmt.Errorf("couldn't watch service in namespace: %s", err.Message)
	}

	return fmt.Errorf("couldn't watch service in namespace")
}

func (i *networkDisruptionInjector) findServiceFilter(tcFilters []tcServiceFilter, toFind tcServiceFilter) int {
	for idx, tcFilter := range tcFilters {
		if tcFilter.service.String() == toFind.service.String() {
			return idx
		}
	}

	return -1
}

// handlePodEndpointsOnServicePortsChange on service changes, delete old filters with the wrong service ports and create new filters
func (i *networkDisruptionInjector) handlePodEndpointsServiceFiltersOnKubernetesServiceChanges(serviceSpec v1beta1.NetworkDisruptionServiceSpec, oldFilters []tcServiceFilter, pods []v1.Pod, servicePorts []v1.ServicePort, interfaces []string, flowid string) ([]tcServiceFilter, error) {
	tcFiltersToCreate, finalTcFilters := []tcServiceFilter{}, []tcServiceFilter{}

	for _, pod := range pods {
		if pod.Status.PodIP != "" { // pods without ip are newly created and will be picked up in the other watcher
			tcFiltersToCreate = append(tcFiltersToCreate, i.buildServiceFiltersFromPod(pod, servicePorts)...) // we build the updated list of tc filters
		}
	}

	// update the list of tc filters by deleting old ones not in the new list of tc filters and creating new tc filters
	for _, oldFilter := range oldFilters {
		if idx := i.findServiceFilter(tcFiltersToCreate, oldFilter); idx >= 0 {
			finalTcFilters = append(finalTcFilters, oldFilter)
			tcFiltersToCreate = append(tcFiltersToCreate[:idx], tcFiltersToCreate[idx+1:]...)
		} else { // delete tc filters which are not in the updated list of tc filters
			if err := i.removeServiceFilter(interfaces, oldFilter); err != nil {
				return nil, err
			}
		}
	}

	createdTcFilters, err := i.addServiceFilters(serviceSpec.Name, tcFiltersToCreate, interfaces, flowid)
	if err != nil {
		return nil, err
	}

	return append(finalTcFilters, createdTcFilters...), nil
}

// handleKubernetesPodsChanges for every changes happening in the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) handleKubernetesServiceChanges(event watch.Event, watcher *serviceWatcher, interfaces []string, flowid string) error {
	var err error

	if event.Type == watch.Error {
		return i.handleWatchError(event)
	}

	service, ok := event.Object.(*v1.Service)
	if !ok {
		return fmt.Errorf("couldn't watch service in namespace, invalid type of watched object received")
	}

	// keep track of resource version to continue watching pods when the watcher has timed out
	// at the right resource already computed.
	if event.Type == watch.Bookmark {
		watcher.servicesResourceVersion = service.ResourceVersion

		return nil
	}

	// We just watch the specified name service
	if watcher.watchedServiceSpec.Name != service.Name {
		return nil
	}

	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	podList, err := i.config.K8sClient.CoreV1().Pods(watcher.watchedServiceSpec.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromValidatedSet(service.Spec.Selector).String(),
	})
	if err != nil {
		return fmt.Errorf("error watching the list of pods for the given kubernetes service (%s/%s): %w", service.Namespace, service.Name, err)
	}

	if isHeadless(*service) {
		// If this is a headless service, we want to block all traffic to the endpoint IPs
		watcher.servicePorts = append(watcher.servicePorts, v1.ServicePort{Port: 0})
	} else {
		watcher.servicePorts, _ = watcher.watchedServiceSpec.ExtractAffectedPortsInServicePorts(service)
	}

	watcher.tcFiltersFromPodEndpoints, err = i.handlePodEndpointsServiceFiltersOnKubernetesServiceChanges(watcher.watchedServiceSpec, watcher.tcFiltersFromPodEndpoints, podList.Items, watcher.servicePorts, interfaces, flowid)
	if err != nil {
		return err
	}

	nsServicesTcFilters := i.buildServiceFiltersFromService(*service, watcher.servicePorts)

	switch event.Type {
	case watch.Added:
		createdTcFilters, err := i.addServiceFilters(watcher.watchedServiceSpec.Name, nsServicesTcFilters, interfaces, flowid)
		if err != nil {
			return err
		}

		watcher.tcFiltersFromNamespaceServices = append(watcher.tcFiltersFromNamespaceServices, createdTcFilters...)
	case watch.Modified:
		if _, err := i.removeServiceFiltersInList(interfaces, watcher.tcFiltersFromNamespaceServices, watcher.tcFiltersFromNamespaceServices); err != nil {
			return err
		}

		watcher.tcFiltersFromNamespaceServices, err = i.addServiceFilters(watcher.watchedServiceSpec.Name, nsServicesTcFilters, interfaces, flowid)
		if err != nil {
			return err
		}
	case watch.Deleted:
		watcher.tcFiltersFromNamespaceServices, err = i.removeServiceFiltersInList(interfaces, watcher.tcFiltersFromNamespaceServices, nsServicesTcFilters)
		if err != nil {
			return err
		}
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// handleKubernetesPodsChanges for every changes happening in the pods related to the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) handleKubernetesPodsChanges(event watch.Event, watcher *serviceWatcher, interfaces []string, flowid string) error {
	var err error

	if event.Type == watch.Error {
		return i.handleWatchError(event)
	}

	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return fmt.Errorf("couldn't watch pods in namespace, invalid type of watched object received")
	}

	// keep track of resource version to continue watching pods when the watcher has timed out
	// at the right resource already computed.
	if event.Type == watch.Bookmark {
		watcher.servicesResourceVersion = pod.ResourceVersion

		return nil
	}

	if err = i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	tcFiltersFromPod := i.buildServiceFiltersFromPod(*pod, watcher.servicePorts)
	if len(tcFiltersFromPod) == 0 {
		return fmt.Errorf("unable to find service %s/%s endpoints to filter", watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace)
	}

	switch event.Type {
	case watch.Added:
		// if the filter already exists, we do nothing
		if i.findServiceFilter(watcher.tcFiltersFromPodEndpoints, tcFiltersFromPod[0]) >= 0 {
			break
		}

		if pod.Status.PodIP != "" {
			createdTcFilters, err := i.addServiceFilters(watcher.watchedServiceSpec.Name, tcFiltersFromPod, interfaces, flowid)
			if err != nil {
				return err
			}

			watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints, createdTcFilters...)
		} else {
			i.config.Log.Infow("newly created destination port has no IP yet, adding to the watch list of pods", "destinationPodName", pod.Name)

			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs, pod.Name)
		}
	case watch.Modified:
		// From the list of pods without IPs that has been added, we create the one that got the IP assigned
		podToCreateIdx := -1

		for idx, podName := range watcher.podsWithoutIPs {
			if podName == pod.Name && pod.Status.PodIP != "" {
				podToCreateIdx = idx

				break
			}
		}

		if podToCreateIdx > -1 {
			tcFilters, err := i.addServiceFilters(watcher.watchedServiceSpec.Name, tcFiltersFromPod, interfaces, flowid)
			if err != nil {
				return err
			}

			watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints, tcFilters...)
			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs[:podToCreateIdx], watcher.podsWithoutIPs[podToCreateIdx+1:]...)
		}
	case watch.Deleted:
		watcher.tcFiltersFromPodEndpoints, err = i.removeServiceFiltersInList(interfaces, watcher.tcFiltersFromPodEndpoints, tcFiltersFromPod)
		if err != nil {
			return err
		}
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// watchServiceChanges for every changes happening in the kubernetes service destination or in the pods related to the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) watchServiceChanges(ctx context.Context, watcher serviceWatcher, interfaces []string, flowid string) {
	for {
		// We create the watcher channels when it's closed
		if watcher.kubernetesServiceWatcher == nil {
			serviceWatcher, err := i.config.K8sClient.CoreV1().Services(watcher.watchedServiceSpec.Namespace).Watch(context.Background(), metav1.ListOptions{
				ResourceVersion:     watcher.servicesResourceVersion,
				AllowWatchBookmarks: true,
			})
			if err != nil {
				i.config.Log.Errorf("error watching the changes for the given kubernetes service (%s/%s): %w", watcher.watchedServiceSpec.Namespace, watcher.watchedServiceSpec.Name, err)

				return
			}

			i.config.Log.Infow("starting kubernetes service watch", "serviceName", watcher.watchedServiceSpec.Name, "serviceNamespace", watcher.watchedServiceSpec.Namespace)
			watcher.kubernetesServiceWatcher = serviceWatcher.ResultChan()
		}

		if watcher.kubernetesPodEndpointsWatcher == nil {
			podsWatcher, err := i.config.K8sClient.CoreV1().Pods(watcher.watchedServiceSpec.Namespace).Watch(context.Background(), metav1.ListOptions{
				LabelSelector:       watcher.labelServiceSelector,
				ResourceVersion:     watcher.podsResourceVersion,
				AllowWatchBookmarks: true,
			})
			if err != nil {
				i.config.Log.Errorf("error watching the list of pods for the given kubernetes service (%s/%s): %w", watcher.watchedServiceSpec.Namespace, watcher.watchedServiceSpec.Name, err)

				return
			}

			i.config.Log.Infow("starting kubernetes pods watch", "serviceName", watcher.watchedServiceSpec.Name, "serviceNamespace", watcher.watchedServiceSpec.Namespace)
			watcher.kubernetesPodEndpointsWatcher = podsWatcher.ResultChan()
		}

		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.kubernetesServiceWatcher: // We have changes in the service watched
			if !ok { // channel is closed
				watcher.kubernetesServiceWatcher = nil
			} else {
				i.config.Log.Debugw(fmt.Sprintf("changes in service %s/%s", watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace), "eventType", event.Type)

				if err := i.handleKubernetesServiceChanges(event, &watcher, interfaces, flowid); err != nil {
					i.config.Log.Errorf("couldn't apply changes to tc filters: %w... Rebuilding watcher", err)

					if _, err = i.removeServiceFiltersInList(interfaces, watcher.tcFiltersFromNamespaceServices, watcher.tcFiltersFromNamespaceServices); err != nil {
						i.config.Log.Errorf("couldn't clean list of tc filters: %w", err)
					}

					watcher.kubernetesServiceWatcher = nil // restart the watcher in case of error
					watcher.tcFiltersFromNamespaceServices = []tcServiceFilter{}
				}
			}
		case event, ok := <-watcher.kubernetesPodEndpointsWatcher: // We have changes in the pods watched
			if !ok { // channel is closed
				watcher.kubernetesPodEndpointsWatcher = nil
			} else {
				i.config.Log.Debugw(fmt.Sprintf("changes in pods of service %s/%s", watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace), "eventType", event.Type)

				if err := i.handleKubernetesPodsChanges(event, &watcher, interfaces, flowid); err != nil {
					i.config.Log.Errorf("couldn't apply changes to tc filters: %w... Rebuilding watcher", err)

					if _, err = i.removeServiceFiltersInList(interfaces, watcher.tcFiltersFromPodEndpoints, watcher.tcFiltersFromPodEndpoints); err != nil {
						i.config.Log.Errorf("couldn't clean list of tc filters: %w", err)
					}

					watcher.kubernetesPodEndpointsWatcher = nil // restart the watcher in case of error
					watcher.tcFiltersFromPodEndpoints = []tcServiceFilter{}
				}
			}
		}
	}
}

// handleFiltersForServices creates tc filters on given interfaces for services in disruption spec classifying matching packets in the given flowid
func (i *networkDisruptionInjector) handleFiltersForServices(interfaces []string, flowid string) error {
	// build the watchers to handle changes in services and pod endpoints
	serviceWatchers := []serviceWatcher{}

	for _, serviceSpec := range i.spec.Services {
		// retrieve serviceSpec
		k8sService, err := i.config.K8sClient.CoreV1().Services(serviceSpec.Namespace).Get(context.Background(), serviceSpec.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting the given kubernetes service (%s/%s): %w", serviceSpec.Namespace, serviceSpec.Name, err)
		}

		servicePorts, _ := serviceSpec.ExtractAffectedPortsInServicePorts(k8sService)

		serviceWatcher := serviceWatcher{
			watchedServiceSpec:   serviceSpec,
			servicePorts:         servicePorts,
			labelServiceSelector: labels.SelectorFromValidatedSet(k8sService.Spec.Selector).String(), // keep this information to later create watchers on resources destination

			kubernetesPodEndpointsWatcher: nil,                 // watch pods related to the kubernetes service filtered on
			tcFiltersFromPodEndpoints:     []tcServiceFilter{}, // list of tc filters targeting pods related to the kubernetes service filtered on
			podsWithoutIPs:                []string{},          // some pods are created without IPs. We keep track of them to later create a tc filter on update
			podsResourceVersion:           "",

			kubernetesServiceWatcher:       nil,                 // watch service filtered on
			tcFiltersFromNamespaceServices: []tcServiceFilter{}, // list of tc filters targeting the service filtered on
			servicesResourceVersion:        "",
		}

		serviceWatchers = append(serviceWatchers, serviceWatcher)
	}

	if i.cancel != nil {
		return fmt.Errorf("some watcher goroutines are already launched, call Clean on injector prior to Inject")
	}

	var ctx context.Context
	ctx, cancelFunc := context.WithCancel(context.Background())
	i.cancel = append(i.cancel, cancelFunc)

	for _, serviceWatcher := range serviceWatchers {
		go i.watchServiceChanges(ctx, serviceWatcher, interfaces, flowid)
	}

	return nil
}

// handleFiltersForServices creates tc filters on given interfaces for hosts in disruption spec classifying matching packets in the given flowid
func (i *networkDisruptionInjector) handleFiltersForHosts(interfaces []string, flowid string) error {
	hosts := hostsWatcher{}

	hostFilterMap, err := i.addFiltersForHosts(interfaces, i.spec.Hosts, flowid)
	if err != nil {
		return err
	}

	hosts.hostFilterMap = hostFilterMap

	var ctx context.Context
	ctx, cancelFunc := context.WithCancel(context.Background())
	i.cancel = append(i.cancel, cancelFunc)

	go i.watchHostChanges(ctx, interfaces, hosts, flowid)

	return nil
}

// watchHostChanges watches for changes to the resolved IP for hosts
func (i *networkDisruptionInjector) watchHostChanges(ctx context.Context, interfaces []string, hosts hostsWatcher, flowid string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(i.config.HostResolveInterval):
			changedHosts := []v1beta1.NetworkDisruptionHostSpec{}
			for host, tcFilters := range hosts.hostFilterMap {
				newIps, err := resolveHost(i.config.DNSClient, host.Host)
				if err != nil {
					i.config.Log.Errorw("error resolving Host", "err", err, "host", host.Host)

					// If we can't get a new set of IPs for this host, just move on to the next one
					continue
				}

				if len(newIps) != len(tcFilters) {
					// TODO debug log?
					// If we have more or fewer IPs than before, we obviously have a change and need to update the tc filters
					changedHosts = append(changedHosts, host)
					continue
				}

				for _, tcF := range tcFilters {
					// TODO debug log?
					if !containsIP(newIps, tcF.ip) {
						// If any of the IPs have changed, lets completely reset the filters for this host
						// TODO for review should we add more logic to track the individual IPs, and only update the ones needed?
						// That's more correct, but has more room for bugs due to complexity, also we'd have to rewrite addFiltersForHosts?
						changedHosts = append(changedHosts, host)
					}
				}
			}

			if len(changedHosts) > 0 {
				for _, changedHost := range changedHosts {
					for _, filter := range hosts.hostFilterMap[changedHost] {
						err := i.removeTcFilter(interfaces, filter.priority)
						if err != nil {
							// TODO test the behavior on missing filters, what happens if we're retrying to remove old filters after failing at i.addFiltersForHosts(interfaces, changedHosts, flowid) and restarting
							i.config.Log.Errorw("error removing out of date tc filter", "err", err, "host", changedHost.Host) // TODO we could leak these filters
						}
					}
				}

				filterMap, err := i.addFiltersForHosts(interfaces, changedHosts, flowid)

				if err != nil {
					i.config.Log.Errorw("error updating filters for hosts", "hosts", changedHosts, "err", err) // TODO we could have partially applied filters
					continue
				}

				for changedHost, filter := range filterMap {
					hosts.hostFilterMap[changedHost] = filter
				}
			}
		}
	}
}

func containsIP(ips []*net.IPNet, lookupIP *net.IPNet) bool {
	for _, ip := range ips {
		if ip == lookupIP {
			return true
		}
	}

	return false
}

// addFiltersForHosts creates tc filters on given interfaces for given hosts classifying matching packets in the given flowid
func (i *networkDisruptionInjector) addFiltersForHosts(interfaces []string, hosts []v1beta1.NetworkDisruptionHostSpec, flowid string) (map[v1beta1.NetworkDisruptionHostSpec][]*tcFilter, error) {
	hostFilterMap := map[v1beta1.NetworkDisruptionHostSpec][]*tcFilter{}

	for _, host := range hosts {
		// resolve given hosts if needed
		ips, err := resolveHost(i.config.DNSClient, host.Host)
		if err != nil {
			return nil, fmt.Errorf("error resolving given host %s: %w", host.Host, err)
		}

		i.config.Log.Infof("resolved %s as %s", host.Host, ips)

		filtersForHost := []*tcFilter{}

		for _, ip := range ips {
			var (
				srcPort, dstPort int
				srcIP, dstIP     *net.IPNet
			)

			// handle flow direction
			switch host.Flow {
			case v1beta1.FlowIngress:
				srcPort = host.Port
				srcIP = ip
			default:
				dstPort = host.Port
				dstIP = ip
			}

			// cast connection state
			connState := network.NewConnState(host.ConnState)
			for _, protocol := range network.AllProtocols(host.Protocol) {
				// create tc filter
				priority, err := i.config.TrafficController.AddFilter(interfaces, "1:0", "", srcIP, dstIP, srcPort, dstPort, protocol, connState, flowid)
				if err != nil {
					return nil, fmt.Errorf("error adding filter for host %s: %w", host.Host, err)
				}

				filtersForHost = append(filtersForHost, &tcFilter{
					ip:       ip,
					priority: priority,
				})
			}

		}
		hostFilterMap[host] = filtersForHost
	}

	return hostFilterMap, nil
}

// AddNetem adds network disruptions using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addNetemOperation(delay, delayJitter time.Duration, drop int, corrupt int, duplicate int) {
	// closure which adds netem disruptions
	operation := func(interfaces []string, parent string, handle string) error {
		return i.config.TrafficController.AddNetem(interfaces, parent, handle, delay, delayJitter, drop, corrupt, duplicate)
	}

	i.operations = append(i.operations, operation)
}

// AddOutputLimit adds a network bandwidth disruption using the drivers in the networkDisruptionInjector
func (i *networkDisruptionInjector) addOutputLimitOperation(bytesPerSec uint) {
	// closure which adds a bandwidth limit
	operation := func(interfaces []string, parent string, handle string) error {
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

	// clear operations to avoid them to stack up
	i.operations = []linkOperation{}

	return nil
}

// isHeadless returns true if the service is a headless service, i.e., has no defined ClusterIP
func isHeadless(service v1.Service) bool {
	return service.Spec.ClusterIP == "" || strings.ToLower(service.Spec.ClusterIP) == "none"
}

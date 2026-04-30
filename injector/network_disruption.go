// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/bpfdisrupt"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/types"
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
	spec                 v1beta1.NetworkDisruptionSpec
	config               NetworkDisruptionInjectorConfig
	operations           []linkOperation
	serviceWatcherCancel context.CancelFunc
	hostWatcherCancel    context.CancelFunc
	engine               *bpfdisrupt.Engine

	// Stored state for BPF rule rebuilds (used by both host and service watchers)
	safeguardIPs   []*net.IPNet
	sshSafeguard   []bpfdisrupt.Rule // SSH safeguard for node-level disruptions
	serviceRulesMu sync.Mutex
	serviceRules   []bpfdisrupt.Rule // current service-derived rules
}

// NetworkDisruptionInjectorConfig contains all needed drivers to create a network disruption using `tc`
type NetworkDisruptionInjectorConfig struct {
	Config
	TrafficController   network.TrafficController
	IPTables            network.IPTables
	NetlinkAdapter      network.NetlinkAdapter
	DNSClient           network.DNSClient
	HostResolveInterval time.Duration
	BPFConfigInformer   ebpf.ConfigInformer
	DNSPodResolvConf    string
	DNSNodeResolvConf   string
	BPFDisruptCmdRunner bpfdisrupt.CmdRunner // optional, for BPF disruption engine
}

// tcServiceFilter describes a service endpoint used for BPF rule generation.
type tcServiceFilter struct {
	service networkDisruptionService
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
		// Create DNS client with custom resolv.conf paths if provided
		dnsConfig := network.DNSClientConfig{
			PodResolvConfPath:  config.DNSPodResolvConf,
			NodeResolvConfPath: config.DNSNodeResolvConf,
			Logger:             config.Log,
		}
		config.DNSClient = network.NewDNSClient(dnsConfig)
	}

	if spec.HasHTTPFilters() && config.BPFConfigInformer == nil {
		config.BPFConfigInformer, err = ebpf.NewConfigInformer(config.Log, config.Disruption.DryRun, nil, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("could not create the eBPF config informer instance for the network disruption: %w", err)
		}
	}

	// Create BPF disruption engine
	var engine *bpfdisrupt.Engine
	if config.BPFDisruptCmdRunner != nil {
		engine = bpfdisrupt.NewEngine(config.TrafficController, config.NetlinkAdapter, config.BPFDisruptCmdRunner, config.Log)
	} else {
		// Use the default generic executor for running the BPF config binary
		cmdRunner := &network.GenericExecutor{Log: config.Log, DryRun: config.Disruption.DryRun}
		engine = bpfdisrupt.NewEngine(config.TrafficController, config.NetlinkAdapter, cmdRunner, config.Log)
	}

	return &networkDisruptionInjector{
		spec:       spec,
		config:     config,
		operations: []linkOperation{},
		engine:     engine,
	}, nil
}

func (i *networkDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindNetworkDisruption
}

func (i *networkDisruptionInjector) TargetName() string {
	return i.config.TargetName()
}

// Inject injects the given network disruption into the given container
func (i *networkDisruptionInjector) Inject() error {
	if i.spec.HasHTTPFilters() {
		if err := i.config.BPFConfigInformer.ValidateRequiredSystemConfig(); err != nil {
			return err
		}

		if !i.config.BPFConfigInformer.GetMapTypes().HaveArrayMapType {
			return fmt.Errorf("the http network failure needs the array map type, but the kernel does not support this type of map")
		}
	}

	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	i.config.Log.Infow("adding network disruptions",
		tags.DropKey, i.spec.Drop,
		tags.DuplicateKey, i.spec.Duplicate,
		tags.CorruptKey, i.spec.Corrupt,
		tags.DelayKey, i.spec.Delay,
		tags.DelayJitterKey, i.spec.DelayJitter,
		tags.BandwidthLimitKey, i.spec.BandwidthLimit,
	)

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
	// With the BPF disruption engine, cgroup marking is only needed when HTTP filters are active
	// (the nested mark filter in the prio sub-tree still requires it).
	// Without HTTP filters, the BPF classifier handles all IP-based classification.
	if i.config.Disruption.Level == types.DisruptionLevelPod && !i.config.Disruption.OnInit && i.spec.HasHTTPFilters() {
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
	if i.serviceWatcherCancel != nil {
		i.serviceWatcherCancel()
		i.serviceWatcherCancel = nil
	}

	if i.hostWatcherCancel != nil {
		i.hostWatcherCancel()
		i.hostWatcherCancel = nil
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
				i.config.Log.Warnw("unable to find target container's net_cls.classid file, we will assume we cannot find the cgroup path because it is gone",
					tags.TargetContainerIDKey, i.config.TargetContainer.ID(),
					tags.ErrorKey, err,
				)

				return nil
			}

			return fmt.Errorf("error cleaning net_cls classid: %w", err)
		}
	}

	return nil
}

// applyOperations applies the added operations by building a tc tree
// Here's what happen on tc side without eBPF filtering:
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
//
// Here's what happen on tc side with eBPF filtering:
//   - a first prio qdisc will be created and attached to root
//     it'll be used to apply the first filter, filtering on packet IP destination, source/destination ports and protocol
//   - a second prio qdisc will be created and attached to the first one
//     it'll be used to apply the second eBPF filter, filtering on method
//   - a third prio qdisc will be created and attached to the second one
//     it'll be used to apply the second eBPF filter, filtering on path
//   - a fourth prio qdisc will be created and attached to the third one
//     it'll be used to apply the third filter, filtering on packet mark to identify packets coming from the targeted process
//   - operations will be chained to the third band of the third prio qdisc
//   - an fw filter will be created to classify packets according to their mark (if any)
//   - a first eBPF filter will be created to classify packets according to their method (if any)
//   - a second eBPF filter will be created to classify packets according to their path (if any)
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
//	  |- (2:) <-- prio qdisc with 2 bands with an eBPF filter to classify packets according to their method
//	    |- (2:1) <-- first band
//	    |- (2:2) <-- second band
//	      |- (3:) <-- <-- prio qdisc with 2 bands with an eBPF filter to classify packets according to their path
//	        |- (3:1) <-- first band
//	        |- (3:2) <-- second band
//		      |- (4:) <-- prio qdisc with 2 bands with an fw filter to classify packets according to their mark (packets with mark 2:2 will be affected by operations)
//			    |- (4:1) <-- first band
//			    |- (4:2) <-- second band
//	          	  |- (5:)  <-- first operation
//	                |- (6:) <-- second operation
//		               ...
func (i *networkDisruptionInjector) applyOperations() error {
	// get interfaces
	useLocalhost := i.config.Disruption.Level == types.DisruptionLevelPod

	links, err := i.config.NetlinkAdapter.LinkList(useLocalhost, i.config.Log)
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

	// When HTTP filters are active, create the nested prio/mark/eBPF chain for L7 classification.
	// The cgroup mark is needed here to identify target pod traffic before HTTP inspection.
	// Without HTTP filters, the BPF disruption engine at parent 1:0 handles all L3/L4 classification
	// directly via LPM trie lookup — no nested prio/mark needed.
	if i.config.Disruption.Level == types.DisruptionLevelPod && !i.config.Disruption.OnInit && i.spec.HasHTTPFilters() {
		// create second prio with only 2 bands to filter traffic based on http method
		if err := i.config.TrafficController.AddPrio(interfaces, "1:4", "2:", 2, [16]uint32{}); err != nil {
			return fmt.Errorf("can't create a new qdisc: %w", err)
		}

		// create a third prio with only 2 bands to filter traffic based on http path
		if err := i.config.TrafficController.AddPrio(interfaces, "2:2", "3:", 2, [16]uint32{}); err != nil {
			return fmt.Errorf("can't create a new qdisc: %w", err)
		}

		// create a fourth prio with only 2 bands to filter traffic with a specific mark
		if err := i.config.TrafficController.AddPrio(interfaces, "3:2", "4:", 2, [16]uint32{}); err != nil {
			return fmt.Errorf("can't create a new qdisc: %w", err)
		}

		// create fw eBPF filters to classify packets based on http method
		if err := i.config.TrafficController.AddBPFFilter(interfaces, "2:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "2:2", "classifier_methods"); err != nil {
			return fmt.Errorf("can't create the eBPF fw filter: %w", err)
		}

		// create fw eBPF filters to classify packets based on http path
		if err := i.config.TrafficController.AddBPFFilter(interfaces, "3:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "3:2", "classifier_paths"); err != nil {
			return fmt.Errorf("can't create the eBPF fw filter: %w", err)
		}

		// run the program responsible to configure the maps of the eBPF tc filters
		bpfConfigExecutor := network.NewBPFTCFilterConfigExecutor(i.config.Log, i.config.Disruption.DryRun)
		configBPFFilterArgs := []string{}

		for _, path := range i.spec.HTTP.Paths {
			configBPFFilterArgs = append(configBPFFilterArgs, "--path", string(path))
		}

		for _, method := range i.spec.HTTP.Methods {
			configBPFFilterArgs = append(configBPFFilterArgs, "--method", strings.ToUpper(method))
		}

		if err = i.config.TrafficController.ConfigBPFFilter(bpfConfigExecutor, configBPFFilterArgs...); err != nil {
			return fmt.Errorf("could not update the configuration of the bpf-network-tc-filter filter: %w", err)
		}

		// create flower filter to classify packets based on their mark
		if err := i.config.TrafficController.AddFlowerFilter(interfaces, "4:0", types.InjectorCgroupClassID, "4:2"); err != nil {
			return fmt.Errorf("can't create the fw filter: %w", err)
		}

		// parent 4:2 refers to the 3nd band of the 4th prio qdisc
		// handle starts from 5 because 1, 2 and 3 are used by the 4 prio qdiscs
		parent = "4:2"
		handle = uint32(5)
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

	// Build safeguard IPs for BPF ALLOW rules (prevent disruption of critical traffic)
	safeguardIPs := []*net.IPNet{}

	switch i.config.Disruption.Level {
	case types.DisruptionLevelPod:
		for _, defaultRoute := range defaultRoutes {
			safeguardIPs = append(safeguardIPs, &net.IPNet{
				IP:   defaultRoute.Gateway(),
				Mask: net.CIDRMask(32, 32),
			})
		}

		safeguardIPs = append(safeguardIPs, nodeIPNet)
	case types.DisruptionLevelNode:
		safeguardIPs = append(safeguardIPs, metadataIPNet)
		// ARP is non-IP traffic that BPF naturally passes through (no safeguard needed).
	}

	// Resolve all hosts and allowed hosts to IPs
	resolvedIPs := map[string][]*net.IPNet{}

	allHosts := make([]v1beta1.NetworkDisruptionHostSpec, 0, len(i.spec.Hosts)+len(i.spec.AllowedHosts))
	allHosts = append(allHosts, i.spec.Hosts...)
	allHosts = append(allHosts, i.spec.AllowedHosts...)

	for _, host := range allHosts {
		if host.Host == "" {
			continue
		}

		ips, err := resolveHost(i.config.DNSClient, host.Host, host.DNSResolver)
		if err != nil {
			i.config.Log.Warnw("error resolving host for BPF rules", tags.HostKey, host.Host, tags.ErrorKey, err)

			continue
		}

		if host.Percentage != nil && *host.Percentage < 100 {
			seed := fmt.Sprintf("%s-%s", host.Host, i.config.Disruption.DisruptionUID)
			ips = network.SelectIPsByPercentage(ips, *host.Percentage, seed)
		}

		resolvedIPs[host.Host] = ips
	}

	// Warn about connState not being supported in BPF mode
	for _, host := range i.spec.Hosts {
		if host.ConnState != "" {
			i.config.Log.Warnw("connState is not supported in BPF disruption mode and will be ignored",
				tags.HostKey, host.Host, "connState", host.ConnState)
		}
	}

	// Store safeguard state for use by watchers during rule rebuilds
	i.safeguardIPs = safeguardIPs

	// Build BPF rules from spec and attach the disruption engine
	rules := specToRules(i.spec, i.spec.Hosts, resolvedIPs, safeguardIPs)

	// Add SSH safeguard for node-level disruptions (port 22/TCP ALLOW on all IPs)
	if i.config.Disruption.Level == types.DisruptionLevelNode {
		sshRule := bpfdisrupt.Rule{
			Direction: bpfdisrupt.DirEgress,
			CIDR:      "0.0.0.0/0",
			Action:    bpfdisrupt.ActionAllow,
			Port:      22,
			Protocol:  "tcp",
		}
		i.sshSafeguard = []bpfdisrupt.Rule{sshRule}
		rules = append(rules, sshRule)
	}

	needsShaping := hasIngressShaping(i.spec)

	if err := i.engine.Attach(interfaces, rules, i.config.Disruption.DisruptionUID, needsShaping); err != nil {
		return fmt.Errorf("error attaching BPF disruption engine: %w", err)
	}

	// If ingress shaping is needed, build netem/tbf chain on IFB device
	if needsShaping {
		ifbIfaces := []string{i.engine.IFBName()}

		if err := i.config.TrafficController.AddPrio(ifbIfaces, "root", "1:", 4, priomap); err != nil {
			return fmt.Errorf("can't create IFB prio qdisc: %w", err)
		}

		ifbParent := "1:4"
		ifbHandle := uint32(2)

		for _, operation := range i.operations {
			if err := operation(ifbIfaces, ifbParent, fmt.Sprintf("%d:", ifbHandle)); err != nil {
				return fmt.Errorf("could not apply operation on IFB device: %w", err)
			}

			ifbParent = fmt.Sprintf("%d:", ifbHandle)
			ifbHandle++
		}
	}

	// Start BPF host watcher for DNS re-resolution (replaces flower filter host watcher)
	if len(i.spec.Hosts) > 0 {
		i.startBPFHostWatcher(safeguardIPs)
	}

	// Start service watchers (services are resolved to BPF rules via engine.UpdateRules)
	if len(i.spec.Services) > 0 {
		if err := i.handleFiltersForServices(); err != nil {
			return fmt.Errorf("error adding filters for given services: %w", err)
		}
	}

	return nil
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

// selectServiceFiltersByPercentage selects a subset of service filters based on percentage using consistent hashing
func (i *networkDisruptionInjector) selectServiceFiltersByPercentage(filters []tcServiceFilter, percentage int, serviceName string, serviceNamespace string) []tcServiceFilter {
	if percentage <= 0 || percentage >= 100 || len(filters) == 0 {
		return filters
	}

	// Extract IPs from filters
	ips := make([]*net.IPNet, 0, len(filters))
	ipToFiltersMap := make(map[string][]tcServiceFilter)

	for _, filter := range filters {
		ipStr := filter.service.ip.String()
		ipToFiltersMap[ipStr] = append(ipToFiltersMap[ipStr], filter)

		// Only add unique IPs
		alreadyAdded := false

		for _, existingIP := range ips {
			if existingIP.String() == ipStr {
				alreadyAdded = true
				break
			}
		}

		if !alreadyAdded {
			ips = append(ips, filter.service.ip)
		}
	}

	// Use consistent hashing to select subset of IPs
	seed := fmt.Sprintf("%s-%s-%s", serviceName, serviceNamespace, i.config.Disruption.DisruptionUID)
	originalCount := len(ips)
	selectedIPs := network.SelectIPsByPercentage(ips, percentage, seed)

	i.config.Log.Infow("selected subset of service endpoints based on percentage",
		"service", fmt.Sprintf("%s/%s", serviceNamespace, serviceName),
		"percentage", percentage,
		"original_count", originalCount,
		"selected_count", len(selectedIPs),
	)

	// Build result filters from selected IPs
	result := []tcServiceFilter{}

	for _, selectedIP := range selectedIPs {
		if filtersForIP, ok := ipToFiltersMap[selectedIP.String()]; ok {
			result = append(result, filtersForIP...)
		}
	}

	return result
}

// handleKubernetesServiceChanges for every changes happening in the kubernetes service destination, we update the BPF disruption rules
func (i *networkDisruptionInjector) handleKubernetesServiceChanges(event watch.Event, watcher *serviceWatcher) error {
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

	// Rebuild pod endpoint list from current pods
	watcher.tcFiltersFromPodEndpoints = []tcServiceFilter{}

	for _, pod := range podList.Items {
		if pod.Status.PodIP != "" {
			watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints, i.buildServiceFiltersFromPod(pod, watcher.servicePorts)...)
		}
	}

	// Apply percentage-based selection if specified
	if watcher.watchedServiceSpec.Percentage != nil && *watcher.watchedServiceSpec.Percentage < 100 && *watcher.watchedServiceSpec.Percentage > 0 {
		watcher.tcFiltersFromPodEndpoints = i.selectServiceFiltersByPercentage(watcher.tcFiltersFromPodEndpoints, *watcher.watchedServiceSpec.Percentage, watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace)
	}

	nsServicesTcFilters := i.buildServiceFiltersFromService(*service, watcher.servicePorts)

	switch event.Type {
	case watch.Added, watch.Modified:
		watcher.tcFiltersFromNamespaceServices = nsServicesTcFilters
	case watch.Deleted:
		watcher.tcFiltersFromNamespaceServices = []tcServiceFilter{}
	}

	// Convert all service endpoints to BPF rules and update the engine
	allServiceFilters := make([]tcServiceFilter, 0, len(watcher.tcFiltersFromPodEndpoints)+len(watcher.tcFiltersFromNamespaceServices))
	allServiceFilters = append(allServiceFilters, watcher.tcFiltersFromPodEndpoints...)
	allServiceFilters = append(allServiceFilters, watcher.tcFiltersFromNamespaceServices...)

	if err := i.updateServiceRules(serviceEndpointsToRules(allServiceFilters)); err != nil {
		i.config.Log.Warnw("error updating BPF rules for service changes", tags.ErrorKey, err)
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// handleKubernetesPodsChanges for every changes happening in the pods related to the kubernetes service destination, we update the BPF disruption rules
func (i *networkDisruptionInjector) handleKubernetesPodsChanges(event watch.Event, watcher *serviceWatcher) error {
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
		watcher.podsResourceVersion = pod.ResourceVersion

		return nil
	}

	if err = i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	tcFiltersFromPod := i.buildServiceFiltersFromPod(*pod, watcher.servicePorts)
	if len(tcFiltersFromPod) == 0 {
		return fmt.Errorf("unable to find service %s/%s endpoints to filter", watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace)
	}

	needsUpdate := false

	switch event.Type {
	case watch.Added:
		if i.findServiceFilter(watcher.tcFiltersFromPodEndpoints, tcFiltersFromPod[0]) >= 0 {
			break // filter already exists
		}

		if pod.Status.PodIP != "" {
			watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints, tcFiltersFromPod...)
			needsUpdate = true
		} else {
			i.config.Log.Infow("newly created destination port has no IP yet, adding to the watch list of pods", tags.DestinationPodNameKey, pod.Name)
			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs, pod.Name)
		}
	case watch.Modified:
		podToCreateIdx := -1

		for idx, podName := range watcher.podsWithoutIPs {
			if podName == pod.Name && pod.Status.PodIP != "" {
				podToCreateIdx = idx

				break
			}
		}

		if podToCreateIdx > -1 {
			watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints, tcFiltersFromPod...)
			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs[:podToCreateIdx], watcher.podsWithoutIPs[podToCreateIdx+1:]...)
			needsUpdate = true
		}
	case watch.Deleted:
		for _, toRemove := range tcFiltersFromPod {
			if idx := i.findServiceFilter(watcher.tcFiltersFromPodEndpoints, toRemove); idx >= 0 {
				watcher.tcFiltersFromPodEndpoints = append(watcher.tcFiltersFromPodEndpoints[:idx], watcher.tcFiltersFromPodEndpoints[idx+1:]...)
				needsUpdate = true
			}
		}
	}

	if needsUpdate {
		allServiceFilters := make([]tcServiceFilter, 0, len(watcher.tcFiltersFromPodEndpoints)+len(watcher.tcFiltersFromNamespaceServices))
		allServiceFilters = append(allServiceFilters, watcher.tcFiltersFromPodEndpoints...)
		allServiceFilters = append(allServiceFilters, watcher.tcFiltersFromNamespaceServices...)

		if err := i.updateServiceRules(serviceEndpointsToRules(allServiceFilters)); err != nil {
			i.config.Log.Warnw("error updating BPF rules for pod changes", tags.ErrorKey, err)
		}
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// watchServiceChanges for every changes happening in the kubernetes service destination or in the pods related to the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) watchServiceChanges(ctx context.Context, watcher serviceWatcher) {
	log := i.config.Log.With(tags.ServiceNamespaceKey, watcher.watchedServiceSpec.Namespace, tags.ServiceNameKey, watcher.watchedServiceSpec.Name)

	for {
		// We create the watcher channels when it's closed
		if watcher.kubernetesServiceWatcher == nil {
			watchLog := log.With(tags.WatcherNameKey, "kubernetesServiceWatcher")

			k8sServiceWatcher, err := i.config.K8sClient.CoreV1().Services(watcher.watchedServiceSpec.Namespace).Watch(context.Background(), metav1.ListOptions{
				ResourceVersion:     watcher.servicesResourceVersion,
				AllowWatchBookmarks: true,
			})
			if err != nil {
				watchLog.Errorw("error watching the changes for the given kubernetes service", tags.ErrorKey, err)

				return
			}

			watchLog.Infow("starting kubernetes service watch")

			watcher.kubernetesServiceWatcher = k8sServiceWatcher.ResultChan()
		}

		if watcher.kubernetesPodEndpointsWatcher == nil {
			watcherLog := log.With(tags.WatcherNameKey, "kubernetesPodEndpointsWatcher")

			podsWatcher, err := i.config.K8sClient.CoreV1().Pods(watcher.watchedServiceSpec.Namespace).Watch(context.Background(), metav1.ListOptions{
				LabelSelector:       watcher.labelServiceSelector,
				ResourceVersion:     watcher.podsResourceVersion,
				AllowWatchBookmarks: true,
			})
			if err != nil {
				watcherLog.Errorw("error watching the list of pods for the given kubernetes service", tags.ErrorKey, err)

				return
			}

			watcherLog.Infow("starting kubernetes pods watch")

			watcher.kubernetesPodEndpointsWatcher = podsWatcher.ResultChan()
		}

		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.kubernetesServiceWatcher: // We have changes in the service watched
			if !ok { // channel is closed
				watcher.kubernetesServiceWatcher = nil
			} else {
				watcherLog := log.With(tags.WatcherNameKey, "kubernetesServiceWatcher")
				watcherLog.Debugw("changes in service", tags.EventTypeKey, event.Type)

				if err := i.handleKubernetesServiceChanges(event, &watcher); err != nil {
					watcherLog.Errorw("couldn't apply service changes: Rebuilding watcher", tags.ErrorKey, err)

					watcher.kubernetesServiceWatcher = nil // restart the watcher in case of error
					watcher.tcFiltersFromNamespaceServices = []tcServiceFilter{}
				}
			}
		case event, ok := <-watcher.kubernetesPodEndpointsWatcher: // We have changes in the pods watched
			if !ok { // channel is closed
				watcher.kubernetesPodEndpointsWatcher = nil
			} else {
				watcherLog := log.With(tags.WatcherNameKey, "kubernetesPodEndpointsWatcher")
				watcherLog.Debugw(fmt.Sprintf("changes in pods of service %s/%s", watcher.watchedServiceSpec.Name, watcher.watchedServiceSpec.Namespace), tags.EventTypeKey, event.Type)

				if err := i.handleKubernetesPodsChanges(event, &watcher); err != nil {
					watcherLog.Errorw("couldn't apply pod changes: Rebuilding watcher", tags.ErrorKey, err)

					watcher.kubernetesPodEndpointsWatcher = nil // restart the watcher in case of error
					watcher.tcFiltersFromPodEndpoints = []tcServiceFilter{}
				}
			}
		}
	}
}

// handleFiltersForServices sets up Kubernetes service watchers that resolve service endpoints
// to BPF disruption rules and update the engine when endpoints change.
func (i *networkDisruptionInjector) handleFiltersForServices() error {
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

	if i.serviceWatcherCancel != nil {
		return fmt.Errorf("some service watcher goroutines are already launched, call Clean on injector prior to Inject")
	}

	var ctx context.Context

	ctx, cancelFunc := context.WithCancel(context.Background())
	i.serviceWatcherCancel = cancelFunc

	for _, serviceWatcher := range serviceWatchers {
		go i.watchServiceChanges(ctx, serviceWatcher)
	}

	return nil
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
	useLocalhost := i.config.Disruption.Level == types.DisruptionLevelPod

	links, err := i.config.NetlinkAdapter.LinkList(useLocalhost, i.config.Log)
	if err != nil {
		return fmt.Errorf("can't get interfaces per IP map: %w", err)
	}

	// clear all interfaces root qdisc so it gets back to default
	interfaces := []string{}
	for _, link := range links {
		interfaces = append(interfaces, link.Name())
	}

	// Detach BPF disruption engine (removes clsact qdisc and IFB device)
	if i.engine != nil {
		if err := i.engine.Detach(); err != nil {
			i.config.Log.Warnw("error detaching BPF disruption engine", tags.ErrorKey, err)
		}
	}

	// clear link qdisc if needed
	if err := i.config.TrafficController.ClearQdisc(interfaces); err != nil {
		return fmt.Errorf("error deleting root qdisc: %w", err)
	}

	// clear operations to avoid them to stack up
	i.operations = []linkOperation{}

	return nil
}

// specToRules converts a NetworkDisruptionSpec and resolved hosts into BPF disruption rules.
// safeguardIPs are added as ALLOW rules to prevent disruption of critical traffic.
func specToRules(spec v1beta1.NetworkDisruptionSpec, hosts []v1beta1.NetworkDisruptionHostSpec, resolvedIPs map[string][]*net.IPNet, safeguardIPs []*net.IPNet) []bpfdisrupt.Rule {
	rules := []bpfdisrupt.Rule{}

	// Add safeguard IPs as ALLOW rules (LPM /32 entries beat /0 match-all)
	for _, ip := range safeguardIPs {
		rules = append(rules, bpfdisrupt.Rule{
			Direction: bpfdisrupt.DirEgress,
			CIDR:      ip.String(),
			Action:    bpfdisrupt.ActionAllow,
		})
	}

	// Add allowed hosts as ALLOW rules
	for _, host := range spec.AllowedHosts {
		ips, ok := resolvedIPs[host.Host]
		if !ok {
			continue
		}

		for _, ip := range ips {
			rules = append(rules, bpfdisrupt.Rule{
				Direction: bpfdisrupt.DirEgress, // allowed hosts affect both directions via LPM precedence
				CIDR:      ip.String(),
				Action:    bpfdisrupt.ActionAllow,
			})
		}
	}

	// If no hosts specified, add match-all rule
	if len(hosts) == 0 {
		rules = append(rules, bpfdisrupt.Rule{
			Direction: bpfdisrupt.DirEgress,
			CIDR:      "0.0.0.0/0",
			Action:    bpfdisrupt.ActionDisrupt,
		})

		return rules
	}

	// Add rules for each host
	for _, host := range hosts {
		ips, ok := resolvedIPs[host.Host]
		if !ok {
			continue
		}

		// Determine direction
		dir := bpfdisrupt.DirEgress
		if host.Flow == v1beta1.FlowIngress {
			dir = bpfdisrupt.DirIngress
		}

		// Determine action
		action := bpfdisrupt.ActionDisrupt
		dropPct := 0

		// For ingress-only drop (no delay/bandwidth/corrupt/duplicate), use ActionDrop
		if dir == bpfdisrupt.DirIngress && spec.Drop > 0 &&
			spec.Delay == 0 && spec.BandwidthLimit == 0 && spec.Corrupt == 0 && spec.Duplicate == 0 {
			action = bpfdisrupt.ActionDrop
			dropPct = spec.Drop
		}

		for _, ip := range ips {
			rules = append(rules, bpfdisrupt.Rule{
				Direction: dir,
				CIDR:      ip.String(),
				Action:    action,
				DropPct:   dropPct,
				Port:      host.Port,
				Protocol:  host.Protocol,
			})
		}
	}

	return rules
}

// hasIngressShaping returns true if any host has FlowIngress with disruptions
// that require shaping (delay, jitter, bandwidth, corruption, duplication) rather than just drop.
func hasIngressShaping(spec v1beta1.NetworkDisruptionSpec) bool {
	hasIngress := false

	for _, host := range spec.Hosts {
		if host.Flow == v1beta1.FlowIngress {
			hasIngress = true

			break
		}
	}

	if !hasIngress {
		return false
	}

	// If any shaping parameter is set alongside ingress hosts, IFB is needed
	return spec.Delay > 0 || spec.BandwidthLimit > 0 || spec.Corrupt > 0 || spec.Duplicate > 0
}

// startBPFHostWatcher launches a background goroutine that periodically re-resolves
// host DNS entries and updates the BPF LPM trie rules via engine.UpdateRules().
func (i *networkDisruptionInjector) startBPFHostWatcher(safeguardIPs []*net.IPNet) {
	if i.hostWatcherCancel != nil {
		return // already running
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	i.hostWatcherCancel = cancelFunc

	go i.watchBPFHostChanges(ctx, safeguardIPs)
}

// watchBPFHostChanges periodically re-resolves all hosts and updates BPF disruption rules.
func (i *networkDisruptionInjector) watchBPFHostChanges(ctx context.Context, safeguardIPs []*net.IPNet) {
	watcherLog := i.config.Log.With(tags.RetryIntervalKey, i.config.HostResolveInterval.String())

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(i.config.HostResolveInterval):
			if err := i.config.Netns.Enter(); err != nil {
				watcherLog.Errorw("unable to enter the given container network namespace, retrying on next watch occurrence", tags.ErrorKey, err)

				continue
			}

			// Re-resolve all hosts
			resolvedIPs := map[string][]*net.IPNet{}

			allHosts := make([]v1beta1.NetworkDisruptionHostSpec, 0, len(i.spec.Hosts)+len(i.spec.AllowedHosts))
			allHosts = append(allHosts, i.spec.Hosts...)
			allHosts = append(allHosts, i.spec.AllowedHosts...)

			for _, host := range allHosts {
				if host.Host == "" {
					continue
				}

				ips, err := resolveHost(i.config.DNSClient, host.Host, host.DNSResolver)
				if err != nil {
					watcherLog.Errorw("error resolving host", tags.ErrorKey, err, tags.HostKey, host.Host)

					continue
				}

				if host.Percentage != nil && *host.Percentage < 100 {
					seed := fmt.Sprintf("%s-%s", host.Host, i.config.Disruption.DisruptionUID)
					ips = network.SelectIPsByPercentage(ips, *host.Percentage, seed)
				}

				resolvedIPs[host.Host] = ips
			}

			// Rebuild host rules and merge with service rules atomically
			hostRules := specToRules(i.spec, i.spec.Hosts, resolvedIPs, safeguardIPs)
			if err := i.rebuildAndUpdateAllRules(hostRules); err != nil {
				watcherLog.Errorw("error updating BPF disruption rules", tags.ErrorKey, err)
			}

			if err := i.config.Netns.Exit(); err != nil {
				watcherLog.Errorw("unable to exit the given container network namespace", tags.ErrorKey, err)
			}
		}
	}
}

// rebuildAndUpdateAllRules rebuilds the complete BPF rule set from all sources
// (hosts, services, safeguards) and atomically updates the engine.
// Called by both host and service watchers when their resolved IPs change.
func (i *networkDisruptionInjector) rebuildAndUpdateAllRules(hostRules []bpfdisrupt.Rule) error {
	i.serviceRulesMu.Lock()
	currentServiceRules := make([]bpfdisrupt.Rule, len(i.serviceRules))
	copy(currentServiceRules, i.serviceRules)
	i.serviceRulesMu.Unlock()

	allRules := make([]bpfdisrupt.Rule, 0, len(hostRules)+len(currentServiceRules)+len(i.sshSafeguard))
	allRules = append(allRules, hostRules...)
	allRules = append(allRules, currentServiceRules...)
	allRules = append(allRules, i.sshSafeguard...)

	return i.engine.UpdateRules(allRules)
}

// updateServiceRules updates the service portion of the BPF rules and triggers a full rebuild.
func (i *networkDisruptionInjector) updateServiceRules(serviceRules []bpfdisrupt.Rule) error {
	i.serviceRulesMu.Lock()
	i.serviceRules = serviceRules
	i.serviceRulesMu.Unlock()

	// Rebuild host rules from current state
	resolvedIPs := map[string][]*net.IPNet{}

	allHosts := make([]v1beta1.NetworkDisruptionHostSpec, 0, len(i.spec.Hosts)+len(i.spec.AllowedHosts))
	allHosts = append(allHosts, i.spec.Hosts...)
	allHosts = append(allHosts, i.spec.AllowedHosts...)

	for _, host := range allHosts {
		if host.Host == "" {
			continue
		}

		ips, err := resolveHost(i.config.DNSClient, host.Host, host.DNSResolver)
		if err != nil {
			continue
		}

		if host.Percentage != nil && *host.Percentage < 100 {
			seed := fmt.Sprintf("%s-%s", host.Host, i.config.Disruption.DisruptionUID)
			ips = network.SelectIPsByPercentage(ips, *host.Percentage, seed)
		}

		resolvedIPs[host.Host] = ips
	}

	hostRules := specToRules(i.spec, i.spec.Hosts, resolvedIPs, i.safeguardIPs)

	return i.rebuildAndUpdateAllRules(hostRules)
}

// serviceEndpointsToRules converts a list of service tc filters (endpoint IP/port/protocol tuples)
// into BPF disruption rules. This is the service analog of specToRules for hosts.
func serviceEndpointsToRules(filters []tcServiceFilter) []bpfdisrupt.Rule {
	rules := []bpfdisrupt.Rule{}

	for _, filter := range filters {
		if filter.service.ip == nil {
			continue
		}

		protocol := ""

		switch filter.service.protocol {
		case v1.ProtocolTCP:
			protocol = "tcp"
		case v1.ProtocolUDP:
			protocol = "udp"
		}

		rules = append(rules, bpfdisrupt.Rule{
			Direction: bpfdisrupt.DirEgress,
			CIDR:      filter.service.ip.String(),
			Action:    bpfdisrupt.ActionDisrupt,
			Port:      filter.service.port,
			Protocol:  protocol,
		})
	}

	return rules
}

// isHeadless returns true if the service is a headless service, i.e., has no defined ClusterIP
func isHeadless(service v1.Service) bool {
	return service.Spec.ClusterIP == "" || strings.ToLower(service.Spec.ClusterIP) == "none"
}

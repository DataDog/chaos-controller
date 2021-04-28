// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DNSDisruptionInjector describes a dns disruption
type DNSDisruptionInjector struct {
	spec   v1beta1.DNSDisruptionSpec
	config DNSDisruptionInjectorConfig
}

// DNSDisruptionInjectorConfig contains all needed drivers to create a dns disruption using `iptables`
type DNSDisruptionInjectorConfig struct {
	Config
	Iptables     network.Iptables
	FileWriter   FileWriter
	PythonRunner PythonRunner
}

const INJECTOR_DNS_CGROUP_CLASSID = "0x00100010"

// NewDNSDisruptionInjector creates a DNSDisruptionInjector object with the given config,
// missing fields are initialized with the defaults
func NewDNSDisruptionInjector(spec v1beta1.DNSDisruptionSpec, config DNSDisruptionInjectorConfig) (Injector, error) {
	var err error
	if config.Iptables == nil {
		config.Iptables, err = network.NewIptables(config.Log, config.DryRun)
	}

	if config.FileWriter == nil {
		config.FileWriter = standardFileWriter{
			dryRun: config.DryRun,
		}
	}

	if config.PythonRunner == nil {
		config.PythonRunner = standardPythonRunner{
			dryRun: config.DryRun,
			log:    config.Log,
		}
	}

	return DNSDisruptionInjector{
		spec:   spec,
		config: config,
	}, err
}

// Inject injects the given dns disruption into the given container
func (i DNSDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding dns disruption", "spec", i.spec)

	// get the chaos pod node IP from the environment variable
	podIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	// Set up resolver config file
	resolverConfig := []string{}
	for _, record := range i.spec {
		resolverConfig = append(resolverConfig, fmt.Sprintf("%s %s %s", record.Record.Type, record.Hostname, record.Record.Value))
	}

	if err := i.config.FileWriter.Write("/tmp/dns.conf", 0644, strings.Join(resolverConfig, "\n")); err != nil {
		return fmt.Errorf("unable to write resolver config: %w", err)
	}

	_, _, err := i.config.PythonRunner.RunPython("/usr/local/bin/dns_disruption_resolver.py", "-c", "/tmp/dns.conf")
	if err != nil {
		return fmt.Errorf("unable to run resolver: %w", err)
	}

	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// Set up iptables rules
	if err := i.config.Iptables.CreateChain("CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to create new iptables chain: %w", err)
	}

	if err := i.config.Iptables.AddRuleWithIP("CHAOS-DNS", "udp", "53", "DNAT", podIP); err != nil {
		return fmt.Errorf("unable to create new iptables rule: %w", err)
	}

	// write classid to container net_cls cgroup - for iptable filtering
	if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", INJECTOR_DNS_CGROUP_CLASSID); err != nil {
		return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
	}

	// Redirect traffic marked by targeted INJECTOR_DNS_CGROUP_CLASSID to CHAOS-DNS
	if err := i.config.Iptables.AddCgroupFilterRule("OUTPUT", INJECTOR_DNS_CGROUP_CLASSID, "udp", "53", "CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to create new iptables rule: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelNode {
		// Exempt chaos pod from iptables re-routing
		if err := i.config.Iptables.PrependRule("CHAOS-DNS", "-s", podIP, "-j", "RETURN"); err != nil {
			return fmt.Errorf("unable to create new iptables rule: %w", err)
		}

		// Re-route all pods under node
		if err := i.config.Iptables.PrependRule("OUTPUT", "-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"); err != nil {
			return fmt.Errorf("unable to create new iptables rule: %w", err)
		}

		if err := i.config.Iptables.PrependRule("PREROUTING", "-p", "udp", "--dport", "53", "-j", "CHAOS-DNS"); err != nil {
			return fmt.Errorf("unable to create new iptables rule: %w", err)
		}
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// Clean removes the injected disruption from the given container
func (i DNSDisruptionInjector) Clean() error {
	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// reset net_cls classid
	if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0x0"); err != nil {
		return fmt.Errorf("error writing classid to pod net_cls cgroup: %w", err)
	}

	// Delete iptables rules
	if err := i.config.Iptables.DeleteCgroupFilterRule("OUTPUT", INJECTOR_DNS_CGROUP_CLASSID, "udp", "53", "CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to remove injected iptables rule: %w", err)
	}

	if i.config.Level == chaostypes.DisruptionLevelNode {
		// Delete prerouting rule affecting all pods on node
		if err := i.config.Iptables.DeleteRule("OUTPUT", "udp", "53", "CHAOS-DNS"); err != nil {
			return fmt.Errorf("unable to remove new iptables rule: %w", err)
		}

		if err := i.config.Iptables.DeleteRule("PREROUTING", "udp", "53", "CHAOS-DNS"); err != nil {
			return fmt.Errorf("unable to remove new iptables rule: %w", err)
		}
	}

	if err := i.config.Iptables.ClearAndDeleteChain("CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to remove injected iptables chain: %w", err)
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	// There is nothing we need to do to shut down the resolver beyond letting the pod terminate
	return nil
}

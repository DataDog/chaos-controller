// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/process"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var (
	launchDNSServer    sync.Once
	launchDNSServerErr error
)

// DNSDisruptionInjector describes a dns disruption
type DNSDisruptionInjector struct {
	spec   v1beta1.DNSDisruptionSpec
	config DNSDisruptionInjectorConfig
}

// DNSDisruptionInjectorConfig contains all needed drivers to create a dns disruption using `iptables`
type DNSDisruptionInjectorConfig struct {
	Config
	DisruptionName      string
	DisruptionNamespace string
	TargetName          string
	FileWriter          FileWriter
	IPTables            network.IPTables
	CmdFactory          command.Factory
	ProcessManager      process.Manager
}

// NewDNSDisruptionInjector creates a DNSDisruptionInjector object with the given config,
// missing fields are initialized with the defaults
func NewDNSDisruptionInjector(spec v1beta1.DNSDisruptionSpec, config DNSDisruptionInjectorConfig) (Injector, error) {
	var err error
	if config.IPTables == nil {
		config.IPTables, err = network.NewIPTables(config.Log, config.Disruption.DryRun)
	}

	if config.FileWriter == nil {
		config.FileWriter = standardFileWriter{
			dryRun: config.Disruption.DryRun,
		}
	}

	if config.CmdFactory == nil {
		config.CmdFactory = command.NewFactory(config.Disruption.DryRun)
	}

	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager(config.Disruption.DryRun)
	}

	return &DNSDisruptionInjector{
		spec:   spec,
		config: config,
	}, err
}

func (i *DNSDisruptionInjector) TargetName() string {
	return i.config.Config.TargetName()
}

func (i *DNSDisruptionInjector) GetDisruptionKind() chaostypes.DisruptionKindName {
	return chaostypes.DisruptionKindDNSDisruption
}

// Inject injects the given dns disruption into the given container
func (i *DNSDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding dns disruption", "spec", i.spec)

	// get the chaos pod node IP from the environment variable
	podIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	// Create a Fake DNS server, once
	launchDNSServer.Do(func() {
		// Set up resolver config file
		resolverConfig := []string{}
		for _, record := range i.spec {
			resolverConfig = append(resolverConfig, fmt.Sprintf("%s %s %s", record.Record.Type, record.Hostname, record.Record.Value))
		}

		if err := i.config.FileWriter.Write("/tmp/dns.conf", 0o644, strings.Join(resolverConfig, "\n")); err != nil {
			launchDNSServerErr = fmt.Errorf("unable to write resolver config: %w", err)
			return
		}

		args := []string{"/usr/local/bin/dns_disruption_resolver.py", "-c", "/tmp/dns.conf"}

		if i.config.Disruption.DisruptionName != "" {
			args = append(args, "--log-context-disruption-name", i.config.Disruption.DisruptionName)
		}

		if i.config.Disruption.DisruptionNamespace != "" {
			args = append(args, "--log-context-disruption-namespace", i.config.Disruption.DisruptionNamespace)
		}

		if i.config.Disruption.TargetName != "" {
			args = append(args, "--log-context-target-name", i.config.Disruption.TargetName)
		}

		if i.config.Disruption.TargetNodeName != "" {
			args = append(args, "--log-context-target-node-name", i.config.Disruption.TargetNodeName)
		}

		if i.config.DNS.DNSServer != "" {
			args = append(args, "--dns", i.config.DNS.DNSServer)
		}

		if i.config.DNS.KubeDNS != "" {
			args = append(args, "--kube-dns", i.config.DNS.KubeDNS)
		}

		cmd := i.config.CmdFactory.NewCmd(context.Background(), "/usr/bin/python3", args)

		bgCmd := command.NewBackgroundCmd(cmd, i.config.Log, i.config.ProcessManager)
		if err := bgCmd.Start(); err != nil {
			launchDNSServerErr = fmt.Errorf("unable to run resolver: %w", err)
			return
		}
	})

	// This will return an error on all injectors if the launchDNSServer once call failed
	// if we did not succeed to create a fake DNS server, we should not continue
	if launchDNSServerErr != nil {
		return launchDNSServerErr
	}

	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// Set up iptables rules to redirect dns requests to the injector pod
	// which holds the dns proxy process
	if err := i.config.IPTables.RedirectTo("udp", "53", podIP); err != nil {
		return fmt.Errorf("unable to create new iptables rule: %w", err)
	}

	if i.config.Disruption.Level == chaostypes.DisruptionLevelPod {
		if i.config.Disruption.OnInit {
			// Redirect all dns related traffic in the pod to CHAOS-DNS
			if err := i.config.IPTables.Intercept("udp", "53", "", "", ""); err != nil {
				return fmt.Errorf("unable to create new iptables rule: %w", err)
			}
		} else {
			cgroupPath := ""
			classID := ""

			if i.config.Cgroup.IsCgroupV2() { // Filter packets on cgroup path for cgroup v2
				cgroupPath = i.config.Cgroup.RelativePath("")
			} else { // Filter packets on net_cls classid for cgroup v1
				classID = chaostypes.InjectorCgroupClassID

				// Apply the classid through net_cls to all packets created by the container
				if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", classID); err != nil {
					return fmt.Errorf("unable to write net_cls classid: %w", err)
				}
			}

			// Redirect packets based on their cgroup or classid depending on cgroup version to CHAOS-DNS
			if err := i.config.IPTables.Intercept("udp", "53", cgroupPath, classID, podIP); err != nil {
				return fmt.Errorf("unable to create new iptables rule: %w", err)
			}
		}
	}

	if i.config.Disruption.Level == chaostypes.DisruptionLevelNode {
		// Re-route all pods under node except for injector pod itself
		if err := i.config.IPTables.Intercept("udp", "53", "", "", podIP); err != nil {
			return fmt.Errorf("unable to create new iptables rule: %w", err)
		}
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

func (i *DNSDisruptionInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes the injected disruption from the given container
func (i *DNSDisruptionInjector) Clean() error {
	// enter target network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// clean injected iptables
	if err := i.config.IPTables.Clear(); err != nil {
		return fmt.Errorf("unable to clean iptables rules and chain: %w", err)
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	// Remove the net_cls classid for cgroup v1
	if !i.config.Cgroup.IsCgroupV2() {
		if err := i.config.Cgroup.Write("net_cls", "net_cls.classid", "0"); err != nil {
			if os.IsNotExist(err) {
				i.config.Log.Warnw("unable to find target container's net_cls.classid file, we will assume we cannot find the cgroup path because it is gone", "targetContainerID", i.config.TargetContainer.ID(), "error", err)
				return nil
			}

			return fmt.Errorf("error cleaning net_cls classid: %w", err)
		}
	}

	// There is nothing we need to do to shut down the resolver beyond letting the pod terminate
	return nil
}

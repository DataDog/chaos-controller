// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DNSDisruptionInjector describes a dns disruption
type DNSDisruptionInjector struct {
	spec   v1beta1.DNSDisruptionSpec
	config DNSDisruptionInjectorConfig
}

// DNSDisruptionInjectorConfig contains all needed drivers to create a dns disruption using `tc`
type DNSDisruptionInjectorConfig struct {
	Config
	Iptables   network.Iptables
	FileWriter FileWriter
}

// NewDNSDisruptionInjector creates a DNSDisruptionInjector object with the given config,
// missing field being initialized with the defaults
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

	return DNSDisruptionInjector{
		spec:   spec,
		config: config,
	}, err
}

// Inject injects the given dns disruption into the given container
func (i DNSDisruptionInjector) Inject() error {
	i.config.Log.Infow("adding dns disruption", "spec", i.spec)

	// get the targeted pod node IP from the environment variable
	podIP, ok := os.LookupEnv(chaostypes.ChaosPodIPEnv)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", chaostypes.ChaosPodIPEnv)
	}

	// Set up resolver config file
	resolverConfig := []string{}
	for _, record := range i.spec {
		resolverConfig = append(resolverConfig, fmt.Sprintf("%s %s %s", record.Record.Type, record.Host, record.Record.Value))
	}

	if err := i.config.FileWriter.Write("/tmp/dns.conf", 0644, strings.Join(resolverConfig, "\n")); err != nil {
		return fmt.Errorf("unable to write resolver config: %w", err)
	}

	// Run resolver (python is at /usr/bin/python3) (resolver is at /usr/local/bin)
	_, _, err := i.RunPython("/usr/local/bin/dns_disruption_resolver.py", "-c", "/tmp/dns.conf")
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

	if err := i.config.Iptables.AddRule("OUTPUT", "udp", "53", "CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to create new iptables rule: %w", err)
	}

	// exit target network namespace
	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// Clean removes the injected disruption in the given container
func (i DNSDisruptionInjector) Clean() error {
	// Delete iptables rules
	if err := i.config.Iptables.DeleteRule("OUTPUT", "udp", "53", "CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to remove injected iptables rule: %w", err)
	}

	if err := i.config.Iptables.DeleteRuleByNum("CHAOS-DNS", 1); err != nil {
		return fmt.Errorf("unable to remove injected iptables rule: %w", err)
	}

	if err := i.config.Iptables.DeleteChain("CHAOS-DNS"); err != nil {
		return fmt.Errorf("unable to remove injected iptables chain: %w", err)
	}

	// Resolver goes away when pod dies?
	return nil
}

// RunPython executes the given args using the python3 command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (i DNSDisruptionInjector) RunPython(args ...string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command("/usr/bin/python3", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	i.config.Log.Infof("running python3 command: %v", cmd.String())

	// early exit if dry-run mode is enabled
	if i.config.DryRun {
		return 0, "", nil
	}

	err := cmd.Start()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	return cmd.ProcessState.ExitCode(), stdout.String(), err
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
	"github.com/coreos/go-iptables/iptables"
	"go.uber.org/zap"
)

const iptablesTable = "filter"
const iptablesOutputChain = "OUTPUT"
const iptablesChaosChainPrefix = "CHAOS-"

// IPTables is an interface abstracting an iptables driver
type IPTables interface {
	Exists(table, chain string, parts ...string) (bool, error)
	Delete(table, chain string, parts ...string) error
	ClearChain(table, chain string) error
	DeleteChain(table, chain string) error
	ListChains(table string) ([]string, error)
	AppendUnique(table, chain string, parts ...string) error
	NewChain(table, chain string) error
}

// NetworkFailureInjectorConfig contains needed drivers
// for instanciating a NetworkFailureInjector object
type NetworkFailureInjectorConfig struct {
	IPTables  IPTables
	DNSClient network.DNSClient
}

// networkFailureInjector describes a network failure
type networkFailureInjector struct {
	containerInjector
	config *NetworkFailureInjectorConfig
	spec   v1beta1.NetworkFailureSpec
}

// NewNetworkFailureInjector creates a NetworkFailureInjector object with default drivers
func NewNetworkFailureInjector(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) (Injector, error) {
	return NewNetworkFailureInjectorWithConfig(uid, spec, ctn, log, ms, &NetworkFailureInjectorConfig{})
}

// NewNetworkFailureInjectorWithConfig creates a NetworkFailureInjector object with the given config,
// missing field being initialized with the defaults
func NewNetworkFailureInjectorWithConfig(uid string, spec v1beta1.NetworkFailureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config *NetworkFailureInjectorConfig) (Injector, error) {
	// iptables driver
	if config.IPTables == nil {
		ipt, err := iptables.New()
		if err != nil {
			return nil, fmt.Errorf("can't initialize iptables driver: %w", err)
		}

		config.IPTables = ipt
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
		config: config,
		spec:   spec,
	}, nil
}

// Inject injects the given network failure into the given container
func (i networkFailureInjector) Inject() {
	var ruleParts []string

	// enter container network namespace
	err := i.container.EnterNetworkNamespace()
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

	// Default to 0.0.0.0/0 if no host has been specified
	hosts := []string{}
	if len(i.spec.Hosts) == 0 {
		hosts = append(hosts, "0.0.0.0/0")
	} else {
		hosts = append(hosts, i.spec.Hosts...)
	}

	// Resolve host
	i.log.Infow("resolving given hosts", "hosts", hosts)

	ips, err := resolveHosts(i.config.DNSClient, hosts)
	if err != nil {
		i.ms.EventInjectFailure(i.container.ID(), i.uid)
		i.ms.MetricInjected(i.container.ID(), i.uid, false, i.kind, []string{})
		i.log.Fatalw("unable to resolve host", "error", err, "host", hosts[0])
	}

	// Inject
	dedicatedChain := i.getDedicatedChainName()
	i.log.Infow("starting the injection", "chain", dedicatedChain)

	// Create a new dedicated chain
	if !i.chainExists(dedicatedChain) {
		i.log.Infow("creating the dedicated iptables chain", "chain", dedicatedChain)

		err = i.config.IPTables.NewChain(iptablesTable, dedicatedChain)
		if err != nil {
			i.ms.EventInjectFailure(i.container.ID(), i.uid)
			i.ms.MetricInjected(i.container.ID(), i.uid, false, i.kind, []string{})
			i.log.Fatalw("error while creating the dedicated chain",
				"error", err,
				"chain", dedicatedChain,
			)
		}
	}

	// Add the dedicated chain jump rule
	ruleParts = []string{"-j", dedicatedChain}
	rule := fmt.Sprintf("iptables -t %s %s", iptablesTable, strings.Join(ruleParts, " "))

	err = i.config.IPTables.AppendUnique(iptablesTable, iptablesOutputChain, ruleParts...)
	if err != nil {
		i.ms.EventInjectFailure(i.container.ID(), i.uid)
		i.ms.MetricInjected(i.container.ID(), i.uid, false, i.kind, []string{})
		i.log.Fatalw("error while injecting the jump rule", "error", err, "rule", rule)
	}

	// Append all rules to the dedicated chain
	for _, ip := range ips {
		// Ignore IPv6 addresses
		if ip.IP.To4() == nil {
			i.log.Infow("non-IPv4 detected, skipping", "ip", ip.String())
			continue
		}

		ruleParts = i.generateRuleParts(ip.String())
		rule := fmt.Sprintf("iptables -t %s -A %s %s", iptablesTable, dedicatedChain, strings.Join(ruleParts, " "))
		i.log.Infow(
			"injecting drop rule",
			"table", iptablesTable,
			"chain", dedicatedChain,
			"ip", ip.String(),
			"port", i.spec.Port,
			"protocol", i.spec.Protocol,
			"probability", i.spec.Probability,
			"rule", rule,
		)

		err = i.config.IPTables.AppendUnique(iptablesTable, dedicatedChain, ruleParts...)
		if err != nil {
			i.ms.EventInjectFailure(i.container.ID(), i.uid)
			i.ms.MetricInjected(i.container.ID(), i.uid, false, i.kind, []string{})
			i.ms.MetricIPTablesRulesInjected(i.container.ID(), i.uid, false, i.kind, []string{})
			i.log.Fatalw("error while injecting the drop rule", "error", err, "rule", rule)

			return
		}

		i.ms.EventWithTags(
			"network failure injected",
			"the following rule has been injected: "+rule,
			[]string{
				"containerID:" + i.container.ID(),
				"UID:" + i.uid,
			},
		)

		i.ms.MetricIPTablesRulesInjected(i.container.ID(), i.uid, true, i.kind, []string{})
	}

	i.ms.MetricInjected(i.container.ID(), i.uid, true, i.kind, []string{})
}

// Clean removes all the injected failures in the given container
func (i networkFailureInjector) Clean() {
	// enter container network namespace
	err := i.container.EnterNetworkNamespace()
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

	dedicatedChain := i.getDedicatedChainName()
	i.log.Infow("starting the cleaning", "chain", dedicatedChain)

	// clear, delete the dedicated chain and its jump rule if it exists
	if i.chainExists(dedicatedChain) {
		ruleParts := []string{"-j", dedicatedChain}

		// delete the jump rule if it exists
		exists, err := i.config.IPTables.Exists(iptablesTable, iptablesOutputChain, ruleParts...)
		if err != nil {
			i.ms.EventCleanFailure(i.container.ID(), i.uid)
			i.ms.MetricCleaned(i.container.ID(), i.uid, false, i.kind, []string{})
			i.log.Fatalw("unable to check if the dedicated chain jump rule exists", "chain", dedicatedChain, "error", err)
		}

		if exists {
			i.log.Infow("deleting the dedicated chain jump rule", "chain", dedicatedChain)

			err = i.config.IPTables.Delete(iptablesTable, iptablesOutputChain, ruleParts...)
			if err != nil {
				i.ms.EventCleanFailure(i.container.ID(), i.uid)
				i.ms.MetricCleaned(i.container.ID(), i.uid, false, i.kind, []string{})
				i.log.Fatalw("failed to clean dedicated chain jump rule", "chain", dedicatedChain, "error", err)
			}
		}

		// clear and delete the dedicated chain
		i.log.Infow("clearing the dedicated chain", "chain", dedicatedChain)

		err = i.config.IPTables.ClearChain(iptablesTable, dedicatedChain)
		if err != nil {
			i.ms.EventCleanFailure(i.container.ID(), i.uid)
			i.ms.MetricCleaned(i.container.ID(), i.uid, false, i.kind, []string{})
			i.log.Fatalw("failed to clean dedicated chain", "chain", dedicatedChain, "error", err)
		}

		i.log.Infow("deleting the dedicated chain", "chain", dedicatedChain)

		err = i.config.IPTables.DeleteChain(iptablesTable, dedicatedChain)
		if err != nil {
			i.ms.EventCleanFailure(i.container.ID(), i.uid)
			i.ms.MetricCleaned(i.container.ID(), i.uid, false, i.kind, []string{})
			i.log.Fatalw("failed to delete dedicated chain", "chain", dedicatedChain, "error", err)
		}
	}

	i.ms.EventWithTags(
		"network failure cleaned",
		"the rules have been cleaned",
		[]string{
			"containerID:" + i.container.ID(),
			"UID:" + i.uid,
		},
	)
	i.ms.MetricCleaned(i.container.ID(), i.uid, true, i.kind, []string{})
}

// generateRuleParts generates the iptables rules to apply
func (i networkFailureInjector) generateRuleParts(ip string) []string {
	var ruleParts = []string{
		"-p", i.spec.Protocol,
		"-d", ip,
		"--dport",
		strconv.Itoa(i.spec.Port),
	}

	//Add modules (if any) here
	ruleParts = append(ruleParts, "-m")
	numModules := 0

	//Probability Module
	if i.spec.Probability != 0 && i.spec.Probability != 100 {
		//Probability expected in decimal format
		var prob = float64(i.spec.Probability) / 100.0
		ruleParts = append(ruleParts,
			"statistic", "--mode", "random", "--probability", fmt.Sprintf("%.2f", prob),
		)
		numModules++
	}

	//If no modules were defined, remove the tailing -m
	if numModules == 0 {
		ruleParts = ruleParts[:len(ruleParts)-1]
	}

	ruleParts = append(ruleParts, "-j", "DROP")

	return ruleParts
}

// getDedicatedChainName crafts the chaos dedicated chain name
// from the failure resource UID
// it basically keeps the first and last part of the UID because
// of the iptables 29-chars chain name limit
func (i networkFailureInjector) getDedicatedChainName() string {
	parts := strings.Split(i.uid, "-")
	shortUID := parts[0] + parts[len(parts)-1]

	return iptablesChaosChainPrefix + shortUID
}

// chainExists returns true if the given chain exists, false otherwise
func (i networkFailureInjector) chainExists(chain string) bool {
	chains, err := i.config.IPTables.ListChains(iptablesTable)
	if err != nil {
		i.log.Fatalw("unable to list iptables chain", "error", err)
	}

	for _, v := range chains {
		if v == chain {
			return true
		}
	}

	return false
}

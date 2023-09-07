// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import (
	"errors"
	"fmt"

	goiptables "github.com/coreos/go-iptables/iptables"
	"go.uber.org/zap"
)

// IPTables is an interface for interacting with target nat firewall/iptables rules
type IPTables interface {
	Clear() error
	LogConntrack() error
	RedirectTo(protocol string, port string, destinationIP string) error
	Intercept(protocol string, port string, cgroupPath string, cgroupClassID string, injectorPodIP string) error
	MarkCgroupPath(cgroupPath string, mark string) error
	MarkClassID(classid string, mark string) error
}

type iptables struct {
	log           *zap.SugaredLogger
	dryRun        bool
	ip            *goiptables.IPTables
	injectedRules []rule
}

type rule struct {
	table    string
	chain    string
	rulespec []string
}

const (
	chaosChainName = "CHAOS-DNS"
)

// NewIPTables returns an implementation of the IPTables interface that can log
func NewIPTables(log *zap.SugaredLogger, dryRun bool) (IPTables, error) {
	ip, err := goiptables.New()

	return &iptables{
		log:           log,
		dryRun:        dryRun,
		ip:            ip,
		injectedRules: []rule{},
	}, err
}

// Clear removes any previously injected rules in any chain and table
func (i *iptables) Clear() error {
	i.log.Infow("deleting injected iptables rules", "chain", chaosChainName)

	if i.dryRun {
		return nil
	}

	// remove previously injected rules
	for _, r := range i.injectedRules {
		i.log.Infow("deleting injected iptables rule", "chain", r.chain, "table", r.table, "rulespec", r.rulespec)

		// skip if it does not exist anymore for idempotency
		exists, err := i.ip.Exists(r.table, r.chain, r.rulespec...)
		if err != nil {
			return err
		}

		if !exists {
			i.log.Infow("iptables rule doesn't exist anymore, skipping cleaning", "table", r.table, "chain", r.chain, "rulespec", r.rulespec)

			continue
		}

		// delete rule
		if err := i.ip.Delete(r.table, r.chain, r.rulespec...); err != nil {
			return err
		}
	}

	// eventually delete the injector dedicated chain
	// the error is ignored here as any remaining rules in the chain, or any remaining
	// jumping rules to that chain, coming from any remaining injector would cause it to error
	_ = i.ip.DeleteChain("nat", chaosChainName)

	return nil
}

// LogConntrack creates a rule logging packets with a new or established connection state,
// usually used to enable the conntrack tracking in non-root network namespaces
func (i *iptables) LogConntrack() error {
	return i.insert("nat", "OUTPUT", "-m", "state", "--state", "new,established", "-j", "LOG")
}

// RedirectTo redirects the matching packets to the given destination IP
func (i *iptables) RedirectTo(protocol string, port string, destinationIP string) error {
	return i.insert("nat", chaosChainName, "-p", protocol, "--dport", port, "-j", "DNAT", "--to-destination", destinationIP+":"+port)
}

// Intercept jumps the matching packets to the injector dedicated chain except for
// packets coming from the injector itself
func (i *iptables) Intercept(protocol string, port string, cgroupPath string, cgroupClassID string, injectorPodIP string) error {
	rulespec := []string{}

	if cgroupPath != "" && cgroupClassID != "" {
		return errors.New("either cgroup path or cgroup class id must be specified, not both")
	}

	// add protocol
	if protocol != "" {
		rulespec = append(rulespec, "-p", protocol)
	}

	// add port
	if port != "" {
		rulespec = append(rulespec, "--dport", port)
	}

	// exclude injector pod IP
	if injectorPodIP != "" {
		rulespec = append(rulespec, "!", "-s", injectorPodIP)
	}

	// add cgroup path filter
	if cgroupPath != "" {
		rulespec = append(rulespec, "-m", "cgroup", "--path", cgroupPath)
	}

	// add cgroup classid filter
	if cgroupClassID != "" {
		rulespec = append(rulespec, "-m", "cgroup", "--cgroup", cgroupClassID)
	}

	rulespec = append(rulespec, "-j", chaosChainName)

	// inject output rule
	if err := i.insert("nat", "OUTPUT", rulespec...); err != nil {
		return err
	}

	// inject prerouting rule only if there's no cgroup path filtering
	// packets going through prerouting chain are not yet associated
	// to a process so there's no possibility to filter on a cgroup at this stage
	if cgroupPath == "" && cgroupClassID == "" {
		if err := i.insert("nat", "PREROUTING", rulespec...); err != nil {
			return err
		}
	}

	return nil
}

// MarkCgroupPath marks the packets created from the given cgroup path with the given mark
func (i *iptables) MarkCgroupPath(cgroupPath string, mark string) error {
	return i.insert("mangle", "POSTROUTING", "-m", "cgroup", "--path", cgroupPath, "-j", "MARK", "--set-mark", mark)
}

// MarkClassID marks the packets created with the given classid with the given mark
func (i *iptables) MarkClassID(classID string, mark string) error {
	return i.insert("mangle", "POSTROUTING", "-m", "cgroup", "--cgroup", classID, "-j", "MARK", "--set-mark", mark)
}

// insert creates a new iptables rule definition, stores it
// for further cleanup and inserts the rule in the given table and chain
// at the first position
func (i *iptables) insert(table string, chain string, rulespec ...string) error {
	i.log.Infow("injecting iptables rule", "table", table, "chain", chain, "rulespec", rulespec)

	if i.dryRun {
		return nil
	}

	// create the injector chain if it does not exist yet and is used here
	if chain == chaosChainName {
		chainExists, err := i.ip.ChainExists(table, chain)
		if err != nil {
			return err
		}

		if !chainExists {
			if err := i.ip.NewChain(table, chain); err != nil {
				return fmt.Errorf("error creating chain %s: %w", chain, err)
			}
		}
	}

	// check if the rule already exists before trying to insert it
	exists, err := i.ip.Exists(table, chain, rulespec...)
	if err != nil {
		return err
	}

	if exists {
		i.log.Infow("iptables rule already exists, skipping", "table", table, "chain", chain, "rulespec", rulespec)

		return nil
	}

	// inject rule
	if err := i.ip.Insert(table, chain, 1, rulespec...); err != nil {
		return fmt.Errorf("error injecting rule: %w", err)
	}

	r := rule{
		table:    table,
		chain:    chain,
		rulespec: rulespec,
	}

	// store rule for further cleanup
	i.injectedRules = append(i.injectedRules, r)

	return nil
}

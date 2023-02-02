// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import (
	"fmt"

	goiptables "github.com/coreos/go-iptables/iptables"
	"go.uber.org/zap"
)

// Iptables is an interface for interacting with target nat firewall/iptables rules
type Iptables interface {
	CreateChain(name string) error
	ClearAndDeleteChain(name string) error
	AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error
	AddWideFilterRule(chain string, protocol string, port string, jump string) error
	AddCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error
	PrependRuleSpec(chain string, rulespec ...string) error
	DeleteRule(chain string, protocol string, port string, jump string) error
	DeleteRuleSpec(chain string, rulespec ...string) error
	DeleteCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error
	ListRules(table string, chain string) ([]string, error)
}

type iptables struct {
	log    *zap.SugaredLogger
	dryRun bool
	ip     *goiptables.IPTables
}

// NewIptables returns an implementation of the Iptables interface that can log
func NewIptables(log *zap.SugaredLogger, dryRun bool) (Iptables, error) {
	ip, err := goiptables.New()

	return iptables{
		log:    log,
		dryRun: dryRun,
		ip:     ip,
	}, err
}

func (i iptables) CreateChain(name string) error {
	if i.dryRun {
		return nil
	}

	if res, _ := i.ip.ChainExists("nat", name); res {
		return nil
	}

	i.log.Infow("creating new iptables chain", "chain name", name)

	return i.ip.NewChain("nat", name)
}

func (i iptables) ClearAndDeleteChain(name string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("deleting iptables chain", "chain name", name)

	return i.ip.ClearAndDeleteChain("nat", name)
}

func (i iptables) AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("creating new iptables rule", "chain name", chain, "protocol", protocol, "port", port, "jump target", jump, "destination", destinationIP)

	return i.ip.AppendUnique("nat", chain, "-p", protocol, "--dport", port, "-j", jump, "--to-destination", fmt.Sprintf("%s:%s", destinationIP, port))
}

func (i iptables) PrependRuleSpec(chain string, rulespec ...string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("creating new iptables rule", "chain name", chain, "rulespec", rulespec)

	// 1 is the first position, not 0
	return i.ip.Insert("nat", chain, 1, rulespec...)
}

// Add a rule with cgroup filter
func (i iptables) AddCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("creating new iptables rule", "table", table, "chain", chain, "cgroup path", cgroupPath, "rulespec", rulespec)

	rulespec = append([]string{"-m", "cgroup", "--path", cgroupPath}, rulespec...)

	return i.ip.AppendUnique(table, chain, rulespec...)
}

func (i iptables) AddWideFilterRule(chain string, protocol string, port string, jump string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("creating new iptables rule", "chain name", chain, "protocol", protocol, "port", port, "jump target", jump)

	return i.ip.AppendUnique("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRule(chain string, protocol string, port string, jump string) error {
	if i.dryRun {
		return nil
	}

	// Why do we check if the jump target exists? A command of the form
	// iptables -t nat -C OUTPUT -p udp --dport 53 -j CHAOS-DNS
	// will actually error if the jump target does not exist. However, you are unable
	// to delete a chain if there are rules that jump to it, so if the target does not exist
	// we can be sure that the rule does not exist.
	if exists, _ := i.ip.ChainExists("nat", jump); !exists {
		return nil
	}

	return i.DeleteRuleSpec("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRuleSpec(chain string, rulespec ...string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("deleting iptables rule", "chain name", chain, "rulespec", rulespec)

	if exists, _ := i.ip.ChainExists("nat", chain); !exists {
		return nil
	}

	return i.ip.DeleteIfExists("nat", chain, rulespec...)
}

// Delete a rule with cgroup filter
func (i iptables) DeleteCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error {
	if i.dryRun {
		return nil
	}

	i.log.Infow("deleting iptables rule", "chain name", chain, "cgroup path", cgroupPath, "rulespec", rulespec)

	if exists, _ := i.ip.ChainExists(table, chain); !exists {
		return nil
	}

	rulespec = append([]string{"-m", "cgroup", "--path", cgroupPath}, rulespec...)

	return i.ip.DeleteIfExists(table, chain, rulespec...)
}

// ListRules lists the rules of the given table and chain
func (i iptables) ListRules(table string, chain string) ([]string, error) {
	return i.ip.List(table, chain)
}

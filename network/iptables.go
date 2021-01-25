// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package network

import (
	"fmt"
	"strconv"

	goiptables "github.com/coreos/go-iptables/iptables"
	"go.uber.org/zap"
)

// Iptables is an interface for interacting with host firewall/iptables rules
type Iptables interface {
	CreateChain(name string) error
	DeleteChain(name string) error
	AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error
	AddRule(chain string, protocol string, port string, jump string) error
	DeleteRule(chain string, protocol string, port string, jump string) error
	DeleteRuleByNum(chain string, rulenum int) error
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

	return i.ip.NewChain("nat", name)
}
func (i iptables) DeleteChain(name string) error {
	if i.dryRun {
		return nil
	}

	if exists, _ := i.ip.ChainExists("nat", name); !exists {
		return nil
	}

	return i.ip.DeleteChain("nat", name)
}

func (i iptables) AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error {
	if i.dryRun {
		return nil
	}

	return i.ip.Append("nat", chain, "-p", protocol, "--dport", port, "-j", jump, "--to-destination", fmt.Sprintf("%s:%s", destinationIP, port))
}

func (i iptables) AddRule(chain string, protocol string, port string, jump string) error {
	if i.dryRun {
		return nil
	}

	return i.ip.Append("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRule(chain string, protocol string, port string, jump string) error {
	if i.dryRun {
		return nil
	}

	if exists, _ := i.ip.ChainExists("nat", chain); !exists {
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

	return i.ip.DeleteIfExists("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRuleByNum(chain string, rulenum int) error {
	if i.dryRun {
		return nil
	}

	if exists, _ := i.ip.ChainExists("nat", chain); !exists {
		return nil
	}

	return i.ip.DeleteIfExists("nat", chain, strconv.Itoa(rulenum))
}

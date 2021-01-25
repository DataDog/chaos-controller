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

// const iptablesPath = "/sbin/iptables"

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
	return i.ip.NewChain("nat", name)
}
func (i iptables) DeleteChain(name string) error {
	return i.ip.DeleteChain("nat", name)
}

func (i iptables) AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error {
	return i.ip.Append("nat", chain, "-p", protocol, "--dport", port, "-j", jump, "--to-destination", fmt.Sprintf("%s:%s", destinationIP, port))
}

func (i iptables) AddRule(chain string, protocol string, port string, jump string) error {
	return i.ip.Append("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRule(chain string, protocol string, port string, jump string) error {
	return i.ip.DeleteIfExists("nat", chain, "-p", protocol, "--dport", port, "-j", jump)
}

func (i iptables) DeleteRuleByNum(chain string, rulenum int) error {
	return i.ip.DeleteIfExists("nat", chain, strconv.Itoa(rulenum))
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package network

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

const iptablesPath = "/sbin/iptables"

// Iptables is an interface for interacting with host firewall/iptables rules
type Iptables interface {
	CreateChain(name string) error
	DeleteChain(name string) error
	AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error
	AddRule(chain string, protocol string, port string, jump string) error
	DeleteRule(chain string, protocol string, port string, jump string) error
	DeleteRuleByNum(chain string, rulenum int) error
}

type iptablesExecuter interface {
	Run(args ...string) (exitCode int, stdout string, stderr error)
}

type defaultIptablesExecuter struct {
	log    *zap.SugaredLogger
	dryRun bool
}

// Run executes the given args using the iptables command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e defaultIptablesExecuter) Run(args ...string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(iptablesPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	e.log.Infof("running iptables command: %v", cmd.String())

	// early exit if dry-run mode is enabled
	if e.dryRun {
		return 0, "", nil
	}

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	return cmd.ProcessState.ExitCode(), stdout.String(), err
}

type iptables struct {
	executer iptablesExecuter
}

// NewIptables returns an implementation of the Iptables interface that can log
func NewIptables(log *zap.SugaredLogger, dryRun bool) Iptables {
	return iptables{
		executer: defaultIptablesExecuter{
			log:    log,
			dryRun: dryRun,
		},
	}
}

func (i iptables) CreateChain(name string) error {
	_, _, err := i.executer.Run(buildIptablesCmd("-N", name, "", "", "", "", "")...)
	return err
}
func (i iptables) DeleteChain(name string) error {
	_, _, err := i.executer.Run(buildIptablesCmd("-X", name, "", "", "", "", "")...)
	return err
}

func (i iptables) AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error {
	_, _, err := i.executer.Run(buildIptablesCmd("-A", chain, protocol, port, jump, destinationIP, "")...)
	return err
}

func (i iptables) AddRule(chain string, protocol string, port string, jump string) error {
	return i.AddRuleWithIP(chain, protocol, port, jump, "")
}

func (i iptables) DeleteRule(chain string, protocol string, port string, jump string) error {
	_, _, err := i.executer.Run(buildIptablesCmd("-D", chain, protocol, port, jump, "", "")...)
	return err
}

func (i iptables) DeleteRuleByNum(chain string, rulenum int) error {
	_, _, err := i.executer.Run(buildIptablesCmd("-D", chain, "", "", "", "", fmt.Sprint(rulenum))...)
	return err
}

func buildIptablesCmd(cmdType string, chain string, protocol string, port string, jump string, destinationIP string, parameters string) []string {
	cmd := fmt.Sprintf("-t nat %s %s", cmdType, chain)

	if protocol != "" {
		cmd += fmt.Sprintf(" -p %s", protocol)
	}

	if port != "" {
		cmd += fmt.Sprintf(" --dport %s", port)
	}

	if jump != "" {
		cmd += fmt.Sprintf(" -j %s", jump)
	}

	if destinationIP != "" {
		cmd += fmt.Sprintf(" --to-destination %s:%s", destinationIP, port)
	}

	if parameters != "" {
		cmd += fmt.Sprintf(" %s", parameters)
	}

	return strings.Split(cmd, " ")
}

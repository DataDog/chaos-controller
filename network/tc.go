// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package network

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const tcPath = "/sbin/tc"

type connState string

var (
	ConnStateUndefined   connState = ""
	ConnStateNew         connState = "+trk+new"
	ConnStateEstablished connState = "+trk+est"
)

func NewConnState(hostConnState string) connState {
	var connState connState

	switch hostConnState {
	case "new":
		connState = ConnStateNew
	case "est":
		connState = ConnStateEstablished
	default:
		connState = ConnStateUndefined
	}

	return connState
}

// TrafficController is an interface being able to interact with the host
// queueing discipline
type TrafficController interface {
	AddNetem(ifaces []string, parent string, handle uint32, delay time.Duration, delayJitter time.Duration, drop int, corrupt int, duplicate int) error
	AddPrio(ifaces []string, parent string, handle uint32, bands uint32, priomap [16]uint32) error
	AddFilter(ifaces []string, parent string, priority uint32, handle uint32, srcIP, dstIP *net.IPNet, srcPort, dstPort int, protocol Protocol, connState connState, flowid string) error
	DeleteFilter(iface string, priority uint32) error
	AddCgroupFilter(ifaces []string, parent string, handle uint32) error
	AddOutputLimit(ifaces []string, parent string, handle uint32, bytesPerSec uint) error
	ClearQdisc(ifaces []string) error
}

type tcExecuter interface {
	Run(args ...string) (exitCode int, stdout string, stderr error)
}

type defaultTcExecuter struct {
	log    *zap.SugaredLogger
	dryRun bool
}

// Run executes the given args using the tc command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e defaultTcExecuter) Run(args ...string) (int, string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(tcPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	e.log.Debugf("running tc command: %v", cmd.String())

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

type tc struct {
	executer tcExecuter
}

// NewTrafficController creates a standard traffic controller using tc
// and being able to log
func NewTrafficController(log *zap.SugaredLogger, dryRun bool) TrafficController {
	return tc{
		executer: defaultTcExecuter{
			log:    log,
			dryRun: dryRun,
		},
	}
}

func (t tc) AddNetem(ifaces []string, parent string, handle uint32, delay time.Duration, delayJitter time.Duration, drop int, corrupt int, duplicate int) error {
	params := ""

	if delay.Milliseconds() != 0 {
		params = fmt.Sprintf("%s delay %dms %dms distribution normal", params, delay.Milliseconds(), delayJitter.Milliseconds())
	}

	if drop != 0 {
		params = fmt.Sprintf("%s loss %d%%", params, drop)
	}

	if duplicate != 0 {
		params = fmt.Sprintf("%s duplicate %d%%", params, duplicate)
	}

	if corrupt != 0 {
		params = fmt.Sprintf("%s corrupt %d%%", params, corrupt)
	}

	params = strings.TrimPrefix(params, " ")

	for _, iface := range ifaces {
		if _, _, err := t.executer.Run(buildCmd("qdisc", iface, parent, "", 0, handle, "netem", params)...); err != nil {
			return err
		}
	}

	return nil
}

func (t tc) AddPrio(ifaces []string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	priomapStr := ""
	for _, bit := range priomap {
		priomapStr += fmt.Sprintf(" %d", bit)
	}

	priomapStr = strings.TrimSpace(priomapStr)
	params := fmt.Sprintf("bands %d priomap %s", bands, priomapStr)

	for _, iface := range ifaces {
		if _, _, err := t.executer.Run(buildCmd("qdisc", iface, parent, "", 0, handle, "prio", params)...); err != nil {
			return err
		}
	}

	return nil
}

func (t tc) AddOutputLimit(ifaces []string, parent string, handle uint32, bytesPerSec uint) error {
	// `latency` is max length of time a packet can sit in the queue before being sent; 50ms should be plenty
	// `burst` is the number of bytes that can be sent at unlimited speed before the rate limiting kicks in,
	// so again we'll be safe by setting `burst` to be the same as `rate` (should be more than enough)
	// for more info, see the following:
	//   - https://unix.stackexchange.com/questions/100785/bucket-size-in-tbf
	//   - https://linux.die.net/man/8/tc-tbf
	for _, iface := range ifaces {
		if _, _, err := t.executer.Run(buildCmd("qdisc", iface, parent, "", 0, handle, "tbf", fmt.Sprintf("rate %d latency 50ms burst %d", bytesPerSec, bytesPerSec))...); err != nil {
			return err
		}
	}

	return nil
}

func (t tc) ClearQdisc(ifaces []string) error {
	for _, iface := range ifaces {
		// tc exits with code 2 when the qdisc does not exist anymore
		if exitCode, _, err := t.executer.Run(strings.Split(fmt.Sprintf("qdisc del dev %s root", iface), " ")...); err != nil && exitCode != 2 {
			return err
		}
	}

	return nil
}

// AddFilter generates a filter to redirect the traffic matching the given ip, port and protocol to the given flowid
// this function relies on the tc flower (https://man7.org/linux/man-pages/man8/tc-flower.8.html) filtering module
func (t tc) AddFilter(ifaces []string, parent string, priority uint32, handle uint32, srcIP, dstIP *net.IPNet, srcPort, dstPort int, protocol Protocol, connState connState, flowid string) error {
	var params, filterProtocol string

	// match protocol if specified, default to tcp otherwise
	switch protocol {
	case TCP, UDP:
		filterProtocol = "ip"
		params += fmt.Sprintf("ip_proto %s ", strings.ToLower(string(protocol)))
	case ARP:
		filterProtocol = "arp"
	default:
		return fmt.Errorf("unexpected protocol: %s", protocol)
	}

	// match ip if specified
	if srcIP != nil && srcIP.String() != "0.0.0.0/0" {
		params += fmt.Sprintf("src_ip %s ", srcIP.String())
	}

	if dstIP != nil && dstIP.String() != "0.0.0.0/0" {
		params += fmt.Sprintf("dst_ip %s ", dstIP.String())
	}

	// match port if specified
	if srcPort != 0 {
		params += fmt.Sprintf("src_port %s ", strconv.Itoa(srcPort))
	}

	if dstPort != 0 {
		params += fmt.Sprintf("dst_port %s ", strconv.Itoa(dstPort))
	}

	// match conn state if specified
	if connState != ConnStateUndefined {
		params += fmt.Sprintf("ct_state %s ", connState)
	}

	params += fmt.Sprintf("flowid %s", flowid)

	for _, iface := range ifaces {
		if _, _, err := t.executer.Run(buildCmd("filter", iface, parent, filterProtocol, priority, handle, "flower", params)...); err != nil {
			return err
		}
	}

	return nil
}

func (t tc) DeleteFilter(iface string, priority uint32) error {
	if _, _, err := t.executer.Run("filter", "delete", "dev", iface, "priority", fmt.Sprintf("%d", priority)); err != nil {
		return err
	}

	return nil
}

// AddCgroupFilter generates a cgroup filter
func (t tc) AddCgroupFilter(ifaces []string, parent string, handle uint32) error {
	for _, iface := range ifaces {
		if _, _, err := t.executer.Run(buildCmd("filter", iface, parent, "", 0, handle, "cgroup", "")...); err != nil {
			return err
		}
	}

	return nil
}

func buildCmd(module string, iface string, parent string, protocol string, priority uint32, handle uint32, kind string, parameters string) []string {
	cmd := fmt.Sprintf("%s add dev %s", module, iface)

	if protocol != "" {
		cmd += fmt.Sprintf(" protocol %s", protocol)
	}

	if priority != 0 {
		cmd += fmt.Sprintf(" priority %d", priority)
	}

	// parent
	if parent == "root" {
		cmd += " root"
	} else {
		cmd += fmt.Sprintf(" parent %s", parent)
	}

	// handle
	if handle != 0 {
		cmd += fmt.Sprintf(" handle %d:", handle)
	}

	// kind
	cmd += fmt.Sprintf(" %s", kind)

	// parameters
	if parameters != "" {
		cmd += fmt.Sprintf(" %s", parameters)
	}

	return strings.Split(cmd, " ")
}

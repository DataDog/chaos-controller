// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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

// TrafficController is an interface being able to interact with the host
// queueing discipline
type TrafficController interface {
	AddDelay(iface string, parent string, handle uint32, delay time.Duration) error
	AddPrio(iface string, parent string, handle uint32, bands uint32, priomap [16]uint32) error
	AddFilter(iface string, parent string, handle uint32, ip *net.IPNet, port int, flowid string) error
	AddOutputLimit(iface string, parent string, handle uint32, bytesPerSec uint) error
	ClearQdisc(iface string) error
	IsQdiscCleared(iface string) (bool, error)
}

type tcExecuter interface {
	Run(args ...string) (stdout string, stderr error)
}

type defaultTcExecuter struct{}

// Run executes the given args using the tc command
// and returns a wrapped error containing both the error returned by the execution and
// the stderr content
func (e defaultTcExecuter) Run(args ...string) (string, error) {
	// parse args and execute
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := exec.Command(tcPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// run command
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("encountered error (%w) using args (%s): %s", err, args, stderr.String())
	}

	return stdout.String(), err
}

type tc struct {
	log      *zap.SugaredLogger
	executer tcExecuter
}

// NewTrafficController creates a standard traffic controller using tc
// and being able to log
func NewTrafficController(log *zap.SugaredLogger) TrafficController {
	return tc{
		log:      log,
		executer: defaultTcExecuter{},
	}
}

func (t tc) AddDelay(iface string, parent string, handle uint32, delay time.Duration) error {
	_, err := t.executer.Run(buildCmd("qdisc", iface, parent, handle, "netem", fmt.Sprintf("delay %s", delay))...)

	return err
}

func (t tc) AddPrio(iface string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	priomapStr := ""
	for _, bit := range priomap {
		priomapStr += fmt.Sprintf(" %d", bit)
	}

	priomapStr = strings.TrimSpace(priomapStr)
	params := fmt.Sprintf("bands %d priomap %s", bands, priomapStr)
	_, err := t.executer.Run(buildCmd("qdisc", iface, parent, handle, "prio", params)...)

	return err
}

func (t tc) AddOutputLimit(iface string, parent string, handle uint32, bytesPerSec uint) error {
	// `latency` is max length of time a packet can sit in the queue before being sent; 50ms should be plenty
	// `burst` is the number of bytes that can be sent at unlimited speed before the rate limiting kicks in,
	// so again we'll be safe by setting `burst` to be the same as `rate` (should be more than enough)
	// for more info, see the following:
	//   - https://unix.stackexchange.com/questions/100785/bucket-size-in-tbf
	//   - https://linux.die.net/man/8/tc-tbf
	mycmd := buildCmd("qdisc", iface, parent, handle, "tbf", fmt.Sprintf("rate %d latency 50ms burst %d", bytesPerSec, bytesPerSec))

	_, err := t.executer.Run(mycmd...)

	return err
}

func (t tc) ClearQdisc(iface string) error {
	_, err := t.executer.Run(strings.Split(fmt.Sprintf("qdisc del dev %s root", iface), " ")...)

	return err
}

func (t tc) AddFilter(iface string, parent string, handle uint32, ip *net.IPNet, port int, flowid string) error {
	params := fmt.Sprintf("match ip dst %s ", ip.String())
	if port != 0 {
		params += fmt.Sprintf("match ip dport %s 0xffff ", strconv.Itoa(port))
	}

	params += fmt.Sprintf("flowid %s", flowid)
	_, err := t.executer.Run(buildCmd("filter", iface, parent, handle, "u32", params)...)

	return err
}

func (t tc) IsQdiscCleared(iface string) (bool, error) {
	cmd := fmt.Sprintf("qdisc show dev %s", iface)

	// list interface qdiscs
	out, err := t.executer.Run(strings.Split(cmd, " ")...)
	if err != nil {
		return false, fmt.Errorf("error getting %s qdisc info: %w", iface, err)
	}

	// ensure the root has no qdisc
	return strings.HasPrefix(out, "qdisc noqueue 0: root refcnt"), nil
}

func buildCmd(module string, iface string, parent string, handle uint32, kind string, parameters string) []string {
	cmd := fmt.Sprintf("%s add dev %s", module, iface)

	// parent
	if parent == "root" {
		cmd += fmt.Sprintf(" root")
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
	cmd += fmt.Sprintf(" %s", parameters)

	return strings.Split(cmd, " ")
}

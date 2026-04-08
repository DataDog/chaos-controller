// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package bpfdisrupt

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/DataDog/chaos-controller/network"
	"go.uber.org/zap"
)

const (
	// BPFObjectPath is the path to the compiled BPF disruption object in the injector container.
	BPFObjectPath = "/usr/local/bin/bpf-network-disruption.bpf.o"
	// BPFConfigPath is the path to the BPF map configuration binary in the injector container.
	BPFConfigPath = "/usr/local/bin/bpf-network-disruption"
	// EgressSection is the BPF program section for egress classification.
	EgressSection = "tc_egress_disruption"
	// IngressSection is the BPF program section for ingress DirectAction.
	IngressSection = "tc_ingress_disruption"
	// EgressFlowID is the tc flowid for disrupted egress traffic (band 1:4 of root prio).
	EgressFlowID = "1:4"
	// EgressParent is the tc parent for the egress BPF classifier (root prio qdisc).
	EgressParent = "1:0"
)

// CmdRunner is an interface for running external commands.
// This abstracts the command execution for testability.
type CmdRunner interface {
	Run(args []string) (exitCode int, stdout string, err error)
}

// Engine manages the BPF-based network disruption packet plane.
// It encapsulates clsact qdisc lifecycle, IFB device management,
// BPF program attachment, and LPM trie map population.
type Engine struct {
	tc         network.TrafficController
	nl         network.NetlinkAdapter
	cmdRunner  CmdRunner
	log        *zap.SugaredLogger
	ifbName    string   // "" if no IFB device created
	ifbIndex   int      // IFB device ifindex (for bpf_redirect)
	interfaces []string // target interfaces
	mu         sync.Mutex // protects attached, ifbName, ifbIndex
	attached   bool
}

// NewEngine creates a new BPF disruption engine.
func NewEngine(tc network.TrafficController, nl network.NetlinkAdapter, cmdRunner CmdRunner, log *zap.SugaredLogger) *Engine {
	return &Engine{
		tc:        tc,
		nl:        nl,
		cmdRunner: cmdRunner,
		log:       log,
	}
}

// IFBName returns the name of the IFB device, or "" if none was created.
func (e *Engine) IFBName() string {
	return e.ifbName
}

// Attached returns whether the engine is currently attached.
func (e *Engine) Attached() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.attached
}

// Attach sets up the BPF disruption engine on the given interfaces.
// It creates a clsact qdisc, attaches egress and ingress BPF programs,
// optionally creates an IFB device for ingress shaping, and populates the LPM trie.
func (e *Engine) Attach(interfaces []string, rules []Rule, disruptionUID string, needsIngressShaping bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.interfaces = interfaces

	// Create IFB device if ingress shaping (delay/jitter/bandwidth) is needed
	if needsIngressShaping {
		uid8 := disruptionUID
		if len(uid8) > 8 {
			uid8 = uid8[:8]
		}

		e.ifbName = "ifb-" + uid8

		ifbLink, err := e.nl.AddIFBDevice(e.ifbName)
		if err != nil {
			return fmt.Errorf("error creating IFB device %s: %w", e.ifbName, err)
		}

		e.ifbIndex = ifbLink.Index()
		e.log.Infof("created IFB device %s (ifindex %d) for ingress shaping", e.ifbName, e.ifbIndex)
	}

	// Attach clsact qdisc for ingress BPF hook
	if err := e.tc.AddClsact(interfaces); err != nil {
		return fmt.Errorf("error adding clsact qdisc: %w", err)
	}

	// Attach egress BPF classifier on the root prio qdisc (parent 1:0)
	// This replaces the per-IP flower filters with a single BPF LPM trie lookup
	if err := e.tc.AddBPFFilter(interfaces, EgressParent, BPFObjectPath, EgressFlowID, EgressSection); err != nil {
		return fmt.Errorf("error attaching egress BPF filter: %w", err)
	}

	// Attach ingress BPF with DirectAction on clsact
	if err := e.tc.AddIngressBPFFilter(interfaces, BPFObjectPath, IngressSection); err != nil {
		return fmt.Errorf("error attaching ingress BPF filter: %w", err)
	}

	// Populate the LPM trie map with rules
	if err := e.populateRules(rules); err != nil {
		return fmt.Errorf("error populating disruption rules: %w", err)
	}

	e.attached = true

	return nil
}

// Detach removes the BPF disruption engine from all interfaces.
// It removes the clsact qdisc (which removes ingress + egress BPF filters)
// and deletes the IFB device if one was created.
// The egress BPF filter on the root prio is removed when ClearQdisc removes the root prio.
func (e *Engine) Detach() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.attached {
		return nil
	}

	// Remove clsact qdisc (removes ingress BPF filter)
	if err := e.tc.ClearIngressQdisc(e.interfaces); err != nil {
		e.log.Warnw("error removing clsact qdisc", "error", err)
	}

	// Delete IFB device if created
	if e.ifbName != "" {
		if err := e.nl.DeleteIFBDevice(e.ifbName); err != nil {
			e.log.Warnw("error deleting IFB device", "device", e.ifbName, "error", err)
		}

		e.ifbName = ""
		e.ifbIndex = 0
	}

	e.attached = false

	return nil
}

// UpdateRules atomically replaces all rules in the BPF LPM trie map.
// Called by DNS/service watchers when resolved IPs change.
func (e *Engine) UpdateRules(rules []Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.attached {
		return fmt.Errorf("engine not attached")
	}

	// Clear existing rules
	if _, _, err := e.cmdRunner.Run([]string{BPFConfigPath, "--clear"}); err != nil {
		return fmt.Errorf("error clearing disruption rules: %w", err)
	}

	// Re-populate with new rules
	return e.populateRules(rules)
}

// populateRules invokes the BPF config binary to add each rule to the LPM trie.
func (e *Engine) populateRules(rules []Rule) error {
	for _, rule := range rules {
		args := []string{
			BPFConfigPath,
			"--ip", rule.CIDR,
			"--action", rule.Action.String(),
		}

		if rule.Action == ActionDrop && rule.DropPct > 0 {
			args = append(args, "--drop-pct", strconv.Itoa(rule.DropPct))
		}

		if rule.Direction == DirIngress && e.ifbIndex > 0 && rule.Action == ActionDisrupt {
			args = append(args, "--ifb-ifindex", strconv.Itoa(e.ifbIndex))
		}

		// L4 port/protocol matching
		if rule.Port > 0 {
			if rule.Direction == DirIngress {
				args = append(args, "--src-port", strconv.Itoa(rule.Port))
			} else {
				args = append(args, "--dst-port", strconv.Itoa(rule.Port))
			}
		}

		if rule.Protocol != "" {
			args = append(args, "--protocol", rule.Protocol)
		}

		if _, _, err := e.cmdRunner.Run(args); err != nil {
			return fmt.Errorf("error adding rule for %s: %w", rule.CIDR, err)
		}
	}

	return nil
}

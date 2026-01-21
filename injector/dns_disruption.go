// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/types"
)

// dnsDisruptionInjector describes a DNS disruption injector
type dnsDisruptionInjector struct {
	spec      v1beta1.DNSDisruptionSpec
	config    DNSDisruptionInjectorConfig
	responder *network.DNSResponder
}

// DNSDisruptionInjectorConfig contains needed drivers to create a DNS disruption injector
type DNSDisruptionInjectorConfig struct {
	Config
	IPTables network.IPTables
}

// NewDNSDisruptionInjector creates a DNSDisruptionInjector object with the given config
func NewDNSDisruptionInjector(spec v1beta1.DNSDisruptionSpec, config DNSDisruptionInjectorConfig) Injector {
	return &dnsDisruptionInjector{
		spec:   spec,
		config: config,
	}
}

func (i *dnsDisruptionInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *dnsDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDNSDisruption
}

// Inject starts the DNS responder and configures IPTables to redirect DNS traffic
func (i *dnsDisruptionInjector) Inject() error {
	i.config.Log.Infow("injecting DNS disruption",
		tags.ContainerKey, i.config.TargetContainer.Name(),
		"domains", i.spec.Domains,
		"failureMode", i.spec.FailureMode,
	)

	// get the chaos pod node IP from the environment variable
	chaosPodIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	// Determine ports
	dnsPort := i.spec.Port
	if dnsPort == 0 {
		dnsPort = 53
	}

	// DNS responder runs on a different port to avoid conflicts
	// We'll redirect from dnsPort to responderPort via IPTables
	responderPort := 5353

	// Determine protocol
	protocol := strings.ToLower(i.spec.Protocol)
	if protocol == "" {
		protocol = "both"
	}

	i.config.Log.Infow("entering target network namespace to configure DNS disruption",
		"dnsPort", dnsPort,
		"responderPort", responderPort,
		"protocol", protocol,
	)

	// Enter target container's network namespace
	// IMPORTANT: DNS responder must start INSIDE the target namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the target container network namespace: %w", err)
	}

	// Create and start DNS responder in the target's network namespace
	responderConfig := network.DNSResponderConfig{
		TargetDomains: i.spec.Domains,
		FailureMode:   i.spec.FailureMode,
		Port:          responderPort,
		Protocol:      protocol,
		Logger:        i.config.Log,
	}

	i.responder = network.NewDNSResponder(responderConfig)

	i.config.Log.Infow("starting DNS responder in target network namespace", "port", responderPort)

	if err := i.responder.Start(); err != nil {
		// Exit namespace before returning error
		_ = i.config.Netns.Exit()
		return fmt.Errorf("failed to start DNS responder: %w", err)
	}

	// Configure IPTables to intercept DNS traffic based on protocol
	// IMPORTANT: RedirectTo must be called before Intercept to create the CHAOS-DNS chain first
	// Note: We're already in the target's network namespace, so we intercept ALL DNS traffic (no cgroup filtering)
	if protocol == "udp" || protocol == "both" {
		// Redirect to DNS responder (creates CHAOS-DNS chain)
		// Note: RedirectTo uses responderPort for the destination, dnsPort is only for Intercept matching
		if err := i.config.IPTables.RedirectTo("udp", fmt.Sprintf("%d", responderPort), chaosPodIP); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()
			return fmt.Errorf("failed to configure IPTables for UDP redirection: %w", err)
		}

		// Intercept ALL UDP DNS traffic in this namespace (no cgroup filtering since we're in target namespace)
		if err := i.config.IPTables.Intercept("udp", fmt.Sprintf("%d", dnsPort), "", "", ""); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()
			return fmt.Errorf("failed to configure IPTables for UDP interception: %w", err)
		}
	}

	if protocol == "tcp" || protocol == "both" {
		// Redirect to DNS responder (creates CHAOS-DNS chain if not already created)
		// Note: RedirectTo uses responderPort for the destination, dnsPort is only for Intercept matching
		if err := i.config.IPTables.RedirectTo("tcp", fmt.Sprintf("%d", responderPort), chaosPodIP); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()
			return fmt.Errorf("failed to configure IPTables for TCP redirection: %w", err)
		}

		// Intercept ALL TCP DNS traffic in this namespace (no cgroup filtering since we're in target namespace)
		if err := i.config.IPTables.Intercept("tcp", fmt.Sprintf("%d", dnsPort), "", "", ""); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()
			return fmt.Errorf("failed to configure IPTables for TCP interception: %w", err)
		}
	}

	// Exit target container's network namespace
	if err := i.config.Netns.Exit(); err != nil {
		i.stopResponder()
		return fmt.Errorf("unable to exit the target container network namespace: %w", err)
	}

	i.config.Log.Infow("DNS disruption injected successfully",
		"domains", i.spec.Domains,
		"failureMode", i.spec.FailureMode,
	)

	// DNS responder is running in background goroutines
	// Return now so the pod can become ready
	// The responder will keep running until Clean() is called
	return nil
}

// stopResponder stops the DNS responder and logs any errors
func (i *dnsDisruptionInjector) stopResponder() {
	if i.responder != nil {
		if err := i.responder.Stop(); err != nil {
			i.config.Log.Errorw("error stopping DNS responder", "error", err)
		}
	}
}

func (i *dnsDisruptionInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes IPTables rules and stops the DNS responder
func (i *dnsDisruptionInjector) Clean() error {
	i.config.Log.Infow("cleaning up DNS disruption")

	var errs []error

	// Stop DNS responder
	if i.responder != nil {
		i.config.Log.Infow("stopping DNS responder")
		if err := i.responder.Stop(); err != nil {
			i.config.Log.Errorw("error stopping DNS responder during cleanup", "error", err)
			errs = append(errs, fmt.Errorf("failed to stop DNS responder: %w", err))
		}
		i.responder = nil
	}

	// Enter target container's network namespace to clear IPTables rules
	i.config.Log.Infow("entering target network namespace to clear IPTables rules")
	if err := i.config.Netns.Enter(); err != nil {
		i.config.Log.Errorw("error entering network namespace during cleanup", "error", err)
		errs = append(errs, fmt.Errorf("failed to enter network namespace: %w", err))
	} else {
		// Clear IPTables rules
		i.config.Log.Infow("clearing IPTables rules")
		if err := i.config.IPTables.Clear(); err != nil {
			i.config.Log.Errorw("error clearing IPTables rules", "error", err)
			errs = append(errs, fmt.Errorf("failed to clear IPTables rules: %w", err))
		}

		// Exit target container's network namespace
		if err := i.config.Netns.Exit(); err != nil {
			i.config.Log.Errorw("error exiting network namespace during cleanup", "error", err)
			errs = append(errs, fmt.Errorf("failed to exit network namespace: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during DNS disruption cleanup: %v", errs)
	}

	i.config.Log.Infow("DNS disruption cleaned up successfully")

	return nil
}

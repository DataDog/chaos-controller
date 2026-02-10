// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/types"
)

var (
	launchDNSServer    sync.Once
	launchDNSServerErr error
)

// ResetDNSServerOnce resets the global DNS server sync.Once to allow restarts (e.g., pulsing).
// Tests also use this to avoid cross-test state.
func ResetDNSServerOnce() {
	launchDNSServer = sync.Once{}
	launchDNSServerErr = nil
}

// dnsDisruptionInjector describes a DNS disruption injector
type dnsDisruptionInjector struct {
	spec      v1beta1.DNSDisruptionSpec
	config    DNSDisruptionInjectorConfig
	responder network.DNSResponder
}

// DNSDisruptionInjectorConfig contains needed drivers to create a DNS disruption injector
type DNSDisruptionInjectorConfig struct {
	Config
	IPTables  network.IPTables
	Responder network.DNSResponder // Optional: for testing. If nil, creates real DNS responder
}

// NewDNSDisruptionInjector creates a DNSDisruptionInjector object with the given config
func NewDNSDisruptionInjector(spec v1beta1.DNSDisruptionSpec, config DNSDisruptionInjectorConfig) Injector {
	return &dnsDisruptionInjector{
		spec:   spec,
		config: config,
	}
}

// convertToResponderRecords converts API spec records to DNSRecordEntry format
func convertToResponderRecords(records []v1beta1.DNSRecord) []network.DNSRecordEntry {
	entries := make([]network.DNSRecordEntry, len(records))
	for i, record := range records {
		entries[i] = network.DNSRecordEntry{
			Hostname:   record.Hostname,
			RecordType: strings.ToUpper(strings.TrimSpace(record.Record.Type)),
			Value:      record.Record.Value,
			TTL:        record.Record.TTL,
		}
	}

	return entries
}

// getUpstreamDNSFromResolvConf reads the target pod's /etc/resolv.conf to get all upstream DNS servers
// Returns a comma-separated list of nameservers, or "8.8.8.8:53" as fallback
func getUpstreamDNSFromResolvConf(resolvConfPath string) string {
	const defaultDNS = "8.8.8.8:53"

	file, err := os.Open(resolvConfPath)
	if err != nil {
		return defaultDNS
	}

	defer func() {
		_ = file.Close()
	}()

	var nameservers []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Look for nameserver entries
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				nameserver := fields[1]

				// If already has brackets (e.g., [::1]:53), it's formatted correctly
				if strings.HasPrefix(nameserver, "[") {
					nameservers = append(nameservers, nameserver)
					continue
				}

				// Try to parse as IP to detect IPv6
				if ip := net.ParseIP(nameserver); ip != nil && ip.To4() == nil {
					// It's an IPv6 address without brackets - add them with port
					nameserver = fmt.Sprintf("[%s]:53", nameserver)
				} else if !strings.Contains(nameserver, ":") {
					// IPv4 without port - add port
					nameserver += ":53"
				}

				nameservers = append(nameservers, nameserver)
			}
		}
	}

	// Return all nameservers as comma-separated list for redundancy
	if len(nameservers) > 0 {
		return strings.Join(nameservers, ",")
	}

	return defaultDNS
}

func (i *dnsDisruptionInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *dnsDisruptionInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDNSDisruption
}

// Inject starts the DNS responder and configures IPTables to redirect DNS traffic
func (i *dnsDisruptionInjector) Inject() error {
	if i.config.Disruption.Level != types.DisruptionLevelPod {
		return fmt.Errorf("DNS disruptions can only be applied at the pod level")
	}

	// Dry run mode - log what would happen without making changes
	if i.config.Disruption.DryRun {
		i.config.Log.Infow("injecting DNS disruption in dry run mode",
			tags.ContainerKey, i.config.TargetContainer.Name(),
			tags.DNSRecordCountKey, len(i.spec.Records),
			tags.PortKey, i.spec.Port,
			tags.ProtocolKey, i.spec.Protocol,
		)

		// Log each record that would be disrupted
		for _, record := range i.spec.Records {
			i.config.Log.Infow("would disrupt DNS record",
				tags.DNSHostnameKey, record.Hostname,
				tags.TypeKey, record.Record.Type,
				tags.ValueKey, record.Record.Value,
				tags.DNSTTLKey, record.Record.TTL,
			)
		}

		i.config.Log.Infow("dry run: would start DNS responder and configure IPTables rules")

		return nil
	}

	i.config.Log.Infow("injecting DNS disruption",
		tags.ContainerKey, i.config.TargetContainer.Name(),
		tags.DNSRecordCountKey, len(i.spec.Records),
	)

	// get the chaos pod node IP from the environment variable
	chaosPodIP, ok := os.LookupEnv(env.InjectorChaosPodIP)
	if !ok {
		return fmt.Errorf("%s environment variable must be set with the chaos pod IP", env.InjectorChaosPodIP)
	}

	// Determine ports (use helper method for consistency)
	dnsPort := i.spec.GetPortWithDefault()

	// Determine protocol (use helper method for consistency)
	protocol := i.spec.GetProtocolWithDefault()

	// DNS responder runs on different ports to avoid conflicts
	// We'll redirect from dnsPort to responder ports via IPTables
	// When protocol is "both", UDP and TCP need separate ports
	const baseResponderPort = 5353
	udpPort := baseResponderPort
	tcpPort := baseResponderPort

	if protocol == "both" {
		tcpPort = baseResponderPort + 1 // TCP uses 5354
	}

	i.config.Log.Infow("DNS responder ports",
		tags.UDPPortKey, udpPort,
		tags.TCPPortKey, tcpPort,
		tags.ProtocolKey, protocol,
		tags.ChaosPodIPKey, chaosPodIP,
	)

	launchDNSServer.Do(func() {
		// Get upstream DNS from target container's resolv.conf
		targetPID := i.config.TargetContainer.PID()
		resolvConfPath := fmt.Sprintf("/proc/%d/root/etc/resolv.conf", targetPID)
		upstreamDNS := getUpstreamDNSFromResolvConf(resolvConfPath)

		i.config.Log.Infow("resolved upstream DNS server",
			tags.UpstreamDNSKey, upstreamDNS,
			tags.ResolvConfPathKey, resolvConfPath,
		)

		// Create and start DNS responder in the target's network namespace
		responderConfig := network.DNSResponderConfig{
			Records:     convertToResponderRecords(i.spec.Records),
			UDPPort:     udpPort,
			TCPPort:     tcpPort,
			Protocol:    protocol,
			UpstreamDNS: upstreamDNS,
			Logger:      i.config.Log,
		}

		// Use provided responder if available (for testing), otherwise create real one
		if i.config.Responder != nil {
			i.responder = i.config.Responder
		} else {
			i.responder = network.NewDNSResponder(responderConfig)
		}

		i.config.Log.Infow("starting DNS responder in target network namespace",
			tags.UDPPortKey, udpPort,
			tags.TCPPortKey, tcpPort,
			tags.ProtocolKey, protocol,
		)

		if err := i.responder.Start(); err != nil {
			launchDNSServerErr = fmt.Errorf("failed to start DNS responder: %w", err)

			return
		}
	})

	// This will return an error on all injectors if the launchDNSServer once call failed
	// if we did not succeed to create a fake DNS server, we should not continue
	if launchDNSServerErr != nil {
		return launchDNSServerErr
	}

	i.config.Log.Infow("entering target network namespace to configure DNS disruption",
		tags.DNSPortKey, dnsPort,
		tags.UDPPortKey, udpPort,
		tags.TCPPortKey, tcpPort,
		tags.ProtocolKey, protocol,
	)

	// Enter target container's network namespace
	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the target container network namespace: %w", err)
	}

	// Configure IPTables to intercept DNS traffic based on protocol
	// Setup IPTables for UDP if needed
	if protocol == "udp" || protocol == "both" {
		if err := i.setupIPTablesForProtocol("udp", dnsPort, udpPort, chaosPodIP); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()

			return err
		}
	}

	// Setup IPTables for TCP if needed
	if protocol == "tcp" || protocol == "both" {
		if err := i.setupIPTablesForProtocol("tcp", dnsPort, tcpPort, chaosPodIP); err != nil {
			i.stopResponder()
			_ = i.config.Netns.Exit()

			return err
		}
	}

	// Exit target container's network namespace
	if err := i.config.Netns.Exit(); err != nil {
		i.stopResponder()

		return fmt.Errorf("unable to exit the target container network namespace: %w", err)
	}

	i.config.Log.Infow("DNS disruption injected successfully",
		tags.DNSRecordCountKey, len(i.spec.Records),
	)

	// DNS responder is running in background goroutines
	// Return now so the pod can become ready
	// The responder will keep running until Clean() is called
	return nil
}

// setupIPTablesForProtocol configures IPTables rules for a specific protocol (udp or tcp)
// IMPORTANT: RedirectTo must be called before Intercept to create the CHAOS-DNS chain first
func (i *dnsDisruptionInjector) setupIPTablesForProtocol(proto string, dnsPort, responderPort int, chaosPodIP string) error {
	// Redirect to DNS responder (creates CHAOS-DNS chain)
	if err := i.config.IPTables.RedirectTo(proto, fmt.Sprintf("%d", dnsPort), fmt.Sprintf("%s:%d", chaosPodIP, responderPort)); err != nil {
		return fmt.Errorf("failed to configure IPTables for %s redirection: %w", proto, err)
	}

	// Intercept ALL DNS traffic in this namespace (no cgroup filtering since we're in target namespace)
	if err := i.config.IPTables.Intercept(proto, fmt.Sprintf("%d", dnsPort), "", "", ""); err != nil {
		return fmt.Errorf("failed to configure IPTables for %s interception: %w", proto, err)
	}

	return nil
}

// stopResponder stops the DNS responder and logs any errors
func (i *dnsDisruptionInjector) stopResponder() {
	if i.responder != nil {
		if err := i.responder.Stop(); err != nil {
			i.config.Log.Errorw("error stopping DNS responder", tags.ErrorKey, err)
		}
	}
}

func (i *dnsDisruptionInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean removes IPTables rules and stops the DNS responder
func (i *dnsDisruptionInjector) Clean() error {
	// Dry run mode - nothing to clean
	if i.config.Disruption.DryRun {
		i.config.Log.Infow("cleaning up DNS disruption in dry run mode (no-op)")
		return nil
	}

	i.config.Log.Infow("cleaning up DNS disruption")

	var errs []error

	// Stop DNS responder
	if i.responder != nil {
		i.config.Log.Infow("stopping DNS responder")

		if err := i.responder.Stop(); err != nil {
			i.config.Log.Errorw("error stopping DNS responder during cleanup", tags.ErrorKey, err)
			errs = append(errs, fmt.Errorf("failed to stop DNS responder: %w", err))
		} else {
			ResetDNSServerOnce()
		}

		i.responder = nil
	}

	// Enter target container's network namespace to clear IPTables rules
	i.config.Log.Infow("entering target network namespace to clear IPTables rules")

	if err := i.config.Netns.Enter(); err != nil {
		i.config.Log.Errorw("error entering network namespace during cleanup", tags.ErrorKey, err)
		errs = append(errs, fmt.Errorf("failed to enter network namespace: %w", err))
	} else {
		// Clear IPTables rules
		i.config.Log.Infow("clearing IPTables rules")

		if err := i.config.IPTables.Clear(); err != nil {
			i.config.Log.Errorw("error clearing IPTables rules", tags.ErrorKey, err)
			errs = append(errs, fmt.Errorf("failed to clear IPTables rules: %w", err))
		}

		// Exit target container's network namespace
		if err := i.config.Netns.Exit(); err != nil {
			i.config.Log.Errorw("error exiting network namespace during cleanup", tags.ErrorKey, err)
			errs = append(errs, fmt.Errorf("failed to exit network namespace: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during DNS disruption cleanup: %v", errs)
	}

	i.config.Log.Infow("DNS disruption cleaned up successfully")

	return nil
}

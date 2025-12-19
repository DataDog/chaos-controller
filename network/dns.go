// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package network

import (
	"fmt"
	"net"

	"github.com/avast/retry-go"
	"github.com/hashicorp/go-multierror"
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

const (
	// DNSResolverPod uses only pod DNS configuration
	DNSResolverPod = "pod"
	// DNSResolverNode uses only node DNS configuration
	DNSResolverNode = "node"
	// DNSResolverPodFallbackNode tries pod first, then falls back to node (default behavior)
	DNSResolverPodFallbackNode = "pod-fallback-node"
	// DNSResolverNodeFallbackPod tries node first, then falls back to pod
	DNSResolverNodeFallbackPod = "node-fallback-pod"

	DefaultPodResolvConfPath  = "/etc/resolv.conf"
	DefaultNodeResolvConfPath = "/mnt/host/etc/resolv.conf"
)

type DNSConfig struct {
	DNSServer string
	KubeDNS   string
}

// DNSClient is a client being able to resolve the given host
type DNSClient interface {
	ResolveWithStrategy(host string, strategy string) ([]net.IP, error)
}

type dnsClient struct {
	podResolvConfPath  string
	nodeResolvConfPath string
	log                *zap.SugaredLogger
}

// DNSClientConfig contains configuration for the DNS client
type DNSClientConfig struct {
	PodResolvConfPath  string
	NodeResolvConfPath string
	Logger             *zap.SugaredLogger
}

// NewDNSClient creates a standard DNS client with optional configuration
func NewDNSClient(config DNSClientConfig) DNSClient {
	client := dnsClient{
		podResolvConfPath:  DefaultPodResolvConfPath,
		nodeResolvConfPath: DefaultNodeResolvConfPath,
	}

	// Apply configuration if provided
	if config.PodResolvConfPath != "" {
		client.podResolvConfPath = config.PodResolvConfPath
	}

	if config.NodeResolvConfPath != "" {
		client.nodeResolvConfPath = config.NodeResolvConfPath
	}

	client.log = config.Logger

	return client
}

func (c dnsClient) ResolveWithStrategy(host string, strategy string) ([]net.IP, error) {
	resolvers, names, err := c.getResolversAndNames(host, strategy)
	if err != nil {
		return nil, err
	}

	// do the request on the first configured dns resolver
	response := &dns.Msg{}

	var ips []net.IP

	err = retry.Do(func() error {
		// query possible resolvers and fqdn based on servers and search domains specified in the dns configuration
		multiErr := &multierror.Error{}

		for _, name := range names {
			// try to resolve the given host as an A record
			response, err = c.resolve(name, "udp", resolvers)
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
			}

			if response == nil {
				continue
			}

			// if the response is truncated, retry with TCP
			if response.Truncated {
				response, err = c.resolve(name, "tcp", resolvers)
				if err != nil {
					multiErr = multierror.Append(multiErr, err)
				}
			}

			if response != nil && len(response.Answer) > 0 {
				return nil
			}
		}

		return multiErr
	}, retry.Attempts(3))
	if err != nil {
		return nil, fmt.Errorf("can't resolve the given hostname %s: %w", host, err)
	}

	// parse returned records
	for _, answer := range response.Answer {
		if ip, ok := answer.(*dns.A); ok {
			ips = append(ips, ip.A)
		}
	}

	// error if no A records can be found
	if len(ips) == 0 {
		return nil, fmt.Errorf("no A records were found for the given hostname %s", host)
	}

	return ips, nil
}

// getResolversAndNames reads DNS configs based on strategy and returns resolvers and names lists
func (c dnsClient) getResolversAndNames(host string, strategy string) (resolvers, names []string, err error) {
	// Default to pod-fallback-node if strategy is empty
	if strategy == "" {
		strategy = DNSResolverPodFallbackNode
	}

	// Use configured paths from the client
	podPath := c.podResolvConfPath
	nodePath := c.nodeResolvConfPath

	switch strategy {
	case DNSResolverPod:
		podDNSConfig, err := dns.ClientConfigFromFile(podPath)
		if err != nil {
			return nil, nil, fmt.Errorf("can't read pod resolv.conf file: %w", err)
		}

		c.log.Infow("loaded pod DNS configuration", "resolv_conf_path", podPath, "nameservers", podDNSConfig.Servers)

		resolvers = podDNSConfig.Servers
		names = podDNSConfig.NameList(host)
	case DNSResolverNode:
		nodeDNSConfig, err := dns.ClientConfigFromFile(nodePath)
		if err != nil {
			return nil, nil, fmt.Errorf("can't read node resolv.conf file: %w", err)
		}

		c.log.Infow("loaded node DNS configuration", "resolv_conf_path", nodePath, "nameservers", nodeDNSConfig.Servers)

		resolvers = nodeDNSConfig.Servers
		names = nodeDNSConfig.NameList(host)
	case DNSResolverPodFallbackNode, DNSResolverNodeFallbackPod:
		podDNSConfig, err := dns.ClientConfigFromFile(podPath)
		if err != nil {
			return nil, nil, fmt.Errorf("can't read pod resolv.conf file: %w", err)
		}

		nodeDNSConfig, err := dns.ClientConfigFromFile(nodePath)
		if err != nil {
			return nil, nil, fmt.Errorf("can't read node resolv.conf file: %w", err)
		}

		if strategy == DNSResolverPodFallbackNode {
			// Pod first, then node
			resolvers = append([]string{}, podDNSConfig.Servers...)
			resolvers = append(resolvers, nodeDNSConfig.Servers...)
			names = append([]string{}, podDNSConfig.NameList(host)...)
			names = append(names, nodeDNSConfig.NameList(host)...)

			c.log.Infow("loaded pod and node DNS configuration (pod-fallback-node strategy)",
				"pod_resolv_conf_path", podPath,
				"pod_nameservers", podDNSConfig.Servers,
				"node_resolv_conf_path", nodePath,
				"node_nameservers", nodeDNSConfig.Servers)
		} else {
			// Node first, then pod
			resolvers = append([]string{}, nodeDNSConfig.Servers...)
			resolvers = append(resolvers, podDNSConfig.Servers...)
			names = append([]string{}, nodeDNSConfig.NameList(host)...)
			names = append(names, podDNSConfig.NameList(host)...)

			c.log.Infow("loaded pod and node DNS configuration (node-fallback-pod strategy)",
				"node_resolv_conf_path", nodePath,
				"node_nameservers", nodeDNSConfig.Servers,
				"pod_resolv_conf_path", podPath,
				"pod_nameservers", podDNSConfig.Servers)
		}
	default:
		return nil, nil, fmt.Errorf("unknown DNS resolver strategy: %s", strategy)
	}

	return resolvers, names, nil
}

func (c dnsClient) resolve(hostName string, protocol string, resolvers []string) (response *dns.Msg, multiErr error) {
	client := dns.Client{}

	dnsMessage := dns.Msg{}
	dnsMessage.SetQuestion(hostName, dns.TypeA)

	switch protocol {
	case "tcp":
		client.Net = "tcp"
	case "udp":
		client.Net = "udp"
		// Increase EDNS buffer size to reduce truncation
		// Refer to RFC 5966 https://www.rfc-editor.org/rfc/rfc5966
		dnsMessage.SetEdns0(4096, true)
	default:
		return nil, fmt.Errorf("unknown protocol %s. Supported protocols are tcp and udp", protocol)
	}

	for _, server := range resolvers {
		var err error
		response, _, err = client.Exchange(&dnsMessage, fmt.Sprintf("%s:53", server))

		if response != nil && len(response.Answer) > 0 {
			break
		}

		multiErr = multierror.Append(multiErr, err)
	}

	return response, multiErr
}

// Test helpers for unexported functions

// ReadResolvConfFileForTest is a test helper that exposes readResolvConfFile for unit testing
func ReadResolvConfFileForTest(path string) (*dns.ClientConfig, error) {
	config, err := dns.ClientConfigFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read resolv.conf from %s: %w", path, err)
	}

	return config, nil
}

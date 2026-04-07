// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"net"

	v1 "k8s.io/api/core/v1"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/bpfdisrupt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Tests run via TestInjector in network_disruption_test.go (single RunSpecs per package)

var _ = Describe("specToRules", func() {
	var (
		spec         v1beta1.NetworkDisruptionSpec
		hosts        []v1beta1.NetworkDisruptionHostSpec
		resolvedIPs  map[string][]*net.IPNet
		safeguardIPs []*net.IPNet
	)

	BeforeEach(func() {
		spec = v1beta1.NetworkDisruptionSpec{
			Drop:  50,
			Delay: 100,
		}
		hosts = []v1beta1.NetworkDisruptionHostSpec{}
		resolvedIPs = map[string][]*net.IPNet{}
		safeguardIPs = []*net.IPNet{}
	})

	Context("with no hosts specified", func() {
		It("should create a match-all DISRUPT rule", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			Expect(rules).To(HaveLen(1))
			Expect(rules[0].CIDR).To(Equal("0.0.0.0/0"))
			Expect(rules[0].Action).To(Equal(bpfdisrupt.ActionDisrupt))
		})
	})

	Context("with safeguard IPs", func() {
		BeforeEach(func() {
			_, gw, _ := net.ParseCIDR("192.168.0.1/32")
			safeguardIPs = []*net.IPNet{gw}
		})

		It("should create ALLOW rules for safeguard IPs", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			Expect(rules[0].Action).To(Equal(bpfdisrupt.ActionAllow))
			Expect(rules[0].CIDR).To(Equal("192.168.0.1/32"))
		})
	})

	Context("with allowed hosts", func() {
		BeforeEach(func() {
			spec.AllowedHosts = []v1beta1.NetworkDisruptionHostSpec{
				{Host: "dns.google"},
			}
			_, ip, _ := net.ParseCIDR("8.8.8.8/32")
			resolvedIPs["dns.google"] = []*net.IPNet{ip}
		})

		It("should create ALLOW rules for allowed hosts", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			found := false

			for _, r := range rules {
				if r.CIDR == "8.8.8.8/32" && r.Action == bpfdisrupt.ActionAllow {
					found = true
				}
			}

			Expect(found).To(BeTrue())
		})
	})

	Context("with egress hosts", func() {
		BeforeEach(func() {
			hosts = []v1beta1.NetworkDisruptionHostSpec{
				{Host: "example.com", Port: 80, Protocol: "tcp", Flow: v1beta1.FlowEgress},
			}
			_, ip, _ := net.ParseCIDR("93.184.216.34/32")
			resolvedIPs["example.com"] = []*net.IPNet{ip}
		})

		It("should create DISRUPT rules with port and protocol", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			var hostRule *bpfdisrupt.Rule

			for i := range rules {
				if rules[i].CIDR == "93.184.216.34/32" {
					hostRule = &rules[i]
				}
			}

			Expect(hostRule).ToNot(BeNil())
			Expect(hostRule.Direction).To(Equal(bpfdisrupt.DirEgress))
			Expect(hostRule.Action).To(Equal(bpfdisrupt.ActionDisrupt))
			Expect(hostRule.Port).To(Equal(80))
			Expect(hostRule.Protocol).To(Equal("tcp"))
		})
	})

	Context("with ingress hosts (drop-only)", func() {
		BeforeEach(func() {
			spec = v1beta1.NetworkDisruptionSpec{Drop: 50} // drop only, no delay
			hosts = []v1beta1.NetworkDisruptionHostSpec{
				{Host: "attacker.com", Flow: v1beta1.FlowIngress},
			}
			_, ip, _ := net.ParseCIDR("10.0.0.5/32")
			resolvedIPs["attacker.com"] = []*net.IPNet{ip}
		})

		It("should create DROP rules with drop percentage", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			var hostRule *bpfdisrupt.Rule

			for i := range rules {
				if rules[i].CIDR == "10.0.0.5/32" {
					hostRule = &rules[i]
				}
			}

			Expect(hostRule).ToNot(BeNil())
			Expect(hostRule.Direction).To(Equal(bpfdisrupt.DirIngress))
			Expect(hostRule.Action).To(Equal(bpfdisrupt.ActionDrop))
			Expect(hostRule.DropPct).To(Equal(50))
		})
	})

	Context("with ingress hosts (shaping)", func() {
		BeforeEach(func() {
			spec = v1beta1.NetworkDisruptionSpec{Drop: 50, Delay: 100} // drop + delay = shaping
			hosts = []v1beta1.NetworkDisruptionHostSpec{
				{Host: "client.com", Flow: v1beta1.FlowIngress},
			}
			_, ip, _ := net.ParseCIDR("10.0.0.5/32")
			resolvedIPs["client.com"] = []*net.IPNet{ip}
		})

		It("should create DISRUPT rules (not DROP) when shaping is needed", func() {
			rules := specToRules(spec, hosts, resolvedIPs, safeguardIPs)
			var hostRule *bpfdisrupt.Rule

			for i := range rules {
				if rules[i].CIDR == "10.0.0.5/32" {
					hostRule = &rules[i]
				}
			}

			Expect(hostRule).ToNot(BeNil())
			Expect(hostRule.Direction).To(Equal(bpfdisrupt.DirIngress))
			Expect(hostRule.Action).To(Equal(bpfdisrupt.ActionDisrupt))
		})
	})
})

var _ = Describe("serviceEndpointsToRules", func() {
	It("should convert service filters to BPF rules", func() {
		_, ip, _ := net.ParseCIDR("10.0.0.1/32")
		filters := []tcServiceFilter{
			{
				service: networkDisruptionService{
					ip:       ip,
					port:     8080,
					protocol: v1.ProtocolTCP,
				},
			},
		}

		rules := serviceEndpointsToRules(filters)
		Expect(rules).To(HaveLen(1))
		Expect(rules[0].CIDR).To(Equal("10.0.0.1/32"))
		Expect(rules[0].Port).To(Equal(8080))
		Expect(rules[0].Protocol).To(Equal("tcp"))
		Expect(rules[0].Action).To(Equal(bpfdisrupt.ActionDisrupt))
		Expect(rules[0].Direction).To(Equal(bpfdisrupt.DirEgress))
	})

	It("should handle UDP protocol", func() {
		_, ip, _ := net.ParseCIDR("10.0.0.2/32")
		filters := []tcServiceFilter{
			{service: networkDisruptionService{ip: ip, port: 53, protocol: v1.ProtocolUDP}},
		}

		rules := serviceEndpointsToRules(filters)
		Expect(rules[0].Protocol).To(Equal("udp"))
	})

	It("should skip entries with nil IP", func() {
		filters := []tcServiceFilter{
			{service: networkDisruptionService{ip: nil, port: 80, protocol: v1.ProtocolTCP}},
		}

		rules := serviceEndpointsToRules(filters)
		Expect(rules).To(BeEmpty())
	})
})

var _ = Describe("hasIngressShaping", func() {
	It("should return false when no ingress hosts exist", func() {
		spec := v1beta1.NetworkDisruptionSpec{
			Delay: 100,
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{Host: "example.com", Flow: v1beta1.FlowEgress},
			},
		}
		Expect(hasIngressShaping(spec)).To(BeFalse())
	})

	It("should return false for ingress drop-only", func() {
		spec := v1beta1.NetworkDisruptionSpec{
			Drop: 50,
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{Host: "example.com", Flow: v1beta1.FlowIngress},
			},
		}
		Expect(hasIngressShaping(spec)).To(BeFalse())
	})

	It("should return true for ingress with delay", func() {
		spec := v1beta1.NetworkDisruptionSpec{
			Drop:  50,
			Delay: 100,
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{Host: "example.com", Flow: v1beta1.FlowIngress},
			},
		}
		Expect(hasIngressShaping(spec)).To(BeTrue())
	})

	It("should return true for ingress with bandwidth limit", func() {
		spec := v1beta1.NetworkDisruptionSpec{
			BandwidthLimit: 1000,
			Hosts: []v1beta1.NetworkDisruptionHostSpec{
				{Host: "example.com", Flow: v1beta1.FlowIngress},
			},
		}
		Expect(hasIngressShaping(spec)).To(BeTrue())
	})
})

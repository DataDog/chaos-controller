// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"net"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/network"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("Network ingress peer address", func() {
	DescribeTable("matches the peer as the destination of the outgoing response", func(host, cidr, flow string, sourcePort, destinationPort int) {
		spec := v1beta1.NetworkDisruptionSpec{Drop: 100, Hosts: []v1beta1.NetworkDisruptionHostSpec{{
			Host: host, Port: 80, Protocol: "tcp", Flow: flow,
		}}}
		Expect(spec.Validate()).To(Succeed())
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(1001, nil).Once()
		inj := networkDisruptionInjector{spec: spec, config: NetworkDisruptionInjectorConfig{
			Config: Config{Log: zap.NewNop().Sugar(), InjectorCtx: context.Background()}, TrafficController: traffic,
		}}
		_, err := inj.addFiltersForHosts([]string{"eth0"}, spec.Hosts, "1:4")
		Expect(err).NotTo(HaveOccurred())
		_, peer, err := net.ParseCIDR(cidr)
		Expect(err).NotTo(HaveOccurred())
		var noSourceIP *net.IPNet
		traffic.AssertCalled(GinkgoT(), "AddFilter", []string{"eth0"}, "1:0", "", noSourceIP, peer, sourcePort, destinationPort, network.TCP, network.ConnStateUndefined, "1:4")
	},
		Entry("ingress from one client", "192.0.2.10", "192.0.2.10/32", v1beta1.FlowIngress, 80, 0),
		Entry("ingress from a client subnet", "192.0.2.0/24", "192.0.2.0/24", v1beta1.FlowIngress, 80, 0),
		Entry("egress remains destination-filtered", "192.0.2.10", "192.0.2.10/32", v1beta1.FlowEgress, 0, 80),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"errors"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("Network cleanup namespace balance", func() {
	DescribeTable("exits the target namespace on cleanup errors", func(qdiscErr, iptablesErr, exitErr error) {
		ns := netns.NewManagerMock(GinkgoT())
		ns.EXPECT().Enter().Return(nil).Once()
		ns.EXPECT().Exit().Return(exitErr).Maybe()
		link := network.NewNetlinkLinkMock(GinkgoT())
		link.EXPECT().Name().Return("eth0").Once()
		nl := network.NewNetlinkAdapterMock(GinkgoT())
		nl.EXPECT().LinkList(true, mock.Anything).Return([]network.NetlinkLink{link}, nil).Once()
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.EXPECT().ClearQdisc([]string{"eth0"}).Return(qdiscErr).Once()
		iptables := network.NewIPTablesMock(GinkgoT())
		iptables.EXPECT().Clear().Return(iptablesErr).Maybe()
		inj := networkDisruptionInjector{config: NetworkDisruptionInjectorConfig{
			Config: Config{Log: zap.NewNop().Sugar(), Netns: ns,
				Disruption: chaosapi.DisruptionArgs{Level: chaostypes.DisruptionLevelPod}},
			TrafficController: traffic,
			IPTables:          iptables,
			NetlinkAdapter:    nl,
		}}
		err := inj.Clean()
		Expect(err).To(HaveOccurred())
		ns.AssertNumberOfCalls(GinkgoT(), "Exit", 1)
		for _, cause := range []error{qdiscErr, iptablesErr, exitErr} {
			if cause != nil {
				Expect(errors.Is(err, cause)).To(BeTrue(), "cleanup lost error: %v", cause)
			}
		}
	},
		Entry("qdisc removal fails", errors.New("qdisc failure"), nil, nil),
		Entry("iptables removal fails", nil, errors.New("iptables failure"), nil),
		Entry("qdisc removal and namespace exit both fail", errors.New("qdisc failure"), nil, errors.New("exit failure")),
	)
})

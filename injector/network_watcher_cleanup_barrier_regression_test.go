// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("Network watcher cleanup barrier", func() {
	It("waits for an in-flight host update before clearing qdiscs", func() {
		resolving := make(chan struct{})
		release := make(chan struct{})
		cleared := make(chan struct{})
		var resolvingOnce, releaseOnce sync.Once
		var exits int32
		dns := network.NewDNSClientMock(GinkgoT())
		dns.EXPECT().ResolveWithStrategy("peer.example", "").Return([]net.IP{net.ParseIP("192.0.2.1")}, nil).Once()
		dns.EXPECT().ResolveWithStrategy("peer.example", "").Run(func(string, string) {
			resolvingOnce.Do(func() { close(resolving) })
			<-release
		}).Return([]net.IP{net.ParseIP("192.0.2.2")}, nil)
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(1001, nil)
		traffic.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil).Maybe()
		traffic.EXPECT().ClearQdisc([]string{"eth0"}).Run(func([]string) { close(cleared) }).Return(nil).Once()
		ns := netns.NewManagerMock(GinkgoT())
		ns.EXPECT().Enter().Return(nil)
		ns.EXPECT().Exit().Run(func() { atomic.AddInt32(&exits, 1) }).Return(nil)
		link := network.NewNetlinkLinkMock(GinkgoT())
		link.EXPECT().Name().Return("eth0")
		nl := network.NewNetlinkAdapterMock(GinkgoT())
		nl.EXPECT().LinkList(true, mock.Anything).Return([]network.NetlinkLink{link}, nil).Once()
		iptables := network.NewIPTablesMock(GinkgoT())
		iptables.EXPECT().Clear().Return(nil).Once()
		cg := cgroup.NewManagerMock(GinkgoT())
		cg.EXPECT().IsCgroupV2().Return(true).Once()
		inj := networkDisruptionInjector{
			spec: v1beta1.NetworkDisruptionSpec{Drop: 100, Hosts: []v1beta1.NetworkDisruptionHostSpec{{Host: "peer.example", Protocol: "tcp"}}},
			config: NetworkDisruptionInjectorConfig{
				Config: Config{Log: zap.NewNop().Sugar(), InjectorCtx: context.Background(), Netns: ns, Cgroup: cg,
					Disruption: chaosapi.DisruptionArgs{Level: chaostypes.DisruptionLevelPod}},
				DNSClient: dns, TrafficController: traffic, NetlinkAdapter: nl, IPTables: iptables, HostResolveInterval: 30 * time.Millisecond,
			},
		}
		Expect(inj.handleFiltersForHosts([]string{"eth0"}, "1:4")).To(Succeed())
		cancel := inj.hostWatcherCancel
		cancelCalled := make(chan struct{})
		inj.hostWatcherCancel = func() {
			cancel()
			close(cancelCalled)
		}
		Eventually(resolving, 5*time.Second).Should(BeClosed())
		cleanResult := make(chan error, 1)
		cleanFinished := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(cleanFinished)
			cleanResult <- inj.Clean()
		}()
		DeferCleanup(func() {
			releaseOnce.Do(func() { close(release) })
			cancel()
			Eventually(cleanFinished, 5*time.Second).Should(BeClosed())
			Eventually(func() int32 { return atomic.LoadInt32(&exits) }, 5*time.Second).Should(BeNumerically(">=", 2))
		})
		Eventually(cancelCalled, 5*time.Second).Should(BeClosed())
		Consistently(cleared, 200*time.Millisecond).ShouldNot(BeClosed())
		releaseOnce.Do(func() { close(release) })
		Eventually(cleanResult, 5*time.Second).Should(Receive(BeNil()))
		Expect(cleared).To(BeClosed())
	})
})

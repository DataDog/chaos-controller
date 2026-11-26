// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-controller/network"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Headless service filter cleanup", func() {
	It("removes both protocol priorities on every interface", func() {
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.EXPECT().AddFilter(mock.Anything, "1:0", "", mock.Anything, mock.Anything, 0, 0, network.TCP, network.ConnStateUndefined, "1:4").Return(1001, nil).Once()
		traffic.EXPECT().AddFilter(mock.Anything, "1:0", "", mock.Anything, mock.Anything, 0, 0, network.UDP, network.ConnStateUndefined, "1:4").Return(1002, nil).Once()
		traffic.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil)
		inj := networkDisruptionInjector{config: NetworkDisruptionInjectorConfig{
			Config:            Config{Log: zap.NewNop().Sugar()},
			TrafficController: traffic,
		}}
		interfaces := []string{"eth0", "lo"}
		pod := v1.Pod{Status: v1.PodStatus{PodIP: "10.0.0.1"}}
		filters := inj.buildServiceFiltersFromPod(pod, []v1.ServicePort{{Port: 0}})
		installed, err := inj.addServiceFilters("headless", filters, interfaces, "1:4")
		Expect(err).NotTo(HaveOccurred())
		Expect(installed).To(HaveLen(1))
		Expect(inj.removeServiceFilter(interfaces, installed[0])).To(Succeed())
		for _, iface := range interfaces {
			traffic.AssertCalled(GinkgoT(), "DeleteFilter", iface, uint32(1001))
			traffic.AssertCalled(GinkgoT(), "DeleteFilter", iface, uint32(1002))
		}
		traffic.AssertNumberOfCalls(GinkgoT(), "DeleteFilter", 4)
	})
})

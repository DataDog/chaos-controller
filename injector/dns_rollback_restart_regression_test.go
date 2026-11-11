// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"errors"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

type rollbackDNSStartCounter struct {
	network.DNSResponder
	starts int
}

func (r *rollbackDNSStartCounter) Start() error {
	r.starts++
	return r.DNSResponder.Start()
}

var _ = Describe("DNS restart after redirect rollback", func() {
	DescribeTable("restarts the real responder", func(cleanBeforeRetry bool) {
		injector.ResetDNSServerOnce()
		DeferCleanup(injector.ResetDNSServerOnce)
		GinkgoT().Setenv("CHAOS_POD_IP", "10.244.0.10")
		logger := zap.NewNop().Sugar()
		responder := &rollbackDNSStartCounter{DNSResponder: network.NewDNSResponder(network.DNSResponderConfig{
			UDPPort: 0, Protocol: "udp", Logger: logger,
		})}
		iptables := network.NewIPTablesMock(GinkgoT())
		iptables.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(errors.New("redirect unavailable")).Once()
		iptables.EXPECT().RedirectTo("udp", "53", "10.244.0.10:5353").Return(nil).Once()
		iptables.EXPECT().Intercept("udp", "53", "", "", "").Return(nil).Once()
		iptables.EXPECT().Clear().Return(nil)
		ns := netns.NewManagerMock(GinkgoT())
		ns.EXPECT().Enter().Return(nil)
		ns.EXPECT().Exit().Return(nil)
		ctn := container.NewContainerMock(GinkgoT())
		ctn.EXPECT().Name().Return("target")
		ctn.EXPECT().PID().Return(1234)
		spec := v1beta1.DNSDisruptionSpec{Protocol: "udp", Records: []v1beta1.DNSRecord{{
			Hostname: "example.com", Record: v1beta1.DNSRecordConfig{Type: "A", Value: "192.0.2.1"},
		}}}
		inj := injector.NewDNSDisruptionInjector(spec, injector.DNSDisruptionInjectorConfig{
			Config: injector.Config{Log: logger, TargetContainer: ctn, Netns: ns,
				Disruption: chaosapi.DisruptionArgs{Level: chaostypes.DisruptionLevelPod}},
			IPTables: iptables, Responder: responder,
		})
		DeferCleanup(func() {
			_ = inj.Clean()
			_ = responder.Stop()
		})
		Expect(inj.Inject()).To(MatchError(ContainSubstring("redirect unavailable")))
		if cleanBeforeRetry {
			// Match the CLI's retry behavior if cleanup is requested after failure.
			if err := inj.Clean(); err != nil {
				Expect(inj.Clean()).To(Succeed())
			}
		}
		Expect(inj.Inject()).To(Succeed())
		Expect(responder.starts).To(Equal(2))
	},
		Entry("on the next pulse after failed injection", false),
		Entry("after cleanup and its retry", true),
	)
})

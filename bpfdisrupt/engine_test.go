// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package bpfdisrupt_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-controller/bpfdisrupt"
	"github.com/DataDog/chaos-controller/network"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestBPFDisrupt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BPF Disruption Engine Suite")
}

// mockCmdRunner implements bpfdisrupt.CmdRunner for testing
type mockCmdRunner struct {
	mock.Mock
}

func (m *mockCmdRunner) Run(args []string) (int, string, error) {
	ret := m.Called(args)
	return ret.Int(0), ret.String(1), ret.Error(2)
}

var _ = Describe("Engine", func() {
	var (
		tc        *network.TrafficControllerMock
		nl        *network.NetlinkAdapterMock
		cmdRunner *mockCmdRunner
		log       *zap.SugaredLogger
		engine    *bpfdisrupt.Engine
		nlLink    *network.NetlinkLinkMock
	)

	BeforeEach(func() {
		tc = network.NewTrafficControllerMock(GinkgoT())
		nl = network.NewNetlinkAdapterMock(GinkgoT())
		cmdRunner = &mockCmdRunner{}
		log = zaptest.NewLogger(GinkgoT()).Sugar()
		engine = bpfdisrupt.NewEngine(tc, nl, cmdRunner, log)
		nlLink = network.NewNetlinkLinkMock(GinkgoT())
		nlLink.EXPECT().Name().Return("ifb-abcd1234").Maybe()
		nlLink.EXPECT().Index().Return(42).Maybe()
	})

	Describe("Attach", func() {
		Context("with egress-only rules (no IFB needed)", func() {
			BeforeEach(func() {
				tc.EXPECT().AddClsact([]string{"eth0"}).Return(nil)
				tc.EXPECT().AddBPFFilter([]string{"eth0"}, bpfdisrupt.EgressParent, bpfdisrupt.BPFObjectPath, bpfdisrupt.EgressFlowID, bpfdisrupt.EgressSection).Return(nil)
				tc.EXPECT().AddIngressBPFFilter([]string{"eth0"}, bpfdisrupt.BPFObjectPath, bpfdisrupt.IngressSection).Return(nil)
				cmdRunner.On("Run", mock.MatchedBy(func(args []string) bool {
					return len(args) >= 5 && args[2] == "10.0.0.1/32"
				})).Return(0, "", nil)
			})

			It("should attach BPF programs without creating IFB", func() {
				rules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirEgress, CIDR: "10.0.0.1/32", Action: bpfdisrupt.ActionDisrupt},
				}
				err := engine.Attach([]string{"eth0"}, rules, "abcd1234-efgh", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.Attached()).To(BeTrue())
				Expect(engine.IFBName()).To(BeEmpty())
			})
		})

		Context("with ingress shaping rules (IFB needed)", func() {
			BeforeEach(func() {
				nl.EXPECT().AddIFBDevice("ifb-abcd1234").Return(nlLink, nil)
				tc.EXPECT().AddClsact([]string{"eth0"}).Return(nil)
				tc.EXPECT().AddBPFFilter([]string{"eth0"}, bpfdisrupt.EgressParent, bpfdisrupt.BPFObjectPath, bpfdisrupt.EgressFlowID, bpfdisrupt.EgressSection).Return(nil)
				tc.EXPECT().AddIngressBPFFilter([]string{"eth0"}, bpfdisrupt.BPFObjectPath, bpfdisrupt.IngressSection).Return(nil)
				cmdRunner.On("Run", mock.Anything).Return(0, "", nil)
			})

			It("should create IFB device and attach BPF programs", func() {
				rules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirIngress, CIDR: "10.0.0.1/32", Action: bpfdisrupt.ActionDisrupt},
				}
				err := engine.Attach([]string{"eth0"}, rules, "abcd1234-efgh", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.IFBName()).To(Equal("ifb-abcd1234"))
			})
		})

		Context("with drop-only ingress rules (no IFB)", func() {
			BeforeEach(func() {
				tc.EXPECT().AddClsact([]string{"eth0"}).Return(nil)
				tc.EXPECT().AddBPFFilter([]string{"eth0"}, bpfdisrupt.EgressParent, bpfdisrupt.BPFObjectPath, bpfdisrupt.EgressFlowID, bpfdisrupt.EgressSection).Return(nil)
				tc.EXPECT().AddIngressBPFFilter([]string{"eth0"}, bpfdisrupt.BPFObjectPath, bpfdisrupt.IngressSection).Return(nil)
				cmdRunner.On("Run", mock.MatchedBy(func(args []string) bool {
					return len(args) >= 5 && args[4] == "drop"
				})).Return(0, "", nil)
			})

			It("should not create IFB for drop-only rules", func() {
				rules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirIngress, CIDR: "10.0.0.1/32", Action: bpfdisrupt.ActionDrop, DropPct: 50},
				}
				err := engine.Attach([]string{"eth0"}, rules, "abcd1234-efgh", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.IFBName()).To(BeEmpty())
			})
		})
	})

	Describe("Detach", func() {
		Context("when attached with IFB", func() {
			BeforeEach(func() {
				nl.EXPECT().AddIFBDevice("ifb-abcd1234").Return(nlLink, nil)
				tc.EXPECT().AddClsact([]string{"eth0"}).Return(nil)
				tc.EXPECT().AddBPFFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				tc.EXPECT().AddIngressBPFFilter(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				cmdRunner.On("Run", mock.Anything).Return(0, "", nil)

				rules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirIngress, CIDR: "10.0.0.1/32", Action: bpfdisrupt.ActionDisrupt},
				}
				err := engine.Attach([]string{"eth0"}, rules, "abcd1234-efgh", true)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should remove clsact and IFB device", func() {
				tc.EXPECT().ClearIngressQdisc([]string{"eth0"}).Return(nil)
				nl.EXPECT().DeleteIFBDevice("ifb-abcd1234").Return(nil)

				err := engine.Detach()
				Expect(err).ToNot(HaveOccurred())
				Expect(engine.Attached()).To(BeFalse())
				Expect(engine.IFBName()).To(BeEmpty())
			})
		})

		Context("when not attached", func() {
			It("should return nil without doing anything", func() {
				err := engine.Detach()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("UpdateRules", func() {
		Context("when attached", func() {
			BeforeEach(func() {
				tc.EXPECT().AddClsact([]string{"eth0"}).Return(nil)
				tc.EXPECT().AddBPFFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				tc.EXPECT().AddIngressBPFFilter(mock.Anything, mock.Anything, mock.Anything).Return(nil)
				cmdRunner.On("Run", mock.Anything).Return(0, "", nil)

				rules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirEgress, CIDR: "10.0.0.1/32", Action: bpfdisrupt.ActionDisrupt},
				}
				err := engine.Attach([]string{"eth0"}, rules, "abcd1234", false)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should clear and re-populate rules", func() {
				newRules := []bpfdisrupt.Rule{
					{Direction: bpfdisrupt.DirEgress, CIDR: "10.0.0.2/32", Action: bpfdisrupt.ActionDisrupt},
				}
				err := engine.UpdateRules(newRules)
				Expect(err).ToNot(HaveOccurred())

				// Verify clear was called (--clear flag)
				clearCalled := false
				for _, call := range cmdRunner.Calls {
					args := call.Arguments.Get(0).([]string)
					for _, arg := range args {
						if arg == "--clear" {
							clearCalled = true
						}
					}
				}
				Expect(clearCalled).To(BeTrue())
			})
		})

		Context("when not attached", func() {
			It("should return an error", func() {
				err := engine.UpdateRules([]bpfdisrupt.Rule{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not attached"))
			})
		})
	})
})

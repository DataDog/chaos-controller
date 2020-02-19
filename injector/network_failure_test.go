// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"reflect"
	"time"

	"bou.ke/monkey"
	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/datadog"
	. "github.com/DataDog/chaos-fi-controller/injector"
	"github.com/DataDog/datadog-go/statsd"
)

type fakeIPTables struct {
	mock.Mock
}

func (f *fakeIPTables) Exists(table, chain string, parts ...string) (bool, error) {
	args := f.Called(table, chain, parts)
	return args.Bool(0), args.Error(1)
}

func (f *fakeIPTables) Delete(table, chain string, parts ...string) error {
	args := f.Called(table, chain, parts)
	return args.Error(0)
}

func (f *fakeIPTables) ClearChain(table, chain string) error {
	args := f.Called(table, chain)
	return args.Error(0)
}

func (f *fakeIPTables) DeleteChain(table, chain string) error {
	args := f.Called(table, chain)
	return args.Error(0)
}

func (f *fakeIPTables) ListChains(table string) ([]string, error) {
	args := f.Called(table)
	return args.Get(0).([]string), args.Error(1)
}

func (f *fakeIPTables) AppendUnique(table, chain string, parts ...string) error {
	args := f.Called(table, chain, parts)
	return args.Error(0)
}

func (f *fakeIPTables) NewChain(table, chain string) error {
	args := f.Called(table, chain)
	return args.Error(0)
}

var _ = Describe("Network Failure", func() {
	var (
		config            NetworkFailureInjectorConfig
		ctn               fakeContainer
		inj               Injector
		ipt               fakeIPTables
		iptExistsCall     *mock.Call
		iptListChainsCall *mock.Call
		spec              v1beta1.NetworkFailureSpec
		uid               string
	)

	BeforeEach(func() {
		// tests vars
		ctn = fakeContainer{}
		ctn.On("EnterNetworkNamespace").Return(nil)
		ctn.On("ExitNetworkNamespace").Return(nil)

		ipt = fakeIPTables{}
		iptExistsCall = ipt.On("Exists", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		ipt.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		ipt.On("ClearChain", mock.Anything, mock.Anything).Return(nil)
		ipt.On("DeleteChain", mock.Anything, mock.Anything).Return(nil)
		iptListChainsCall = ipt.On("ListChains", mock.Anything).Return([]string{}, nil)
		ipt.On("AppendUnique", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		ipt.On("NewChain", mock.Anything, mock.Anything).Return(nil)

		uid = "110e8400-e29b-11d4-a716-446655440000"

		spec = v1beta1.NetworkFailureSpec{
			Hosts:       []string{"127.0.0.1/32"},
			Port:        666,
			Protocol:    "tcp",
			Probability: 100,
		}

		config = NetworkFailureInjectorConfig{
			IPTables: &ipt,
		}

		// dns
		var dnsClient *dns.Client
		monkey.Patch(dns.ClientConfigFromFile, func(string) (*dns.ClientConfig, error) {
			return &dns.ClientConfig{
				Servers: []string{"127.0.0.1"},
			}, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(dnsClient), "Exchange", func(c *dns.Client, m *dns.Msg, address string) (*dns.Msg, time.Duration, error) {
			return &dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						A: net.IP{byte(192), byte(168), byte(0), byte(1)},
					},
					&dns.A{
						A: net.IP{byte(192), byte(168), byte(0), byte(2)},
					},
				},
			}, time.Second, nil
		})

		// datadog
		monkey.Patch(datadog.GetInstance, func() *statsd.Client {
			return nil
		})
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewNetworkFailureInjectorWithConfig(uid, spec, &ctn, log, &config)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should enter and exit the container network namespace", func() {
			ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")
			ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")
		})

		Context("when the dedicated chain already exists", func() {
			BeforeEach(func() {
				iptListChainsCall.Return([]string{"CHAOS-110e8400446655440000"}, nil)
			})

			It("should not create the chain once again", func() {
				ipt.AssertNumberOfCalls(GinkgoT(), "NewChain", 0)
			})
		})

		Context("when the dedicated chain doesn't exist", func() {
			It("should create the dedicated chain", func() {
				ipt.AssertCalled(GinkgoT(), "NewChain", "filter", "CHAOS-110e8400446655440000")
			})

			It("should create the jump rule", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "OUTPUT", []string{
					"-j", "CHAOS-110e8400446655440000",
				})
			})
		})

		Context("using a CIDR block", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"192.168.0.0/24"}
			})

			It("should inject a rule for the given block", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", mock.Anything, mock.Anything, mock.Anything)
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.0/24", "--dport", "666", "-j", "DROP",
				})
			})
		})

		Context("using a single IP set", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"192.168.0.1", "192.168.0.2"}
			})

			It("should inject a rule for the given IP with a /32 mask", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", mock.Anything, mock.Anything, mock.Anything)
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.1/32", "--dport", "666", "-j", "DROP",
				})
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.2/32", "--dport", "666", "-j", "DROP",
				})
			})
		})

		Context("using a hostname", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"foo.bar.cluster.local"}
			})

			It("should inject a rule per IP resolved by the DNS resolver", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", mock.Anything, mock.Anything, mock.Anything)
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.1/32", "--dport", "666", "-j", "DROP",
				})
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.2/32", "--dport", "666", "-j", "DROP",
				})
			})
		})

		Context("host not specified", func() {
			BeforeEach(func() {
				spec.Hosts = []string{}
			})

			It("should inject a rule matching all the hosts", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", mock.Anything, mock.Anything, mock.Anything)
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "0.0.0.0/0", "--dport", "666", "-j", "DROP",
				})
			})
		})

		Context("using a probability", func() {
			BeforeEach(func() {
				spec.Hosts = []string{"192.168.0.0/24"}
				spec.Probability = 50
			})

			It("should inject a rule for the given probability", func() {
				ipt.AssertCalled(GinkgoT(), "AppendUnique", mock.Anything, mock.Anything, mock.Anything)
				ipt.AssertCalled(GinkgoT(), "AppendUnique", "filter", "CHAOS-110e8400446655440000", []string{
					"-p", "tcp", "-d", "192.168.0.0/24", "--dport", "666", "-m", "statistic", "--mode", "random", "--probability", "0.50", "-j", "DROP",
				})
			})
		})
	})

	Describe("cleaning", func() {
		JustBeforeEach(func() {
			inj.Clean()
		})

		It("should enter and exit the container network namespace", func() {
			ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")
			ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")
		})

		Context("when dedicated chain doesn't exist anymore", func() {
			It("should not try to remove the jump rule", func() {
				ipt.AssertNumberOfCalls(GinkgoT(), "Delete", 0)
			})

			It("should not try to clear nor delete the dedicated chain", func() {
				ipt.AssertNumberOfCalls(GinkgoT(), "ClearChain", 0)
				ipt.AssertNumberOfCalls(GinkgoT(), "DeleteChain", 0)
			})
		})

		Context("when dedicated chain still exists", func() {
			BeforeEach(func() {
				iptListChainsCall.Return([]string{"CHAOS-110e8400446655440000"}, nil)
			})

			It("should clear and delete the dedicated chain", func() {
				ipt.AssertCalled(GinkgoT(), "ClearChain", "filter", "CHAOS-110e8400446655440000")
				ipt.AssertCalled(GinkgoT(), "DeleteChain", "filter", "CHAOS-110e8400446655440000")
			})

			Context("when the jump rule doesn't exist anymore", func() {
				BeforeEach(func() {
					iptExistsCall.Return(false, nil)
				})

				It("should not try to remove the jump rule", func() {
					ipt.AssertNumberOfCalls(GinkgoT(), "Delete", 0)
				})
			})

			Context("when the jump rule still exists", func() {
				It("should not try to remove the jump rule", func() {
					ipt.AssertCalled(GinkgoT(), "Delete", "filter", "OUTPUT", []string{
						"-j", "CHAOS-110e8400446655440000",
					})
				})
			})
		})
	})
})

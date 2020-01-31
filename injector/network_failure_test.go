package injector_test

import (
	"net"
	"reflect"
	"strings"
	"time"

	"bou.ke/monkey"
	"github.com/coreos/go-iptables/iptables"
	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netns"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/container"
	"github.com/DataDog/chaos-fi-controller/datadog"
	. "github.com/DataDog/chaos-fi-controller/injector"
	"github.com/DataDog/datadog-go/statsd"
)

var _ = Describe("Network Failure", func() {
	var f NetworkFailureInjector
	var callEnterNetworkNamespace, callExitNetworkNamespace, iptablesExistsReturnValue bool
	var iptablesAppendRules, iptablesDeleteRules, iptablesListChainsReturnValue []string
	var iptablesClearChainName, iptablesDeleteChainName string

	BeforeEach(func() {
		// tests vars
		f = NetworkFailureInjector{
			ContainerInjector: ContainerInjector{
				Injector: Injector{
					UID: "110e8400-e29b-11d4-a716-446655440000",
					Log: log,
				},
				ContainerID: "fake",
			},
			Spec: &v1beta1.NetworkFailureSpec{
				Hosts:       []string{"127.0.0.1/32"},
				Port:        666,
				Protocol:    "tcp",
				Probability: 100,
			},
		}
		callEnterNetworkNamespace = false
		callExitNetworkNamespace = false
		iptablesAppendRules = []string{}
		iptablesDeleteRules = []string{}
		iptablesClearChainName = ""
		iptablesDeleteChainName = ""
		iptablesExistsReturnValue = true
		iptablesListChainsReturnValue = []string{"CHAOS-110e8400446655440000"}

		// container
		var c container.Container
		monkey.Patch(container.New, func(id string) (container.Container, error) {
			return container.Container{
				ID:               id,
				PID:              666,
				NetworkNamespace: netns.NsHandle(-1),
			}, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(c), "EnterNetworkNamespace", func(container.Container) error {
			callEnterNetworkNamespace = true
			return nil
		})
		monkey.Patch(container.ExitNetworkNamespace, func() error {
			callExitNetworkNamespace = true
			return nil
		})

		// iptables
		var ipt *iptables.IPTables
		monkey.Patch(iptables.New, func() (*iptables.IPTables, error) {
			return nil, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "AppendUnique", func(ref *iptables.IPTables, table string, chain string, rules ...string) error {
			iptablesAppendRules = append(iptablesAppendRules, strings.Join(rules, " "))
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "ListChains", func(*iptables.IPTables, string) ([]string, error) {
			return iptablesListChainsReturnValue, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "NewChain", func(*iptables.IPTables, string, string) error {
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "Delete", func(ref *iptables.IPTables, table string, chain string, rules ...string) error {
			iptablesDeleteRules = append(iptablesDeleteRules, strings.Join(rules, " "))
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "ClearChain", func(ref *iptables.IPTables, table string, chain string) error {
			iptablesClearChainName = chain
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "DeleteChain", func(ref *iptables.IPTables, table string, chain string) error {
			iptablesDeleteChainName = chain
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(ipt), "Exists", func(*iptables.IPTables, string, string, ...string) (bool, error) {
			return iptablesExistsReturnValue, nil
		})

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

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("injection", func() {
		It("should enter and exit the container network namespace", func() {
			f.Inject()
			Expect(callEnterNetworkNamespace).To(Equal(true))
			Expect(callExitNetworkNamespace).To(Equal(true))
		})

		It("should create a dedicated iptables chain with its jump rule", func() {
			f.Inject()
			Expect(len(iptablesAppendRules)).To(Equal(2))
			Expect(iptablesAppendRules[0]).To(Equal("-j CHAOS-110e8400446655440000"))
		})

		Context("using a CIDR block", func() {
			It("should inject a rule for the given block", func() {
				f.Spec.Hosts = []string{"192.168.0.0/24"}
				f.Inject()
				Expect(len(iptablesAppendRules)).To(Equal(2))
				Expect(iptablesAppendRules[1]).To(Equal("-p tcp -d 192.168.0.0/24 --dport 666 -j DROP"))
			})
		})

		Context("using a single IP set", func() {
			It("should inject a rule for the given IP with a /32 mask", func() {
				f.Spec.Hosts = []string{"192.168.0.1", "192.168.0.2"}
				f.Inject()
				Expect(len(iptablesAppendRules)).To(Equal(3))
				Expect(iptablesAppendRules[1]).To(Equal("-p tcp -d 192.168.0.1/32 --dport 666 -j DROP"))
				Expect(iptablesAppendRules[2]).To(Equal("-p tcp -d 192.168.0.2/32 --dport 666 -j DROP"))
			})
		})

		Context("using a hostname", func() {
			It("should inject a rule per IP resolved by the DNS resolver", func() {
				f.Spec.Hosts = []string{"foo.bar.cluster.local"}
				f.Inject()
				Expect(len(iptablesAppendRules)).To(Equal(3))
				Expect(iptablesAppendRules[1]).To(Equal("-p tcp -d 192.168.0.1/32 --dport 666 -j DROP"))
				Expect(iptablesAppendRules[2]).To(Equal("-p tcp -d 192.168.0.2/32 --dport 666 -j DROP"))
			})
		})

		Context("host not specified", func() {
			It("should inject a rule per IP resolved by the DNS resolver", func() {
				f.Spec.Hosts = []string{}
				f.Inject()
				Expect(len(iptablesAppendRules)).To(Equal(2))
				Expect(iptablesAppendRules[1]).To(Equal("-p tcp -d 0.0.0.0/0 --dport 666 -j DROP"))
			})
		})
	})

	Context("using a probability", func() {
		It("should inject a rule for the given probability", func() {
			f.Spec.Hosts = []string{"192.168.0.0/24"}
			f.Spec.Probability = 50
			f.Inject()
			Expect(len(iptablesAppendRules)).To(Equal(2))
			Expect(iptablesAppendRules[1]).To(Equal("-p tcp -d 192.168.0.0/24 --dport 666 -m statistic --mode random --probability 0.50 -j DROP"))
		})
	})

	Context("generating rule parts without modules", func() {
		It("should output correctly formatted iptable rules", func() {
			rules := f.GenerateRuleParts("192.168.0.1")
			rulesString := strings.Join(rules, " ")
			Expect(len(rules)).To(Equal(8))
			Expect(rulesString).To(Equal("-p tcp -d 192.168.0.1 --dport 666 -j DROP"))
		})
	})

	Context("generating rule parts with modules", func() {
		It("should output correctly formatted iptable rules", func() {
			f.Spec.Probability = 15
			rules := f.GenerateRuleParts("192.168.0.1")
			rulesString := strings.Join(rules, " ")
			Expect(len(rules)).To(Equal(14))
			Expect(rulesString).To(Equal("-p tcp -d 192.168.0.1 --dport 666 -m statistic --mode random --probability 0.15 -j DROP"))
		})
	})

	Describe("cleaning (normal case)", func() {
		It("should enter and exit the container network namespace", func() {
			f.Clean()
			Expect(callEnterNetworkNamespace).To(Equal(true))
			Expect(callExitNetworkNamespace).To(Equal(true))
		})

		It("should remove the dedicated chain jump rule", func() {
			f.Clean()
			Expect(len(iptablesDeleteRules)).To(Equal(1))
			Expect(iptablesDeleteRules[0]).To(Equal("-j CHAOS-110e8400446655440000"))
		})

		It("should clear the dedicated chain", func() {
			f.Clean()
			Expect(iptablesClearChainName).To(Equal("CHAOS-110e8400446655440000"))
		})

		It("should delete the dedicated chain", func() {
			f.Clean()
			Expect(iptablesDeleteChainName).To(Equal("CHAOS-110e8400446655440000"))
		})
	})

	Describe("cleaning (edge case, container has already been cleaned)", func() {
		It("should not try to remove the dedicated chain jump rule", func() {
			iptablesExistsReturnValue = false
			f.Clean()
			Expect(len(iptablesDeleteRules)).To(Equal(0))
		})

		It("should not try to clear the dedicated chain", func() {
			iptablesListChainsReturnValue = nil
			f.Clean()
			Expect(iptablesClearChainName).To(Equal(""))
		})

		It("should not try to delete the dedicated chain", func() {
			iptablesListChainsReturnValue = nil
			f.Clean()
			Expect(iptablesDeleteChainName).To(Equal(""))
		})
	})
})

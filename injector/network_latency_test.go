// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"os/exec"
	"reflect"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/container"
	. "github.com/DataDog/chaos-fi-controller/injector"
)

type fakeLink struct {
	attrs *netlink.LinkAttrs
}

func (f fakeLink) Attrs() *netlink.LinkAttrs {
	return f.attrs
}
func (f fakeLink) Type() string {
	return "fake"
}

var _ = Describe("Tc", func() {
	var nli NetworkLatencyInjector
	var cmds []*exec.Cmd
	var cmdErr error
	var netlinkLinkSetTxQLenCallCount int
	var links []fakeLink

	BeforeEach(func() {
		// Variables
		nli = NetworkLatencyInjector{
			ContainerInjector: ContainerInjector{
				Injector: Injector{
					Log: log,
				},
			},
			Spec: &v1beta1.NetworkLatencySpec{
				Delay: 1000,
			},
		}
		cmds = []*exec.Cmd{}
		cmdErr = nil
		netlinkLinkSetTxQLenCallCount = 0
		links = []fakeLink{
			{
				attrs: &netlink.LinkAttrs{
					Name: "lo",
				},
			},
			{
				attrs: &netlink.LinkAttrs{
					Name: "eth0",
				},
			},
		}

		// Patch
		// exec
		var execCommandGuard *monkey.PatchGuard
		// Patch the commands so they are created but never executed
		execCommandGuard = monkey.Patch(exec.Command, func(name string, arg ...string) *exec.Cmd {
			execCommandGuard.Unpatch()
			defer execCommandGuard.Restore()

			// Patch created command instance
			cmd := exec.Command(name, arg...)
			monkey.PatchInstanceMethod(reflect.TypeOf(cmd), "Run", func(*exec.Cmd) error {
				return cmdErr
			})
			cmds = append(cmds, cmd)

			return cmd
		})

		// netlink
		monkey.Patch(netlink.LinkList, func() ([]netlink.Link, error) {
			l := make([]netlink.Link, len(links))
			for i := range links {
				l[i] = links[i]
			}
			return l, nil
		})
		monkey.Patch(netlink.LinkByIndex, func(i int) (netlink.Link, error) {
			return links[i], nil
		})
		monkey.Patch(netlink.LinkByName, func(name string) (netlink.Link, error) {
			for _, link := range links {
				if link.Attrs().Name == name {
					return link, nil
				}
			}

			panic("couldn't retrieve the link into the links list")
		})
		monkey.Patch(netlink.LinkSetTxQLen, func(netlink.Link, int) error {
			netlinkLinkSetTxQLenCallCount++
			return nil
		})
		monkey.Patch(netlink.NewHandle, func(...int) (*netlink.Handle, error) {
			handle := &netlink.Handle{}
			monkey.PatchInstanceMethod(reflect.TypeOf(handle), "RouteGet", func(*netlink.Handle, net.IP) ([]netlink.Route, error) {
				return []netlink.Route{
					netlink.Route{
						LinkIndex: 0,
					},
					netlink.Route{
						LinkIndex: 1,
					},
				}, nil
			})

			return handle, nil
		})

		// container
		monkey.Patch(container.New, func(string) (container.Container, error) {
			c := container.Container{}
			monkey.PatchInstanceMethod(reflect.TypeOf(c), "EnterNetworkNamespace", func(container.Container) error {
				return nil
			})

			return c, nil
		})
		monkey.Patch(container.ExitNetworkNamespace, func() error {
			return nil
		})
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("nli.Inject", func() {
		Context("with no host specified", func() {
			It("should not set or clear the interface qlen", func() {
				nli.Inject()
				Expect(netlinkLinkSetTxQLenCallCount).To(Equal(0))
			})
			It("should add delay to the interfaces", func() {
				nli.Inject()
				Expect(len(cmds)).To(Equal(2))
				Expect(cmds[0].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface 1
				Expect(cmds[0].Args[5]).To(Equal("root"))                  // parent should remain root
				Expect(cmds[0].Args[8]).To(Equal("1s"))                    // delay in string
				Expect(cmds[0].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface 2
				Expect(cmds[1].Args[5]).To(Equal("root"))                  // parent should remain root
				Expect(cmds[1].Args[8]).To(Equal("1s"))                    // delay in string
			})
		})
		Context("with multiple hosts specified and interface without qlen", func() {
			It("should set and clear the interface qlen", func() {
				nli.Spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
				nli.Inject()
				Expect(netlinkLinkSetTxQLenCallCount).To(Equal(4))
			})
			It("should add latency to a prio qdisc and filter on the given hosts", func() {
				nli.Spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
				nli.Inject()
				Expect(len(cmds)).To(Equal(8))

				// first interface
				// prio qdisc creation
				Expect(cmds[0].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[0].Args[8]).To(Equal("prio"))                  // prio qdisc kind
				// delay
				Expect(cmds[1].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[1].Args[5]).To(Equal("parent"))                // parent shouldn't be root
				Expect(cmds[1].Args[7]).To(Equal("netem"))                 // netem qdisc kind
				Expect(cmds[1].Args[8]).To(Equal("delay"))                 // delay module
				Expect(cmds[1].Args[9]).To(Equal("1s"))                    // specified delay
				// filter (one per given host)
				Expect(cmds[2].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[2].Args[15]).To(Equal("1.1.1.1/32"))           // specified host
				Expect(cmds[3].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[3].Args[15]).To(Equal("2.2.2.2/32"))           // specified host

				// second interface
				// prio qdisc creation
				Expect(cmds[4].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[4].Args[8]).To(Equal("prio"))                  // prio qdisc kind
				// delay
				Expect(cmds[5].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[5].Args[5]).To(Equal("parent"))                // parent shouldn't be root
				Expect(cmds[5].Args[7]).To(Equal("netem"))                 // netem qdisc kind
				Expect(cmds[5].Args[8]).To(Equal("delay"))                 // delay module
				Expect(cmds[5].Args[9]).To(Equal("1s"))                    // specified delay
				// filter (one per given host)
				Expect(cmds[6].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[6].Args[15]).To(Equal("1.1.1.1/32"))           // specified host
				Expect(cmds[7].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[7].Args[15]).To(Equal("2.2.2.2/32"))           // specified host
			})
		})
		Context("with multiple hosts specified and interfaces with qlen", func() {
			It("should not set and clear the interface qlen", func() {
				links[0].attrs.TxQLen = 1000
				links[1].attrs.TxQLen = 1000
				nli.Spec.Hosts = []string{"1.1.1.1", "2.2.2.2"}
				nli.Inject()
				Expect(netlinkLinkSetTxQLenCallCount).To(Equal(0))
			})
		})
	})

	Describe("nli.Clean", func() {
		Context("with no error from the tc command", func() {
			It("should clear the interfaces qdisc", func() {
				nli.Clean()
				Expect(len(cmds)).To(Equal(2))
				Expect(cmds[0].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
				Expect(cmds[1].Args[4]).To(Or(Equal("lo"), Equal("eth0"))) // interface
			})
		})
	})
})

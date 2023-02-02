// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"net"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/cgroup/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

var _ = Describe("Failure", func() {
	var (
		ctn                                                     *container.ContainerMock
		inj                                                     Injector
		config                                                  NetworkDisruptionInjectorConfig
		spec                                                    v1beta1.NetworkDisruptionSpec
		cgroupManager                                           *mocks.ManagerMock
		tc                                                      *network.TcMock
		iptables                                                *network.IptablesMock
		nl                                                      *network.NetlinkAdapterMock
		nllink1, nllink2, nllink3                               *network.NetlinkLinkMock
		nllink1TxQlenCall, nllink2TxQlenCall, nllink3TxQlenCall *mock.Call
		nlroute1, nlroute2, nlroute3                            *network.NetlinkRouteMock
		dns                                                     *network.DNSMock
		netnsManager                                            *netns.ManagerMock
		k8sClient                                               *kubernetes.Clientset
		fakeService                                             *corev1.Service
		fakeEndpoint                                            *corev1.Pod
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = &mocks.ManagerMock{}
		cgroupManager.On("RelativePath", mock.Anything).Return("/kubepod.slice/foo")

		// netns
		netnsManager = &netns.ManagerMock{}
		netnsManager.On("Enter").Return(nil)
		netnsManager.On("Exit").Return(nil)

		// tc
		tc = network.NewTcMock()
		tc.On("AddNetem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddPrio", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddFwFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("AddOutputLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		tc.On("DeleteFilter", mock.Anything, mock.Anything).Return(nil)
		tc.On("ClearQdisc", mock.Anything).Return(nil)

		// iptables
		iptables = &network.IptablesMock{}
		iptables.On("CreateChain", mock.Anything).Return(nil)
		iptables.On("ClearAndDeleteChain", mock.Anything).Return(nil)
		iptables.On("AddRuleWithIP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("AddWideFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("AddCgroupFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("PrependRuleSpec", mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteRuleSpec", mock.Anything, mock.Anything).Return(nil)
		iptables.On("DeleteCgroupFilterRule", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// netlink
		nllink1 = &network.NetlinkLinkMock{}
		nllink1.On("Name").Return("lo")
		nllink1.On("SetTxQLen", mock.Anything).Return(nil)
		nllink1TxQlenCall = nllink1.On("TxQLen").Return(0)
		nllink2 = &network.NetlinkLinkMock{}
		nllink2.On("Name").Return("eth0")
		nllink2.On("SetTxQLen", mock.Anything).Return(nil)
		nllink2TxQlenCall = nllink2.On("TxQLen").Return(0)
		nllink3 = &network.NetlinkLinkMock{}
		nllink3.On("Name").Return("eth1")
		nllink3.On("SetTxQLen", mock.Anything).Return(nil)
		nllink3TxQlenCall = nllink3.On("TxQLen").Return(0)

		nlroute1 = &network.NetlinkRouteMock{}
		nlroute1.On("Link").Return(nllink1)
		nlroute1.On("Gateway").Return(net.IP([]byte{}))
		nlroute2 = &network.NetlinkRouteMock{}
		nlroute2.On("Link").Return(nllink2)
		nlroute2.On("Gateway").Return(net.ParseIP("192.168.0.1"))
		nlroute3 = &network.NetlinkRouteMock{}
		nlroute3.On("Link").Return(nllink3)
		nlroute3.On("Gateway").Return(net.ParseIP("192.168.1.1"))

		nl = &network.NetlinkAdapterMock{}
		nl.On("LinkList").Return([]network.NetlinkLink{nllink1, nllink2, nllink3}, nil)
		nl.On("LinkByIndex", 0).Return(nllink1, nil)
		nl.On("LinkByIndex", 1).Return(nllink2, nil)
		nl.On("LinkByIndex", 2).Return(nllink3, nil)
		nl.On("LinkByName", "lo").Return(nllink1, nil)
		nl.On("LinkByName", "eth0").Return(nllink2, nil)
		nl.On("LinkByName", "eth1").Return(nllink3, nil)
		nl.On("DefaultRoutes").Return([]network.NetlinkRoute{nlroute2}, nil)

		// dns
		dns = &network.DNSMock{}
		dns.On("Resolve", "kubernetes.default").Return([]net.IP{net.ParseIP("192.168.0.254")}, nil)
		dns.On("Resolve", "testhost").Return([]net.IP{net.ParseIP("1.1.1.1")}, nil)

		// container
		ctn = &container.ContainerMock{}

		// environment variables
		Expect(os.Setenv(env.InjectorTargetPodHostIP, "10.0.0.2")).To(BeNil())

		// fake kubernetes client and resources
		fakeService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "172.16.0.1",
				Ports: []corev1.ServicePort{
					{
						Port:       80,
						TargetPort: intstr.FromInt(8080),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Selector: map[string]string{
					"app": "foo",
				},
			},
		}

		fakeEndpoint = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-abcd-1234",
				Namespace: "bar",
				Labels: map[string]string{
					"app": "foo",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.1.0.4",
			},
		}

		k8sClient = kubernetes.NewSimpleClientset(fakeService, fakeEndpoint)

		// config
		config = NetworkDisruptionInjectorConfig{
			Config: Config{
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
				Netns:           netnsManager,
				Cgroup:          cgroupManager,
				Level:           chaostypes.DisruptionLevelPod,
				K8sClient:       k8sClient,
			},
			TrafficController: tc,
			Iptables:          iptables,
			NetlinkAdapter:    nl,
			DNSClient:         dns,
		}

		spec = v1beta1.NetworkDisruptionSpec{
			Hosts:          []v1beta1.NetworkDisruptionHostSpec{},
			Services:       []v1beta1.NetworkDisruptionServiceSpec{},
			AllowedHosts:   []v1beta1.NetworkDisruptionHostSpec{},
			Drop:           90,
			Duplicate:      80,
			Corrupt:        70,
			Delay:          1000,
			DelayJitter:    100,
			BandwidthLimit: 10000,
		}
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewNetworkDisruptionInjector(spec, config)
		Expect(err).To(BeNil())
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		// general tests that should work for all contexts
		It("should enter and exit the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should create 2 prio qdiscs on main interfaces", func() {
			tc.AssertCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "root", "1:", uint32(4), mock.Anything)
			tc.AssertCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "1:4", "2:", uint32(2), mock.Anything)
		})

		It("should add an fw filter to classify packets according to their classid set by iptables mark", func() {
			tc.AssertCalled(GinkgoT(), "AddFwFilter", []string{"lo", "eth0", "eth1"}, "2:0", "0x00020002", "2:2")
		})

		It("should apply disruptions to main interfaces 2nd band", func() {
			tc.AssertCalled(GinkgoT(), "AddNetem", []string{"lo", "eth0", "eth1"}, "2:2", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", []string{"lo", "eth0", "eth1"}, "3:", mock.Anything, uint(spec.BandwidthLimit))
		})

		It("should mark packets going out from the identified (container or host) cgroup for the tc fw filter", func() {
			iptables.AssertCalled(GinkgoT(), "AddCgroupFilterRule", "mangle", "OUTPUT", "/kubepod.slice/foo", []string{"-j", "MARK", "--set-mark", chaostypes.InjectorCgroupClassID})
		})

		// qlen cases
		Context("with interfaces without a qlen value", func() {
			It("should set or clear the interface qlen on all interfaces", func() {
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink1.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink2.AssertCalled(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 1000)
				nllink3.AssertCalled(GinkgoT(), "SetTxQLen", 0)
			})
		})

		Context("with interfaces with a qlen value", func() {
			BeforeEach(func() {
				nllink1TxQlenCall.Return(1000)
				nllink2TxQlenCall.Return(1000)
				nllink3TxQlenCall.Return(1000)
			})

			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		// hosts and services filtering cases
		Context("with no hosts specified", func() {
			It("should add a filter to redirect all traffic on main interfaces on the disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, 0, "tcp", "", "1:4")
			})
		})

		Context("with multiple hosts specified", func() {
			BeforeEach(func() {
				spec.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host:      "1.1.1.1",
						Port:      80,
						Protocol:  "tcp",
						ConnState: "new",
					},
					{
						Host:      "2.2.2.2",
						Port:      443,
						Protocol:  "tcp",
						ConnState: "est",
					},
				}
			})

			It("should add a filter to redirect targeted traffic on all interfaces on the disrupted band filter on given hosts as destination IP", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "1.1.1.1/32", 0, 80, "tcp", "+trk+new", "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "2.2.2.2/32", 0, 443, "tcp", "+trk+est", "1:4")
			})
		})

		Context("with multiple services specified", func() {
			BeforeEach(func() {
				spec.Services = []v1beta1.NetworkDisruptionServiceSpec{
					{
						Name:      "foo",
						Namespace: "bar",
					},
				}

				podsWatcher := watch.NewFake()
				servicesWatcher := watch.NewFake()

				k8sClient.PrependWatchReactor("pods", testing.DefaultWatchReactor(podsWatcher, nil))
				k8sClient.PrependWatchReactor("services", testing.DefaultWatchReactor(servicesWatcher, nil))

				// fake watchers for service handling
				go func() {
					// Set up
					time.Sleep(300 * time.Millisecond)
					servicesWatcher.Add(fakeService)
					time.Sleep(300 * time.Millisecond)
					podsWatcher.Add(fakeEndpoint)

					// Deleting a pod
					time.Sleep(300 * time.Millisecond)
					podsWatcher.Delete(fakeEndpoint)
				}()

			})

			It("should add a filter for every service and pods filtered on", func() {
				// wait for all the addFilters at the beginning of injection to complete
				time.Sleep(5 * time.Second)
				tcPriority := 1000                 // first priority set using add filters
				priority := uint32(tcPriority + 3) // 3 add filters are called during injection

				Eventually(func() bool {
					return tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "172.16.0.1/32", 0, 80, "TCP", "", "1:4")
				}, time.Second*5, time.Second).Should(BeTrue())
				Eventually(func() bool {
					return tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "10.1.0.4/32", 0, 8080, "TCP", "", "1:4")
				}, time.Second*5, time.Second).Should(BeTrue())

				Eventually(func() bool {
					return tc.AssertCalled(GinkgoT(), "DeleteFilter", "lo", priority)
				}, time.Second*5, time.Second).Should(BeTrue())
				Eventually(func() bool {
					return tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth0", priority)
				}, time.Second*5, time.Second).Should(BeTrue())
				Eventually(func() bool {
					return tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth1", priority)
				}, time.Second*5, time.Second).Should(BeTrue())
			})

			AfterEach(func() {
				inj.Clean()
			})

		})

		// safeguards
		Context("pod level safeguards", func() {
			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"eth0"}, "1:0", mock.Anything, "nil", "192.168.0.1/32", 0, 0, "tcp", "", "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "10.0.0.2/32", 0, 0, "tcp", "", "1:1")
			})
		})

		Context("node level safeguards", func() {
			BeforeEach(func() {
				config.Level = chaostypes.DisruptionLevelNode
			})

			It("should add a filter to redirect SSH traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "nil", 22, 0, "tcp", "", "1:1")
			})

			It("should add a filter to redirect ARP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "nil", 0, 0, "arp", "", "1:1")
			})

			It("should add a filter to redirect metadata service traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "169.254.169.254/32", 0, 0, "tcp", "", "1:1")
			})
		})

		Context("with ingress flow", func() {
			BeforeEach(func() {
				spec.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Port: 80,
						Flow: "ingress",
					},
				}
			})

			It("should add a filter to redirect all traffic on main interfaces on the disrupted band with specified port as source port", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "0.0.0.0/0", "nil", 80, 0, "tcp", "", "1:4")
			})
		})

		Context("on pod initialization", func() {
			BeforeEach(func() {
				config.OnInit = true
			})

			It("should not add a second prio band with the cgroup filter", func() {
				tc.AssertNotCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "1:4", "2:", uint32(2), mock.Anything)
			})

			It("should apply tc filters to block traffic", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "0.0.0.0/0", 0, 0, "tcp", "", "1:4")
			})
		})

		Context("with allowed hosts", func() {
			BeforeEach(func() {
				spec.AllowedHosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host: "8.8.8.8",
						Port: 53,
					},
				}
			})

			It("should add a filter to redirect traffic going to 8.8.8.8/32 on port 53 on the not disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", mock.Anything, "nil", "8.8.8.8/32", 0, 53, "tcp", "", "1:1")
			})
		})
	})

	Describe("inj.Clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(BeNil())
		})

		It("should enter the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should remove the cgroup iptables packet marking rule", func() {
			iptables.AssertCalled(GinkgoT(), "DeleteCgroupFilterRule", "mangle", "OUTPUT", "/kubepod.slice/foo", []string{"-j", "MARK", "--set-mark", chaostypes.InjectorCgroupClassID})
		})

		Context("qdisc cleanup should happen", func() {
			It("should clear the interfaces qdisc", func() {
				tc.AssertCalled(GinkgoT(), "ClearQdisc", []string{"lo", "eth0", "eth1"})
			})
		})
	})
})

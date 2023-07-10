// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector_test

import (
	"fmt"
	"net"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
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

const (
	secondGatewayIP, targetPodHostIP, testHostIP, clusterIP, podIP = "192.168.0.1", "10.0.0.2", "1.1.1.1", "172.16.0.1", "10.1.0.4"
)

var _ = Describe("Failure", func() {
	var (
		ctn                                                     *container.ContainerMock
		inj                                                     Injector
		config                                                  NetworkDisruptionInjectorConfig
		spec                                                    v1beta1.NetworkDisruptionSpec
		cgroupManager                                           *cgroup.ManagerMock
		isCgroupV2Call                                          *cgroup.ManagerMock_IsCgroupV2_Call
		tc                                                      *network.TrafficControllerMock
		iptables                                                *network.IPTablesMock
		nl                                                      *network.NetlinkAdapterMock
		nllink1, nllink2, nllink3                               *network.NetlinkLinkMock
		nllink1TxQlenCall, nllink2TxQlenCall, nllink3TxQlenCall *network.NetlinkLinkMock_TxQLen_Call
		nlroute1, nlroute2, nlroute3                            *network.NetlinkRouteMock
		dns                                                     *network.DNSClientMock
		netnsManager                                            *netns.ManagerMock
		k8sClient                                               *kubernetes.Clientset
		fakeService                                             *corev1.Service
		fakeService2                                            *corev1.Service
		fakeEndpoint                                            *corev1.Pod
		fakeEndpoint2                                           *corev1.Pod
		zeroIPNet, nilIPNet                                     *net.IPNet
	)

	BeforeEach(func() {
		nilIPNet = nil
		_, zeroIPNet, _ = net.ParseCIDR("0.0.0.0/0")
		// cgroup
		cgroupManager = cgroup.NewManagerMock(GinkgoT())
		cgroupManager.EXPECT().RelativePath(mock.Anything).Return("/kubepod.slice/foo").Maybe()
		cgroupManager.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		isCgroupV2Call = cgroupManager.EXPECT().IsCgroupV2().Return(false)
		isCgroupV2Call.Maybe()

		// netns
		netnsManager = netns.NewManagerMock(GinkgoT())
		netnsManager.EXPECT().Enter().Return(nil).Maybe()
		netnsManager.EXPECT().Exit().Return(nil).Maybe()

		// tc
		tc = network.NewTrafficControllerMock(GinkgoT())
		tc.EXPECT().AddNetem(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddPrio(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil).Maybe()
		tc.EXPECT().AddFwFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddOutputLimit(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().ClearQdisc(mock.Anything).Return(nil).Maybe()

		// iptables
		iptables = network.NewIPTablesMock(GinkgoT())
		iptables.EXPECT().Clear().Return(nil).Maybe()
		iptables.EXPECT().MarkCgroupPath(mock.Anything, mock.Anything).Return(nil).Maybe()
		iptables.EXPECT().MarkClassID(mock.Anything, mock.Anything).Return(nil).Maybe()
		iptables.EXPECT().LogConntrack().Return(nil).Maybe()

		// netlink
		nllink1 = network.NewNetlinkLinkMock(GinkgoT())
		nllink1.EXPECT().Name().Return("lo").Maybe()
		nllink1.EXPECT().SetTxQLen(mock.Anything).Return(nil).Maybe()
		nllink1TxQlenCall = nllink1.EXPECT().TxQLen().Return(0)
		nllink1TxQlenCall.Maybe()
		nllink2 = network.NewNetlinkLinkMock(GinkgoT())
		nllink2.EXPECT().Name().Return("eth0").Maybe()
		nllink2.EXPECT().SetTxQLen(mock.Anything).Return(nil).Maybe()
		nllink2TxQlenCall = nllink2.EXPECT().TxQLen().Return(0)
		nllink2TxQlenCall.Maybe()
		nllink3 = network.NewNetlinkLinkMock(GinkgoT())
		nllink3.EXPECT().Name().Return("eth1").Maybe()
		nllink3.EXPECT().SetTxQLen(mock.Anything).Return(nil).Maybe()
		nllink3TxQlenCall = nllink3.EXPECT().TxQLen().Return(0)
		nllink3TxQlenCall.Maybe()

		nlroute1 = network.NewNetlinkRouteMock(GinkgoT())
		nlroute1.EXPECT().Link().Return(nllink1).Maybe()
		nlroute1.EXPECT().Gateway().Return(net.IP([]byte{})).Maybe()
		nlroute2 = network.NewNetlinkRouteMock(GinkgoT())
		nlroute2.EXPECT().Link().Return(nllink2).Maybe()
		nlroute2.EXPECT().Gateway().Return(net.ParseIP(secondGatewayIP)).Maybe()
		nlroute3 = network.NewNetlinkRouteMock(GinkgoT())
		nlroute3.EXPECT().Link().Return(nllink3).Maybe()
		nlroute3.EXPECT().Gateway().Return(net.ParseIP("192.168.1.1")).Maybe()

		nl = network.NewNetlinkAdapterMock(GinkgoT())
		nl.EXPECT().LinkList().Return([]network.NetlinkLink{nllink1, nllink2, nllink3}, nil).Maybe()
		nl.EXPECT().LinkByIndex(0).Return(nllink1, nil).Maybe()
		nl.EXPECT().LinkByIndex(1).Return(nllink2, nil).Maybe()
		nl.EXPECT().LinkByIndex(2).Return(nllink3, nil).Maybe()
		nl.EXPECT().LinkByName("lo").Return(nllink1, nil).Maybe()
		nl.EXPECT().LinkByName("eth0").Return(nllink2, nil).Maybe()
		nl.EXPECT().LinkByName("eth1").Return(nllink3, nil).Maybe()
		nl.EXPECT().DefaultRoutes().Return([]network.NetlinkRoute{nlroute2}, nil).Maybe()

		// dns
		dns = network.NewDNSClientMock(GinkgoT())
		dns.EXPECT().Resolve("kubernetes.default").Return([]net.IP{net.ParseIP("192.168.0.254")}, nil).Maybe()
		dns.EXPECT().Resolve("testhost").Return([]net.IP{net.ParseIP(testHostIP)}, nil).Maybe()

		// container
		ctn = container.NewContainerMock(GinkgoT())

		// environment variables
		Expect(os.Setenv(env.InjectorTargetPodHostIP, targetPodHostIP)).To(Succeed())

		// fake kubernetes client and resources
		fakeService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: clusterIP,
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

		fakeService2 = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo2",
				Namespace: "bar",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: clusterIP,
				Ports: []corev1.ServicePort{
					{
						Port:       8180,
						TargetPort: intstr.FromInt(8080),
						Protocol:   corev1.ProtocolTCP,
					},
					{
						Port:       8181,
						TargetPort: intstr.FromInt(8080),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Selector: map[string]string{
					"app": "foo2",
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
				PodIP: podIP,
			},
		}

		fakeEndpoint2 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo2-abcd-1234",
				Namespace: "bar",
				Labels: map[string]string{
					"app": "foo2",
				},
			},
			Status: corev1.PodStatus{
				PodIP: podIP,
			},
		}

		k8sClient = kubernetes.NewSimpleClientset(fakeService, fakeService2, fakeEndpoint, fakeEndpoint2)

		// config
		config = NetworkDisruptionInjectorConfig{
			Config: Config{
				TargetContainer: ctn,
				Log:             log,
				MetricsSink:     ms,
				Netns:           netnsManager,
				Cgroup:          cgroupManager,
				Disruption: api.DisruptionArgs{
					Level: chaostypes.DisruptionLevelPod,
				},
				K8sClient: k8sClient,
			},
			TrafficController:   tc,
			IPTables:            iptables,
			NetlinkAdapter:      nl,
			DNSClient:           dns,
			HostResolveInterval: time.Millisecond * 500,
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
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(Succeed())
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
			tc.AssertNumberOfCalls(GinkgoT(), "AddNetem", 1)
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", []string{"lo", "eth0", "eth1"}, "3:", mock.Anything, uint(spec.BandwidthLimit))
		})

		Context("packet marking with cgroups v1", func() {
			It("should mark packets going out from the identified (container or host) cgroup for the tc fw filter", func() {
				cgroupManager.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", chaostypes.InjectorCgroupClassID)
				iptables.AssertCalled(GinkgoT(), "MarkClassID", chaostypes.InjectorCgroupClassID, chaostypes.InjectorCgroupClassID)
			})
		})

		Context("packet marking with cgroups v2", func() {
			BeforeEach(func() {
				isCgroupV2Call.Return(true)
			})

			It("should mark packets going out from the identified (container or host) cgroup for the tc fw filter", func() {
				iptables.AssertCalled(GinkgoT(), "MarkCgroupPath", "/kubepod.slice/foo", chaostypes.InjectorCgroupClassID)
			})
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
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, zeroIPNet, 0, 0, network.TCP, network.ConnStateUndefined, "1:4")
			})
		})

		Context("with multiple hosts specified", func() {
			BeforeEach(func() {
				spec.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host:      testHostIP,
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
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(testHostIP), 0, 80, network.TCP, network.ConnStateNew, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse("2.2.2.2"), 0, 443, network.TCP, network.ConnStateEstablished, "1:4")
			})
		})

		FContext("when resolved host IPs change", func() {
			BeforeEach(func() {
				spec.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host:      testHostIP,
						Port:      80,
						Protocol:  "tcp",
						ConnState: "new",
					},
				}
			})

			It("should update the filters", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(testHostIP), 0, 80, network.TCP, network.ConnStateNew, "1:4")

				const newTestHostIP = "2.2.2.2"
				dns.EXPECT().Resolve("testhost").Return([]net.IP{net.ParseIP(newTestHostIP)}, nil).Maybe()
				time.Sleep(time.Second) // Wait for changed IPs to be caught by the hostWatcher

				tc.AssertCalled(GinkgoT(), "DeleteFilter", []string{"lo", "eth0", "eth1"}, "")
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(newTestHostIP), 0, 80, network.TCP, network.ConnStateNew, "1:4")

			})
		})

		Context("with one service specified", func() {
			var podsWatcher, servicesWatcher *watch.FakeWatcher

			BeforeEach(func() {
				spec.Services = []v1beta1.NetworkDisruptionServiceSpec{
					{
						Name:      "foo",
						Namespace: "bar",
					},
				}

				podsWatcher = watch.NewFakeWithChanSize(2, false)
				servicesWatcher = watch.NewFakeWithChanSize(2, false)

				k8sClient.PrependWatchReactor("pods", testing.DefaultWatchReactor(podsWatcher, nil))
				k8sClient.PrependWatchReactor("services", testing.DefaultWatchReactor(servicesWatcher, nil))

				// Set up adding 2 services
				servicesWatcher.Add(fakeService)

				// Set up adding 2 pods
				podsWatcher.Add(fakeEndpoint)

				ports := []corev1.ServicePort{
					{
						Port:       81,
						TargetPort: intstr.FromInt(8080),
						Protocol:   corev1.ProtocolTCP,
					},
				}

				updatedFakeService := fakeService.DeepCopy()
				updatedFakeService.Spec.Ports = ports

				servicesWatcher.Modify(updatedFakeService)

				// delete the pod 1
				podsWatcher.Delete(fakeEndpoint)
			})

			It("should add a filter for every service and pods filtered on, modify the filter and then delete a filter", func() {
				WatchersAreEmpty(servicesWatcher, podsWatcher)

				priority := uint32(0)

				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(clusterIP), 0, 80, network.TCP, network.ConnStateUndefined, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(podIP), 0, 8080, network.TCP, network.ConnStateUndefined, "1:4")

				tc.AssertCalled(GinkgoT(), "DeleteFilter", "lo", priority)
				tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth0", priority)
				tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth1", priority)

				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(clusterIP), 0, 81, network.TCP, network.ConnStateUndefined, "1:4") // priority 1005

				tc.AssertCalled(GinkgoT(), "DeleteFilter", "lo", priority)
				tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth0", priority)
				tc.AssertCalled(GinkgoT(), "DeleteFilter", "eth1", priority)
			})

			AfterEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})
		})

		Context("with one service and one port specified", func() {
			var servicesWatcher, podsWatcher *watch.FakeWatcher

			BeforeEach(func() {
				spec.Services = []v1beta1.NetworkDisruptionServiceSpec{
					{
						Name:      "foo2",
						Namespace: "bar",
						Ports: []v1beta1.NetworkDisruptionServicePortSpec{
							{
								Port: 8180,
							},
						},
					},
				}

				podsWatcher = watch.NewFakeWithChanSize(1, false)
				servicesWatcher = watch.NewFakeWithChanSize(1, false)

				k8sClient.PrependWatchReactor("pods", testing.DefaultWatchReactor(podsWatcher, nil))
				k8sClient.PrependWatchReactor("services", testing.DefaultWatchReactor(servicesWatcher, nil))

				// Set up adding 1 service
				servicesWatcher.Add(fakeService2)

				// Set up adding 1 pod
				podsWatcher.Add(fakeEndpoint2)
			})

			It("should add a filter on allowed port, not on not specified port", func() {
				WatchersAreEmpty(servicesWatcher, podsWatcher)

				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(clusterIP), 0, 8180, network.TCP, network.ConnStateUndefined, "1:4")
				tc.AssertNotCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(clusterIP), 0, 8181, network.TCP, network.ConnStateUndefined, "1:4")
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse(podIP), 0, 8080, network.TCP, network.ConnStateUndefined, "1:4")
			})

			AfterEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})
		})

		// safeguards
		Context("pod level safeguards", func() {
			It("should add a filter to redirect default gateway IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"eth0"}, "1:0", "", nilIPNet, buildSingleIPNet(secondGatewayIP), 0, 0, network.TCP, network.ConnStateUndefined, "1:1")
			})

			It("should add a filter to redirect node IP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNet(targetPodHostIP), 0, 0, network.TCP, network.ConnStateUndefined, "1:1")
			})
		})

		Context("node level safeguards", func() {
			BeforeEach(func() {
				config.Disruption.Level = chaostypes.DisruptionLevelNode
			})

			It("should add a filter to redirect SSH traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, nilIPNet, 22, 0, network.TCP, network.ConnStateUndefined, "1:1")
			})

			It("should add a filter to redirect ARP traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, nilIPNet, 0, 0, network.ARP, network.ConnStateUndefined, "1:1")
			})

			It("should add a filter to redirect metadata service traffic on a non-disrupted band", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNet("169.254.169.254"), 0, 0, network.TCP, network.ConnStateUndefined, "1:1")
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
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", zeroIPNet, nilIPNet, 80, 0, network.TCP, network.ConnStateUndefined, "1:4")
			})
		})

		Context("on pod initialization", func() {
			BeforeEach(func() {
				config.Disruption.OnInit = true
			})

			It("should not add a second prio band with the cgroup filter", func() {
				tc.AssertNotCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "1:4", "2:", uint32(2), mock.Anything)
			})

			It("should apply tc filters to block traffic", func() {
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, zeroIPNet, 0, 0, network.TCP, network.ConnStateUndefined, "1:4")
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
				tc.AssertCalled(GinkgoT(), "AddFilter", []string{"lo", "eth0", "eth1"}, "1:0", "", nilIPNet, buildSingleIPNetUsingParse("8.8.8.8"), 0, 53, network.TCP, network.ConnStateUndefined, "1:1")
			})
		})

		Context("with a re-injection", func() {
			JustBeforeEach(func() {
				// When an update event is sent to the injector, the disruption method Clean is called before its Inject method.
				// If the method Clean is not called the AddNetem operations will stack up.
				Expect(inj.Clean()).To(Succeed())
				Expect(inj.Inject()).To(Succeed())
			})

			It("should not stack up AddNetem operations", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", []string{"lo", "eth0", "eth1"}, "2:2", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
				// The first call come from the first injection and the second is form the last injection. So the sum of calls si two.
				tc.AssertNumberOfCalls(GinkgoT(), "AddNetem", 2)
			})
		})
	})

	Describe("inj.Clean", func() {
		JustBeforeEach(func() {
			Expect(inj.Clean()).To(Succeed())
		})

		It("should enter the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should remove iptables rules", func() {
			iptables.AssertCalled(GinkgoT(), "Clear")
		})

		Context("qdisc cleanup should happen", func() {
			It("should clear the interfaces qdisc", func() {
				tc.AssertCalled(GinkgoT(), "ClearQdisc", []string{"lo", "eth0", "eth1"})
			})
		})
	})
})

func buildSingleIPNet(ip string) *net.IPNet {
	return &net.IPNet{
		IP:   net.ParseIP(ip),
		Mask: net.CIDRMask(32, 32),
	}
}

func buildSingleIPNetUsingParse(ip string) *net.IPNet {
	_, r, _ := net.ParseCIDR(fmt.Sprintf("%s/32", ip))
	return r
}

func WatchersAreEmpty(watchers ...*watch.FakeWatcher) {
	Eventually(func() bool {
		for _, watcher := range watchers {
			if len(watcher.ResultChan()) != 0 {
				return false
			}
		}
		return true
	}).Within(10 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

	// Even if channels are "empty" we could still have not YET computed events
	// as it's only for tests, we consider it's good enough for now to wait a small amount of time
	// IRL it's OK to have the disruption injected at an approximate time
	<-time.After(100 * time.Millisecond)
}

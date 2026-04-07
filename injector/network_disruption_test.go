// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/bpfdisrupt"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// bpfCmdRunnerMock implements bpfdisrupt.CmdRunner for testing
type bpfCmdRunnerMock struct {
	mock.Mock
}

func (m *bpfCmdRunnerMock) Run(args []string) (int, string, error) {
	ret := m.Called(args)
	return ret.Int(0), ret.String(1), ret.Error(2)
}

var _ bpfdisrupt.CmdRunner = (*bpfCmdRunnerMock)(nil)

const (
	secondGatewayIP, targetPodHostIP, clusterIP, podIP = "192.168.0.1", "10.0.0.2", "172.16.0.1", "10.1.0.4"
	testHostIP, testHostIPTwo, testHostIPThree         = "1.1.1.1", "2.2.2.2", "3.3.3.3"
	fakeService2PortName                               = "foo2-port"
)

var _ = Describe("Failure", func() {
	var (
		ctn                                                     *container.ContainerMock
		inj                                                     Injector
		config                                                  NetworkDisruptionInjectorConfig
		err                                                     error
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
		bpfCmdRunner                                            *bpfCmdRunnerMock
		netnsManager                                            *netns.ManagerMock
		k8sClient                                               *kubernetes.Clientset
		fakeService                                             *corev1.Service
		fakeService2                                            *corev1.Service
		fakeEndpoint                                            *corev1.Pod
		fakeEndpoint2                                           *corev1.Pod
		BPFConfigInformerMock                                   *ebpf.ConfigInformerMock
	)

	BeforeEach(func() {
		// cgroup
		cgroupManager = cgroup.NewManagerMock(GinkgoT())
		cgroupManager.EXPECT().RelativePath(mock.Anything).Return("/kubepod.slice/foo").Maybe()
		cgroupManager.EXPECT().Write(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		isCgroupV2Call = cgroupManager.EXPECT().IsCgroupV2().Return(false)
		isCgroupV2Call.Maybe()

		// ebpf config informer
		BPFConfigInformerMock = ebpf.NewConfigInformerMock(GinkgoT())
		BPFConfigInformerMock.EXPECT().ValidateRequiredSystemConfig().Return(nil).Maybe()
		BPFConfigInformerMock.EXPECT().GetMapTypes().Return(ebpf.MapTypes{HaveArrayMapType: true}).Maybe()

		// netns
		netnsManager = netns.NewManagerMock(GinkgoT())
		netnsManager.EXPECT().Enter().Return(nil).Maybe()
		netnsManager.EXPECT().Exit().Return(nil).Maybe()

		// tc
		tc = network.NewTrafficControllerMock(GinkgoT())
		tc.EXPECT().AddNetem(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddPrio(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil).Maybe()
		tc.EXPECT().AddFlowerFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddOutputLimit(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().ClearQdisc(mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddClsact(mock.Anything).Return(nil).Maybe()
		tc.EXPECT().AddIngressBPFFilter(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		tc.EXPECT().ClearIngressQdisc(mock.Anything).Return(nil).Maybe()
		// BPF engine attaches egress classifier at parent 1:0 with the disruption BPF object
		tc.EXPECT().AddBPFFilter(mock.Anything, "1:0", "/usr/local/bin/bpf-network-disruption.bpf.o", "1:4", "tc_egress_disruption").Return(nil).Maybe()

		// iptables
		iptables = network.NewIPTablesMock(GinkgoT())
		iptables.EXPECT().Clear().Return(nil).Maybe()
		iptables.EXPECT().MarkCgroupPath(mock.Anything, mock.Anything).Return(nil).Maybe()
		iptables.EXPECT().MarkClassID(mock.Anything, mock.Anything).Return(nil).Maybe()
		iptables.EXPECT().LogConntrack().Return(nil).Maybe()

		// netlink
		nllink1 = network.NewNetlinkLinkMock(GinkgoT())
		nllink1.EXPECT().Name().Return("lo").Maybe()
		nllink1.EXPECT().Index().Return(1).Maybe()
		nllink1.EXPECT().SetTxQLen(mock.Anything).Return(nil).Maybe()
		nllink1TxQlenCall = nllink1.EXPECT().TxQLen().Return(0)
		nllink1TxQlenCall.Maybe()
		nllink2 = network.NewNetlinkLinkMock(GinkgoT())
		nllink2.EXPECT().Name().Return("eth0").Maybe()
		nllink2.EXPECT().Index().Return(2).Maybe()
		nllink2.EXPECT().SetTxQLen(mock.Anything).Return(nil).Maybe()
		nllink2TxQlenCall = nllink2.EXPECT().TxQLen().Return(0)
		nllink2TxQlenCall.Maybe()
		nllink3 = network.NewNetlinkLinkMock(GinkgoT())
		nllink3.EXPECT().Name().Return("eth1").Maybe()
		nllink3.EXPECT().Index().Return(3).Maybe()
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
		nl.EXPECT().LinkList(mock.Anything, mock.Anything).Return([]network.NetlinkLink{nllink1, nllink2, nllink3}, nil).Maybe()
		nl.EXPECT().LinkByIndex(0).Return(nllink1, nil).Maybe()
		nl.EXPECT().LinkByIndex(1).Return(nllink2, nil).Maybe()
		nl.EXPECT().LinkByIndex(2).Return(nllink3, nil).Maybe()
		nl.EXPECT().LinkByName("lo").Return(nllink1, nil).Maybe()
		nl.EXPECT().LinkByName("eth0").Return(nllink2, nil).Maybe()
		nl.EXPECT().LinkByName("eth1").Return(nllink3, nil).Maybe()
		nl.EXPECT().DefaultRoutes().Return([]network.NetlinkRoute{nlroute2}, nil).Maybe()
		nl.EXPECT().AddIFBDevice(mock.Anything).Return(nllink1, nil).Maybe()
		nl.EXPECT().DeleteIFBDevice(mock.Anything).Return(nil).Maybe()

		// bpf cmd runner (for BPF disruption engine)
		bpfCmdRunner = &bpfCmdRunnerMock{}
		bpfCmdRunner.On("Run", mock.Anything).Return(0, "", nil)

		// dns
		dns = network.NewDNSClientMock(GinkgoT())
		dns.EXPECT().ResolveWithStrategy("kubernetes", "default").Return([]net.IP{net.ParseIP("192.168.0.254")}, nil).Maybe()

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
						Name:       "test-port-name",
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
						Name:       fakeService2PortName,
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
				K8sClient:   k8sClient,
				InjectorCtx: context.Background(),
			},
			TrafficController:   tc,
			IPTables:            iptables,
			NetlinkAdapter:      nl,
			DNSClient:           dns,
			HostResolveInterval: time.Millisecond * 500,
			BPFConfigInformer:   BPFConfigInformerMock,
			BPFDisruptCmdRunner: bpfCmdRunner,
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
			HTTP: &v1beta1.NetworkHTTPFilters{
				Methods: v1beta1.HTTPMethods{},
				Paths:   v1beta1.HTTPPaths{v1beta1.DefaultHTTPPathFilter},
			},
		}
	})

	JustBeforeEach(func() {
		inj, err = NewNetworkDisruptionInjector(spec, config)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			err = inj.Inject()
		})

		It("should not raise an error", func() {
			Expect(err).ShouldNot(HaveOccurred())
		})

		// general tests that should work for all contexts
		It("should enter and exit the target network namespace", func() {
			netnsManager.AssertCalled(GinkgoT(), "Enter")
			netnsManager.AssertCalled(GinkgoT(), "Exit")
		})

		It("should create only the root prio qdisc on main interfaces (no nested prio without HTTP filters)", func() {
			tc.AssertCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "root", "1:", uint32(4), mock.Anything)
			tc.AssertNumberOfCalls(GinkgoT(), "AddPrio", 1)
		})

		It("should not add a flower filter without HTTP filters (BPF engine handles classification)", func() {
			tc.AssertNotCalled(GinkgoT(), "AddFlowerFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})

		It("should apply disruptions directly at band 1:4 (no nested prio without HTTP filters)", func() {
			tc.AssertCalled(GinkgoT(), "AddNetem", []string{"lo", "eth0", "eth1"}, "1:4", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
			tc.AssertNumberOfCalls(GinkgoT(), "AddNetem", 1)
			tc.AssertCalled(GinkgoT(), "AddOutputLimit", []string{"lo", "eth0", "eth1"}, "2:", mock.Anything, uint(spec.BandwidthLimit))
		})

		Context("packet marking with cgroups v1 (no HTTP filters)", func() {
			It("should not mark packets when no HTTP filters are active (BPF engine handles classification)", func() {
				cgroupManager.AssertNotCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", chaostypes.InjectorCgroupClassID)
				iptables.AssertNotCalled(GinkgoT(), "MarkClassID", chaostypes.InjectorCgroupClassID, chaostypes.InjectorCgroupClassID)
			})
		})

		Context("packet marking with cgroups v2 (no HTTP filters)", func() {
			BeforeEach(func() {
				isCgroupV2Call.Return(true)
			})

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not mark packets when no HTTP filters are active (BPF engine handles classification)", func() {
				iptables.AssertNotCalled(GinkgoT(), "MarkCgroupPath", "/kubepod.slice/foo", chaostypes.InjectorCgroupClassID)
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not set and clear the interface qlen", func() {
				nllink1.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink2.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
				nllink3.AssertNumberOfCalls(GinkgoT(), "SetTxQLen", 0)
			})
		})

		// hosts and services filtering cases
		Context("with no hosts specified", func() {
			It("should not add tc filters for match-all (BPF engine handles match-all via LPM trie)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not add tc filters for hosts (BPF engine handles host-based rules via LPM trie)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		})

		Context("host watcher", func() {
			BeforeEach(func() {
				spec.Hosts = []v1beta1.NetworkDisruptionHostSpec{
					{
						Host: "testhost",
						Port: 80,
					},
				}

				dns.EXPECT().ResolveWithStrategy("testhost", "").Return([]net.IP{net.ParseIP(testHostIP), net.ParseIP(testHostIPTwo)}, nil).Once()
			})

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not use tc filters for hosts (BPF engine handles host rules)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})

			It("should update BPF rules when the IP changes", func() {
				dns.EXPECT().ResolveWithStrategy("testhost", "").Return([]net.IP{net.ParseIP(testHostIPTwo), net.ParseIP(testHostIPThree)}, nil).Maybe()
				time.Sleep(time.Second) // Wait for changed IPs to be caught by the hostWatcher

				// BPF host watcher calls engine.UpdateRules which invokes bpfCmdRunner.Run
				// No tc AddFilter/DeleteFilter calls should happen for hosts
				tc.AssertNotCalled(GinkgoT(), "DeleteFilter", mock.Anything, mock.Anything)
			})

			It("should update BPF rules when new IPs are added", func() {
				dns.EXPECT().ResolveWithStrategy("testhost", "").Return([]net.IP{net.ParseIP(testHostIP), net.ParseIP(testHostIPTwo), net.ParseIP(testHostIPThree)}, nil).Maybe()
				time.Sleep(time.Second) // Wait for changed IPs to be caught by the hostWatcher

				// No tc filter operations for hosts - BPF engine handles updates
				tc.AssertNotCalled(GinkgoT(), "DeleteFilter", mock.Anything, mock.Anything)
			})

			It("should update BPF rules when IPs are removed", func() {
				dns.EXPECT().ResolveWithStrategy("testhost", "").Return([]net.IP{net.ParseIP(testHostIP)}, nil).Maybe()
				time.Sleep(time.Second) // Wait for changed IPs to be caught by the hostWatcher

				// No tc filter operations for hosts - BPF engine handles updates
				tc.AssertNotCalled(GinkgoT(), "DeleteFilter", mock.Anything, mock.Anything)
			})

			It("should not update when IPs do not change", func() {
				// No tc filter operations expected
				tc.AssertNotCalled(GinkgoT(), "DeleteFilter", mock.Anything, mock.Anything)
			})

			AfterEach(func() {
				Expect(inj.Clean()).To(Succeed())
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not use tc filters for services (BPF engine handles service rules via LPM trie)", func() {
				WatchersAreEmpty(servicesWatcher, podsWatcher)

				// Services are now BPF rules, not tc flower filters
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				tc.AssertNotCalled(GinkgoT(), "DeleteFilter", mock.Anything, mock.Anything)
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
								Name: fakeService2PortName,
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not use tc filters for services (BPF engine handles service rules via LPM trie)", func() {
				WatchersAreEmpty(servicesWatcher, podsWatcher)

				// Services are now BPF rules, not tc flower filters
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})

			AfterEach(func() {
				Expect(inj.Clean()).To(Succeed())
			})
		})

		// safeguards
		Context("pod level safeguards", func() {
			It("should not add tc filters for safeguards (BPF engine handles safeguards via ALLOW rules in LPM trie)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		})

		Context("node level safeguards", func() {
			BeforeEach(func() {
				config.Disruption.Level = chaostypes.DisruptionLevelNode
			})

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not add tc filters for safeguards (BPF engine handles safeguards via ALLOW rules in LPM trie)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not add tc filters for ingress hosts (BPF engine handles direction natively)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		})

		Context("on pod initialization", func() {
			BeforeEach(func() {
				config.Disruption.OnInit = true
			})

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not add a second prio band with the cgroup filter", func() {
				tc.AssertNotCalled(GinkgoT(), "AddPrio", []string{"lo", "eth0", "eth1"}, "1:4", "2:", uint32(2), mock.Anything)
			})

			It("should not add tc filters for match-all (BPF engine handles classification)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not add tc filters for allowed hosts (BPF engine handles allowed hosts via ALLOW rules in LPM trie)", func() {
				tc.AssertNotCalled(GinkgoT(), "AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		})

		Context("with a re-injection", func() {
			JustBeforeEach(func() {
				// When an update event is sent to the injector, the disruption method Clean is called before its Inject method.
				// If the method Clean is not called the AddNetem operations will stack up.
				Expect(inj.Clean()).To(Succeed())
				Expect(inj.Inject()).To(Succeed())
			})

			It("should not raise an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should not stack up AddNetem operations", func() {
				tc.AssertCalled(GinkgoT(), "AddNetem", []string{"lo", "eth0", "eth1"}, "1:4", mock.Anything, time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
				// The first call come from the first injection and the second is form the last injection. So the sum of calls is two.
				tc.AssertNumberOfCalls(GinkgoT(), "AddNetem", 2)
			})
		})

		Context("with method and path filters", func() {
			DescribeTable("success cases",
				func(methods v1beta1.HTTPMethods, paths v1beta1.HTTPPaths) {
					// Arrange
					interfaces := []string{"lo", "eth0", "eth1"}

					spec.HTTP.Methods = methods
					spec.HTTP.Paths = paths

					BPFConfigInformerMock = ebpf.NewConfigInformerMock(GinkgoT())
					BPFConfigInformerMock.EXPECT().ValidateRequiredSystemConfig().Return(nil).Maybe()
					BPFConfigInformerMock.EXPECT().GetMapTypes().Return(ebpf.MapTypes{HaveArrayMapType: true}).Once()
					config.BPFConfigInformer = BPFConfigInformerMock

					// tc
					tc.EXPECT().AddBPFFilter(interfaces, "2:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "2:2", "classifier_methods").Return(nil).Once()
					tc.EXPECT().AddBPFFilter(interfaces, "3:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "3:2", "classifier_paths").Return(nil).Once()

					configBPFArgs := []interface{}{}

					for _, path := range paths {
						configBPFArgs = append(configBPFArgs, "--path", string(path))
					}

					for _, method := range methods {
						configBPFArgs = append(configBPFArgs, "--method", strings.ToUpper(method))
					}

					tc.EXPECT().ConfigBPFFilter(mock.Anything, configBPFArgs...).Return(nil)

					var err error
					inj, err = NewNetworkDisruptionInjector(spec, config)
					Expect(err).ShouldNot(HaveOccurred())

					// Action
					Expect(inj.Inject()).To(Succeed())

					// Assert
					By("creating three prio bands")
					tc.AssertCalled(GinkgoT(), "AddPrio", interfaces, "1:4", "2:", uint32(2), mock.Anything)
					tc.AssertCalled(GinkgoT(), "AddPrio", interfaces, "2:2", "3:", uint32(2), mock.Anything)
					tc.AssertCalled(GinkgoT(), "AddPrio", interfaces, "3:2", "4:", uint32(2), mock.Anything)

					By("adding an fw filter to classify packets according to their classid set by iptables mark")
					tc.AssertCalled(GinkgoT(), "AddFlowerFilter", interfaces, "4:0", "0x00020002", "4:2")

					By("adding an BPF filter to classify packets according to their method")
					tc.AssertCalled(GinkgoT(), "AddBPFFilter", interfaces, "2:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "2:2", "classifier_methods")

					By("adding an BPF filter to classify packets according to their path")
					tc.AssertCalled(GinkgoT(), "AddBPFFilter", interfaces, "3:0", "/usr/local/bin/bpf-network-tc-filter.bpf.o", "3:2", "classifier_paths")

					By("configuring the BPF filter")
					expectedConfigBPFArgs := append([]interface{}{mock.Anything}, configBPFArgs...)
					tc.AssertCalled(GinkgoT(), "ConfigBPFFilter", expectedConfigBPFArgs...)
				},
				Entry("With a HTTPMethodGET method and / path",
					v1beta1.HTTPMethods{http.MethodGet},
					v1beta1.HTTPPaths{v1beta1.DefaultHTTPPathFilter},
				),
				Entry("With a DELETE method and / path",
					v1beta1.HTTPMethods{http.MethodDelete},
					v1beta1.HTTPPaths{v1beta1.DefaultHTTPPathFilter},
				),
				Entry("With a POST method and /test path",
					v1beta1.HTTPMethods{http.MethodPost},
					v1beta1.HTTPPaths{"/test"},
				),
				Entry("With a POST and delete methods and /test path",
					v1beta1.HTTPMethods{http.MethodPost, "delete"},
					v1beta1.HTTPPaths{"/test"},
				),
			)

			Describe("error cases", func() {
				BeforeEach(func() {
					spec.HTTP.Methods = v1beta1.HTTPMethods{http.MethodGet}
					spec.HTTP.Paths = v1beta1.HTTPPaths{v1beta1.DefaultHTTPPathFilter}
				})

				When("the node does not have eBPF requirements", func() {
					BeforeEach(func() {
						// Arrange
						BPFConfigInformerMock = ebpf.NewConfigInformerMock(GinkgoT())
						BPFConfigInformerMock.EXPECT().ValidateRequiredSystemConfig().Return(fmt.Errorf("an error happened")).Once()
						config.BPFConfigInformer = BPFConfigInformerMock
					})

					It("should return the error", func() {
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError("an error happened"))
					})
				})

				When("the bpf map type array is not supported", func() {
					BeforeEach(func() {
						BPFConfigInformerMock = ebpf.NewConfigInformerMock(GinkgoT())
						BPFConfigInformerMock.EXPECT().ValidateRequiredSystemConfig().Return(nil).Once()
						BPFConfigInformerMock.EXPECT().GetMapTypes().Return(ebpf.MapTypes{
							HaveHashMapType:                true,
							HaveArrayMapType:               false,
							HaveProgArrayMapType:           true,
							HavePerfEventArrayMapType:      true,
							HavePercpuHashMapType:          true,
							HavePercpuArrayMapType:         true,
							HaveStackTraceMapType:          true,
							HaveCgroupArrayMapType:         true,
							HaveLruHashMapType:             true,
							HaveLruPercpuHashMapType:       true,
							HaveLpmTrieMapType:             true,
							HaveArrayOfMapsMapType:         true,
							HaveHashOfMapsMapType:          true,
							HaveDevmapMapType:              true,
							HaveSockmapMapType:             true,
							HaveCpumapMapType:              true,
							HaveXskmapMapType:              true,
							HaveSockhashMapType:            true,
							HaveCgroupStorageMapType:       true,
							HaveReuseportSockarrayMapType:  true,
							HavePercpuCgroupStorageMapType: true,
							HaveQueueMapType:               true,
							HaveStackMapType:               true,
						})
						config.BPFConfigInformer = BPFConfigInformerMock
					})

					It("should return an error", func() {
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError("the http network failure needs the array map type, but the kernel does not support this type of map"))
					})
				})
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

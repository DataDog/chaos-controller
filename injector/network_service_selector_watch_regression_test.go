// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

var _ = Describe("Network service selector watch replacement", func() {
	It("replaces the old watch from the new snapshot and tracks pending endpoints", func() {
		ctx, cancel := context.WithCancel(context.Background())
		serviceEvents := watch.NewRaceFreeFake()
		oldPodEvents := watch.NewRaceFreeFake()
		newPodEvents := watch.NewRaceFreeFake()
		type watchRequest struct{ selector, version string }
		requests := make(chan watchRequest, 4)
		pendingFilter := make(chan struct{}, 1)
		client := fake.NewSimpleClientset()
		client.PrependWatchReactor("services", func(ktesting.Action) (bool, watch.Interface, error) {
			return true, serviceEvents, nil
		})
		client.PrependWatchReactor("pods", func(action ktesting.Action) (bool, watch.Interface, error) {
			r := action.(ktesting.WatchAction).GetWatchRestrictions()
			requests <- watchRequest{r.Labels.String(), r.ResourceVersion}
			switch r.Labels.String() {
			case "app=old":
				return true, oldPodEvents, nil
			case "app=new":
				return true, newPodEvents, nil
			default:
				return true, nil, fmt.Errorf("unexpected pod selector %s", r.Labels)
			}
		})
		ready := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ready", Namespace: "default", UID: "ready", Labels: map[string]string{"app": "new"}}, Status: v1.PodStatus{PodIP: "10.0.0.3"}}
		pending := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "default", UID: "pending", Labels: map[string]string{"app": "new"}}}
		client.PrependReactor("list", "pods", func(ktesting.Action) (bool, runtime.Object, error) {
			return true, &v1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "42"}, Items: []v1.Pod{ready, pending}}, nil
		})
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.On("AddFilter", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(uint32(1001), nil).Run(func(args mock.Arguments) {
			if ip, ok := args.Get(4).(*net.IPNet); ok && ip != nil && ip.IP.Equal(net.ParseIP("10.0.0.4")) {
				select {
				case pendingFilter <- struct{}{}:
				default:
				}
			}
		})
		traffic.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil).Maybe()
		ns := netns.NewManagerMock(GinkgoT())
		ns.EXPECT().Enter().Return(nil).Maybe()
		ns.EXPECT().Exit().Return(nil).Maybe()
		inj := networkDisruptionInjector{config: NetworkDisruptionInjectorConfig{
			Config: Config{Log: zap.NewNop().Sugar(), Netns: ns, K8sClient: client}, TrafficController: traffic,
		}}
		ports := []v1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP}}
		watcher := serviceWatcher{
			watchedServiceSpec: v1beta1.NetworkDisruptionServiceSpec{Name: "svc", Namespace: "default"},
			servicePorts:       ports, labelServiceSelector: "app=old", podsResourceVersion: "7",
		}
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			inj.watchServiceChanges(ctx, watcher, []string{"eth0"}, "1:4")
		}()
		DeferCleanup(func() {
			cancel()
			Eventually(done, 5*time.Second).Should(BeClosed())
			serviceEvents.Stop()
			oldPodEvents.Stop()
			newPodEvents.Stop()
		})
		Eventually(requests, 5*time.Second).Should(Receive(Equal(watchRequest{"app=old", "7"})))
		serviceEvents.Modify(&v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "svc", Namespace: "default"},
			Spec:       v1.ServiceSpec{Type: v1.ServiceTypeClusterIP, ClusterIP: "10.96.0.10", Selector: map[string]string{"app": "new"}, Ports: ports},
		})
		Eventually(requests, 5*time.Second).Should(Receive(Equal(watchRequest{"app=new", "42"})))
		Expect(oldPodEvents.IsStopped()).To(BeTrue())
		pending.Status.PodIP = "10.0.0.4"
		newPodEvents.Modify(&pending)
		Eventually(pendingFilter, 5*time.Second).Should(Receive())
	})
})

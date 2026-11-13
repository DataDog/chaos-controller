// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Service percentage pod events", func() {
	It("reselects endpoints after additions, IP assignment, and deletion", func() {
		ctx := context.Background()
		client := fake.NewSimpleClientset()
		traffic := network.NewTrafficControllerMock(GinkgoT())
		traffic.EXPECT().AddFilter(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(1001, nil)
		traffic.EXPECT().DeleteFilter(mock.Anything, mock.Anything).Return(nil).Maybe()
		ns := netns.NewManagerMock(GinkgoT())
		ns.EXPECT().Enter().Return(nil)
		ns.EXPECT().Exit().Return(nil)
		inj := networkDisruptionInjector{config: NetworkDisruptionInjectorConfig{
			Config:            Config{Log: zap.NewNop().Sugar(), Netns: ns, K8sClient: client},
			TrafficController: traffic,
		}}
		percentage := 50
		watcher := serviceWatcher{
			watchedServiceSpec:   v1beta1.NetworkDisruptionServiceSpec{Name: "svc", Namespace: "default", Percentage: &percentage},
			labelServiceSelector: "app=svc",
			servicePorts:         []v1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP}},
		}
		interfaces := []string{"eth0"}
		createPod := func(n int, ip string) *v1.Pod {
			pod, err := client.CoreV1().Pods("default").Create(ctx, &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", n), Namespace: "default", Labels: map[string]string{"app": "svc"}},
				Status:     v1.PodStatus{PodIP: ip},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			return pod
		}
		first := createPod(1, "10.0.0.1")
		second := createPod(2, "10.0.0.2")
		var err error
		watcher.tcFiltersFromPodEndpoints, err = inj.handlePodEndpointsServiceFiltersOnKubernetesServiceChanges(watcher.watchedServiceSpec, nil, []v1.Pod{*first, *second}, watcher.servicePorts, interfaces, "1:4")
		Expect(err).NotTo(HaveOccurred())
		Expect(watcher.tcFiltersFromPodEndpoints).To(HaveLen(1))

		assertSelection := func() {
			pods, err := client.CoreV1().Pods("default").List(ctx, metav1.ListOptions{LabelSelector: watcher.labelServiceSelector})
			Expect(err).NotTo(HaveOccurred())
			all := []tcServiceFilter{}
			for _, pod := range pods.Items {
				if pod.Status.PodIP != "" {
					all = append(all, inj.buildServiceFiltersFromPod(pod, watcher.servicePorts)...)
				}
			}
			expected := []string{}
			for _, filter := range inj.selectServiceFiltersByPercentage(all, percentage, "svc", "default") {
				expected = append(expected, filter.service.String())
			}
			actual := []string{}
			for _, filter := range watcher.tcFiltersFromPodEndpoints {
				actual = append(actual, filter.service.String())
			}
			Expect(actual).To(ConsistOf(expected))
		}
		for n := 3; n <= 4; n++ {
			pod := createPod(n, fmt.Sprintf("10.0.0.%d", n))
			Expect(inj.handleKubernetesPodsChanges(watch.Event{Type: watch.Added, Object: pod}, &watcher, interfaces, "1:4")).To(Succeed())
		}
		Expect(watcher.tcFiltersFromPodEndpoints).To(HaveLen(2))
		assertSelection()

		pending := createPod(5, "")
		Expect(inj.handleKubernetesPodsChanges(watch.Event{Type: watch.Added, Object: pending}, &watcher, interfaces, "1:4")).To(Succeed())
		pending.Status.PodIP = "10.0.0.5"
		pending, err = client.CoreV1().Pods("default").Update(ctx, pending, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(inj.handleKubernetesPodsChanges(watch.Event{Type: watch.Modified, Object: pending}, &watcher, interfaces, "1:4")).To(Succeed())
		Expect(watcher.tcFiltersFromPodEndpoints).To(HaveLen(3))
		assertSelection()

		Expect(client.CoreV1().Pods("default").Delete(ctx, first.Name, metav1.DeleteOptions{})).To(Succeed())
		Expect(inj.handleKubernetesPodsChanges(watch.Event{Type: watch.Deleted, Object: first}, &watcher, interfaces, "1:4")).To(Succeed())
		Expect(watcher.tcFiltersFromPodEndpoints).To(HaveLen(2))
		assertSelection()
	})
})

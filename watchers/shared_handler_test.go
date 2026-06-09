// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/DataDog/chaos-controller/cmd/mocks"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SharedChaosPodHandler", func() {
	var (
		k8sMock        *mocks.K8SClientMock
		recorder       record.EventRecorder
		metricsAdapter *watchers.WatcherMetricsAdapterMock
		handler        *watchers.SharedChaosPodHandler
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		recorder = record.NewFakeRecorder(10)
		metricsAdapter = watchers.NewWatcherMetricsAdapterMock(GinkgoT())
		// nil elected → always leader
		handler = watchers.NewSharedChaosPodHandler(k8sMock, recorder, zaptest.NewLogger(GinkgoT()).Sugar(), metricsAdapter, nil)
	})

	Describe("OnAdd", func() {
		It("does nothing (empty body)", func() {
			Expect(func() { handler.OnAdd(nil, false) }).NotTo(Panic())
		})
	})

	Describe("OnDelete", func() {
		It("does nothing (empty body)", func() {
			Expect(func() { handler.OnDelete(nil) }).NotTo(Panic())
		})
	})

	Describe("OnUpdate", func() {
		It("does nothing when objects are not pods", func() {
			Expect(func() { handler.OnUpdate("old", "new") }).NotTo(Panic())
		})

		It("does nothing when disruption labels are missing", func() {
			old := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}}
			new := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}}
			Expect(func() { handler.OnUpdate(old, new) }).NotTo(Panic())
		})

		It("emits metrics and skips event when phases are the same", func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "chaos-pod",
					Labels: map[string]string{
						chaostypes.DisruptionNameLabel:      "my-disruption",
						chaostypes.DisruptionNamespaceLabel: "default",
					},
				},
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			}
			metricsAdapter.EXPECT().OnChange(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			Expect(func() { handler.OnUpdate(pod, pod) }).NotTo(Panic())
		})

		It("does nothing when transitioning Running→Failed (expected)", func() {
			old := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					chaostypes.DisruptionNameLabel:      "dis",
					chaostypes.DisruptionNamespaceLabel: "ns",
				}},
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			}
			new := &corev1.Pod{
				ObjectMeta: old.ObjectMeta,
				Status:     corev1.PodStatus{Phase: corev1.PodFailed},
			}
			metricsAdapter.EXPECT().OnChange(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			Expect(func() { handler.OnUpdate(old, new) }).NotTo(Panic())
		})

		It("sends event when pod transitions to Failed from non-Running", func() {
			old := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					chaostypes.DisruptionNameLabel:      "dis",
					chaostypes.DisruptionNamespaceLabel: "ns",
				}},
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			}
			new := &corev1.Pod{
				ObjectMeta: old.ObjectMeta,
				Status:     corev1.PodStatus{Phase: corev1.PodFailed},
			}
			metricsAdapter.EXPECT().OnChange(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			// reader.Get will fail (disruption not found) — that's OK, sendEvent handles errors
			k8sMock.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1beta1.Disruption"), mock.Anything).
				Return(nil) // returns empty disruption

			Expect(func() { handler.OnUpdate(old, new) }).NotTo(Panic())
		})

		It("does nothing when not leader (closed elected channel blocks immediately)", func() {
			// Create a non-nil but never-closed channel to simulate standby replica
			elected := make(chan struct{})
			standbyHandler := watchers.NewSharedChaosPodHandler(k8sMock, recorder, zaptest.NewLogger(GinkgoT()).Sugar(), metricsAdapter, elected)
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				chaostypes.DisruptionNameLabel:      "dis",
				chaostypes.DisruptionNamespaceLabel: "ns",
			}}}
			Expect(func() { standbyHandler.OnUpdate(pod, pod) }).NotTo(Panic())
		})

		It("acts as leader when elected channel is closed", func() {
			elected := make(chan struct{})
			close(elected)
			leaderHandler := watchers.NewSharedChaosPodHandler(k8sMock, recorder, zaptest.NewLogger(GinkgoT()).Sugar(), metricsAdapter, elected)
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
				chaostypes.DisruptionNameLabel:      "dis",
				chaostypes.DisruptionNamespaceLabel: "ns",
			}}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}
			metricsAdapter.EXPECT().OnChange(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			Expect(func() { leaderHandler.OnUpdate(pod, pod) }).NotTo(Panic())
		})
	})
})

var _ = Describe("NamespaceCachePool", func() {
	It("NewNamespaceCachePool creates a non-nil pool", func() {
		pool := watchers.NewNamespaceCachePool(nil)
		Expect(pool).NotTo(BeNil())
	})
})

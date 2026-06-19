// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package targetselector_test

import (
	"context"
	"errors"

	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/targetselector"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetMatchingPodsOverTotalPods Filter annotations", func() {
	var (
		k8sMock      *mocks.K8SClientMock
		disruption   *chaosv1beta1.Disruption
		ts           targetselector.TargetSelector
		annotatedPod corev1.Pod
		plainPod     corev1.Pod
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		ts = targetselector.NewRunningTargetSelector(false, "ctrl-node")
		disruption = &chaosv1beta1.Disruption{
			Spec: chaosv1beta1.DisruptionSpec{
				Selector: map[string]string{"app": "test"},
				Filter: &chaosv1beta1.DisruptionFilter{
					Annotations: map[string]string{"env": "prod"},
				},
			},
		}
		annotatedPod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "annotated-pod",
				Annotations: map[string]string{"env": "prod"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		}
		plainPod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "plain-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
	})

	It("includes pods with matching annotation", func() {
		k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
			Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				list.(*corev1.PodList).Items = []corev1.Pod{annotatedPod, plainPod}
			}).Return(nil)

		r, _, err := ts.GetMatchingPodsOverTotalPods(k8sMock, disruption)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Items).To(HaveLen(1))
		Expect(r.Items[0].Name).To(Equal("annotated-pod"))
	})
})

var _ = Describe("GetMatchingNodesOverTotalNodes Filter annotations", func() {
	var (
		k8sMock       *mocks.K8SClientMock
		disruption    *chaosv1beta1.Disruption
		ts            targetselector.TargetSelector
		annotatedNode corev1.Node
		plainNode     corev1.Node
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		ts = targetselector.NewRunningTargetSelector(false, "ctrl-node")
		disruption = &chaosv1beta1.Disruption{
			Spec: chaosv1beta1.DisruptionSpec{
				Selector: map[string]string{"app": "test"},
				Filter: &chaosv1beta1.DisruptionFilter{
					Annotations: map[string]string{"zone": "us-east"},
				},
			},
		}
		annotatedNode = corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "annotated-node",
				Annotations: map[string]string{"zone": "us-east"},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		}
		plainNode = corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "plain-node"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		}
	})

	It("includes nodes with matching annotation", func() {
		k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.NodeList"), mock.Anything).
			Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				list.(*corev1.NodeList).Items = []corev1.Node{annotatedNode, plainNode}
			}).Return(nil)

		r, _, err := ts.GetMatchingNodesOverTotalNodes(k8sMock, disruption)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Items).To(HaveLen(1))
		Expect(r.Items[0].Name).To(Equal("annotated-node"))
	})
})

var _ = Describe("GetMatchingPodsOverTotalPods safeguard with NodeFailure spec", func() {
	var (
		k8sMock    *mocks.K8SClientMock
		disruption *chaosv1beta1.Disruption
		ts         targetselector.TargetSelector
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		ts = targetselector.NewRunningTargetSelector(true, "ctrl-node")
		disruption = &chaosv1beta1.Disruption{
			Spec: chaosv1beta1.DisruptionSpec{
				Selector:    map[string]string{"app": "test"},
				NodeFailure: &chaosv1beta1.NodeFailureSpec{},
			},
		}
	})

	It("skips pod on controller node when NodeFailure spec and safeguards enabled", func() {
		ctrlNodePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "ctrl-pod"},
			Spec:       corev1.PodSpec{NodeName: "ctrl-node"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		otherPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "other-pod"},
			Spec:       corev1.PodSpec{NodeName: "other-node"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}

		k8sMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
			Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
				list.(*corev1.PodList).Items = []corev1.Pod{ctrlNodePod, otherPod}
			}).Return(nil)

		r, _, err := ts.GetMatchingPodsOverTotalPods(k8sMock, disruption)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Items).To(HaveLen(1))
		Expect(r.Items[0].Name).To(Equal("other-pod"))
	})
})

var _ = Describe("TargetIsHealthy pod level with NodeFailure node lookup", func() {
	var (
		k8sMock    *mocks.K8SClientMock
		disruption *chaosv1beta1.Disruption
		ts         targetselector.TargetSelector
	)

	BeforeEach(func() {
		k8sMock = mocks.NewK8SClientMock(GinkgoT())
		ts = targetselector.NewRunningTargetSelector(false, "ctrl")
		disruption = &chaosv1beta1.Disruption{
			Spec: chaosv1beta1.DisruptionSpec{
				Level:       chaostypes.DisruptionLevelPod,
				Selector:    map[string]string{"app": "test"},
				NodeFailure: &chaosv1beta1.NodeFailureSpec{},
			},
		}
	})

	It("returns error when pod node lookup fails (NodeFailure spec)", func() {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "missing-node"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		k8sMock.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Pod"), mock.Anything).
			Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
				*obj.(*corev1.Pod) = pod
			}).Return(nil)
		k8sMock.EXPECT().Get(mock.Anything, mock.Anything, mock.AnythingOfType("*v1.Node"), mock.Anything).
			Return(errors.New("node not found"))

		err := ts.TargetIsHealthy("test-pod", k8sMock, disruption)
		Expect(err).To(HaveOccurred())
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cache_test

import (
	chaoscache "github.com/DataDog/chaos-controller/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PodTransformer", func() {
	var input *corev1.Pod

	BeforeEach(func() {
		input = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Namespace:   "test-ns",
				Labels:      map[string]string{"app": "test"},
				Annotations: map[string]string{"anno": "value"},
				ManagedFields: []metav1.ManagedFieldsEntry{
					{Manager: "kubectl"},
				},
			},
			Spec: corev1.PodSpec{
				NodeName:   "node-1",
				Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
				Volumes:    []corev1.Volume{{Name: "data"}},
			},
			Status: corev1.PodStatus{
				Phase:                 corev1.PodRunning,
				PodIP:                 "10.0.0.1",
				Reason:                "some-reason",
				Message:               "should be stripped",
				Conditions:            []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
				ContainerStatuses:     []corev1.ContainerStatus{{Name: "app", Ready: true}},
				InitContainerStatuses: []corev1.ContainerStatus{{Name: "init", Ready: true}},
			},
		}
	})

	It("retains fields required by the controller", func() {
		result, err := chaoscache.PodTransformer(input)
		Expect(err).NotTo(HaveOccurred())

		pod := result.(*corev1.Pod)
		Expect(pod.Name).To(Equal("test-pod"))
		Expect(pod.Namespace).To(Equal("test-ns"))
		Expect(pod.Labels).To(Equal(map[string]string{"app": "test"}))
		Expect(pod.Annotations).To(Equal(map[string]string{"anno": "value"}))
		Expect(pod.Spec.NodeName).To(Equal("node-1"))
		Expect(pod.Spec.Containers).To(HaveLen(1))
		Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
		Expect(pod.Status.PodIP).To(Equal("10.0.0.1"))
		Expect(pod.Status.Reason).To(Equal("some-reason"))
		Expect(pod.Status.Conditions).To(HaveLen(1))
		Expect(pod.Status.Conditions[0].Type).To(Equal(corev1.PodReady))
		Expect(pod.Status.ContainerStatuses).To(HaveLen(1))
		Expect(pod.Status.InitContainerStatuses).To(HaveLen(1))
	})

	It("strips fields not required by the controller", func() {
		result, err := chaoscache.PodTransformer(input)
		Expect(err).NotTo(HaveOccurred())

		pod := result.(*corev1.Pod)
		Expect(pod.Spec.Volumes).To(BeEmpty())
		Expect(pod.Status.Message).To(BeEmpty())
		Expect(pod.ManagedFields).To(BeEmpty())
	})

	It("passes non-pod objects through unchanged", func() {
		other := "not-a-pod"
		result, err := chaoscache.PodTransformer(other)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(other))
	})
})

var _ = Describe("NodeTransformer", func() {
	var input *corev1.Node

	BeforeEach(func() {
		input = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-node",
				Labels:      map[string]string{"role": "worker"},
				Annotations: map[string]string{"anno": "value"},
				ManagedFields: []metav1.ManagedFieldsEntry{
					{Manager: "kubectl"},
				},
			},
			Spec: corev1.NodeSpec{
				PodCIDR: "10.0.0.0/24",
			},
			Status: corev1.NodeStatus{
				Phase:      corev1.NodeRunning,
				Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
				Images:     []corev1.ContainerImage{{Names: []string{"nginx:latest"}}},
				NodeInfo:   corev1.NodeSystemInfo{OSImage: "Ubuntu 22.04"},
			},
		}
	})

	It("retains fields required by the controller", func() {
		result, err := chaoscache.NodeTransformer(input)
		Expect(err).NotTo(HaveOccurred())

		node := result.(*corev1.Node)
		Expect(node.Name).To(Equal("test-node"))
		Expect(node.Labels).To(Equal(map[string]string{"role": "worker"}))
		Expect(node.Annotations).To(Equal(map[string]string{"anno": "value"}))
		Expect(node.Status.Phase).To(Equal(corev1.NodeRunning))
		Expect(node.Status.Conditions).To(HaveLen(1))
		Expect(node.Status.Conditions[0].Type).To(Equal(corev1.NodeReady))
	})

	It("strips fields not required by the controller", func() {
		result, err := chaoscache.NodeTransformer(input)
		Expect(err).NotTo(HaveOccurred())

		node := result.(*corev1.Node)
		Expect(node.Spec.PodCIDR).To(BeEmpty())
		Expect(node.Status.Images).To(BeEmpty())
		Expect(node.Status.NodeInfo).To(Equal(corev1.NodeSystemInfo{}))
		Expect(node.ManagedFields).To(BeEmpty())
	})

	It("passes non-node objects through unchanged", func() {
		other := "not-a-node"
		result, err := chaoscache.NodeTransformer(other)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(other))
	})
})

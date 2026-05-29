// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cache

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodTransformer strips fields from Pod objects before they are stored in the
// informer cache. On large clusters, the cache holds tens of thousands of pods; retaining
// the full object (volumes, affinity, scheduling fields, etc.) wastes significant memory.
// Only fields actually read by the controller, target selector, and watchers are kept.
func PodTransformer(i interface{}) (interface{}, error) {
	pod, ok := i.(*corev1.Pod)
	if !ok {
		return i, nil
	}

	return &corev1.Pod{
		TypeMeta:   pod.TypeMeta,
		ObjectMeta: withoutManagedFields(pod.ObjectMeta),
		Spec: corev1.PodSpec{
			NodeName:   pod.Spec.NodeName,   // used by safeguards and chaos pod scheduling
			Containers: pod.Spec.Containers, // read by the chaos pod service for injector args
		},
		Status: corev1.PodStatus{
			Phase:                 pod.Status.Phase,                 // used to filter running/pending pods
			PodIP:                 pod.Status.PodIP,                 // passed to chaos pod generation
			Reason:                pod.Status.Reason,                // monitored by chaos pod watcher
			Conditions:            pod.Status.Conditions,            // PodReady condition checked for chaos pod injection state
			ContainerStatuses:     pod.Status.ContainerStatuses,     // monitored for container restarts
			InitContainerStatuses: pod.Status.InitContainerStatuses, // used for OnInit disruptions
		},
	}, nil
}

// NodeTransformer strips fields from Node objects before they are stored in the
// informer cache. Status.Images alone can list hundreds of container images per node
// and is never read by the controller; stripping it significantly reduces memory usage.
func NodeTransformer(i interface{}) (interface{}, error) {
	node, ok := i.(*corev1.Node)
	if !ok {
		return i, nil
	}

	return &corev1.Node{
		TypeMeta:   node.TypeMeta,
		ObjectMeta: withoutManagedFields(node.ObjectMeta),
		Status: corev1.NodeStatus{
			Conditions: node.Status.Conditions, // used to check node readiness
			Phase:      node.Status.Phase,      // monitored by node watcher
		},
	}, nil
}

// withoutManagedFields returns a shallow copy of om with ManagedFields cleared.
func withoutManagedFields(om metav1.ObjectMeta) metav1.ObjectMeta {
	om.ManagedFields = nil
	return om
}

package controllers

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockTargetSelector finds pods in Running Phase for applying network disruptions to a Kubernetes Cluster
type MockTargetSelector struct{}

// GetMatchingPods returns candidate pods for this disruption given a namespace and label selector
func (m MockTargetSelector) GetMatchingPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, error) {
	pods := &corev1.PodList{}

	listOptions := &client.ListOptions{
		LabelSelector: instance.Spec.Selector.AsSelector(),
		Namespace:     instance.Namespace,
	}

	err := c.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// GetMatchingNodes returns the still-existing nodes that were targeted by the disruption,
func (m MockTargetSelector) GetMatchingNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	listOptions := &client.ListOptions{
		LabelSelector: instance.Spec.Selector.AsSelector(),
	}

	err := c.List(context.Background(), nodes, listOptions)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// TargetIsHealthy returns true if the given target exists, false otherwise
func (m MockTargetSelector) TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error {
	return nil
}

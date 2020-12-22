// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockTargetSelector finds pods in Running Phase for applying network disruptions to a Kubernetes Cluster
type MockTargetSelector struct{}

// GetMatchingPods returns candidate pods for this disruption given a namespace and label selector.
// For mocking purposes, pod statuses are not checked.
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

// GetMatchingNodes returns the still-existing nodes that were targeted by the disruption.
// For mocking purposes, node statuses are not checked.
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

// TargetIsHealthy returns an error if the given target is unhealthy or does not exist.
// For mocking purposes, there is never an error.
func (m MockTargetSelector) TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error {
	return nil
}

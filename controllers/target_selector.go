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
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/helpers"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TargetSelector is an interface for applying network disruptions to a Kubernetes Cluster
type TargetSelector interface {
	GetMatchingPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, error)
	GetMatchingNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, error)
	TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error
}

// RunningTargetSelector finds pods in Running Phase for applying network disruptions to a Kubernetes Cluster
type RunningTargetSelector struct{}

// GetMatchingPods returns candidate pods for this disruption given a namespace and label selector
func (r RunningTargetSelector) GetMatchingPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, error) {
	return helpers.GetMatchingPods(c, instance.Namespace, instance.Spec.Selector)
}

// GetMatchingNodes returns candidate nodes for this disruption given a label selector
func (r RunningTargetSelector) GetMatchingNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, error) {
	return helpers.GetMatchingNodes(c, instance.Spec.Selector)
}

// TargetIsHealthy returns true if the given target exists, false otherwise
func (r RunningTargetSelector) TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error {
	return helpers.TargetIsHealthy(target, c, instance.Spec.Level, instance.Namespace)
}

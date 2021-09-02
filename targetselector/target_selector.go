// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package targetselector

import (
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TargetSelector is an interface for applying network disruptions to a Kubernetes Cluster
type TargetSelector interface {
	GetMatchingPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, error)
	GetMatchingNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, error)
	TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error
}

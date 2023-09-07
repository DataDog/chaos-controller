// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package targetselector

import (
	"fmt"
	"regexp"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TargetSelector is an interface for applying network disruptions to a Kubernetes Cluster
type TargetSelector interface {
	// GetMatchingPodsOverTotalPods Returns list of matching ready and untargeted pods and number of total pods
	GetMatchingPodsOverTotalPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, int, error)

	// GetMatchingNodesOverTotalNodes Returns list of matching ready and untargeted nodes and number of total nodes
	GetMatchingNodesOverTotalNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, int, error)

	// TargetIsHealthy Returns an error if the given target is unhealthy or does not exist
	TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error
}

// ValidateLabelSelector assert label selector matches valid grammar, avoids CORE-414
func ValidateLabelSelector(selector labels.Selector) error {
	labelGrammar := "([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]"
	rgx := regexp.MustCompile(labelGrammar)

	if !rgx.MatchString(selector.String()) {
		return fmt.Errorf("given label selector is invalid, it does not match valid selector grammar: %s %s", selector.String(), labelGrammar)
	}

	return nil
}

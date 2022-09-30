// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	"context"
	"fmt"

	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetChaosPods returns chaos pods owned by the given instance and having the given labels
// both instance and label set are optional but at least one must be provided
func GetChaosPods(ctx context.Context, log *zap.SugaredLogger, chaosNamespace string, k8sClient client.Client, instance *Disruption, ls labels.Set) ([]corev1.Pod, error) {
	pods := &corev1.PodList{}

	if k8sClient == nil {
		return nil, fmt.Errorf("you must provide a non nil Kubernetes client")
	}

	// ensure we always have at least a disruption instance or a label set to filter on
	if instance == nil && ls == nil {
		return nil, fmt.Errorf("you must specify at least a disruption instance or a label set to get chaos pods")
	}

	if ls == nil {
		ls = make(map[string]string)
	}

	// add instance specific labels if provided
	if instance != nil {
		ls[chaostypes.DisruptionNameLabel] = instance.Name
		ls[chaostypes.DisruptionNamespaceLabel] = instance.Namespace
	}

	// list pods in the defined namespace and for the given target
	listOptions := &client.ListOptions{
		Namespace:     chaosNamespace,
		LabelSelector: labels.SelectorFromValidatedSet(ls),
	}

	err := k8sClient.List(ctx, pods, listOptions)
	if err != nil {
		return nil, fmt.Errorf("error listing owned pods: %w", err)
	}

	podNames := []string{}
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	if log != nil {
		log.Debugw("searching for chaos pods with label selector...", "labels", ls.String(), "foundPods", podNames)
	}

	return pods.Items, nil
}

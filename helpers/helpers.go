// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package helpers

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ChaosFailureInjectionImageVariableName is the name of the chaos failure injection image variable
	ChaosFailureInjectionImageVariableName = "CHAOS_INJECTOR_IMAGE"
)

// GetMatchingPods returns a pods list containing all pods matching the given label selector and namespace
func GetMatchingPods(c client.Client, namespace string, selector labels.Set) (*corev1.PodList, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if len(selector) < 1 || selector == nil {
		return nil, errors.New("selector can't be an empty set")
	}

	// filter pods based on the label selector and namespace
	pods := &corev1.PodList{}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     namespace,
	}

	// fetch pods from label selector
	err := c.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, err
	}

	runningPods := &corev1.PodList{}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods.Items = append(runningPods.Items, pod)
		}
	}

	return runningPods, nil
}

// GetMatchingNodes returns a nodes list containing all nodes matching the given label selector
func GetMatchingNodes(c client.Client, selector labels.Set) (*corev1.NodeList, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if len(selector) < 1 || selector == nil {
		return nil, errors.New("selector can't be an empty set")
	}

	// filter nodes based on the label selector
	nodes := &corev1.NodeList{}
	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
	}

	// fetch nodes from label selector
	err := c.List(context.Background(), nodes, listOptions)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// GetOwnedPods returns a list of pods owned by the given object
func GetOwnedPods(c client.Client, owner metav1.Object, selector labels.Set) (corev1.PodList, error) {
	// prepare list options
	options := &client.ListOptions{Namespace: owner.GetNamespace()}
	if selector != nil {
		options.LabelSelector = selector.AsSelector()
	}

	// get pods
	pods := corev1.PodList{}
	ownedPods := corev1.PodList{}

	err := c.List(context.Background(), &pods, options)
	if err != nil {
		return ownedPods, err
	}

	// check owner reference
	for _, pod := range pods.Items {
		if metav1.IsControlledBy(&pod, owner) {
			ownedPods.Items = append(ownedPods.Items, pod)
		}
	}

	return ownedPods, nil
}

// ContainsString returns true if the given slice contains the given string,
// or returns false otherwise
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}

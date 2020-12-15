// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package helpers

import (
	"context"
	"errors"
	"math/rand"
	"time"

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

	// filter pods based on the nfi's label selector, and only consider those within the same namespace as the nfi
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

// PickRandomPods returns a shuffled sub-slice with a size of n of the given slice
func PickRandomPods(n uint, pods []corev1.Pod) []corev1.Pod {
	rand.Seed(time.Now().Unix())

	// copy slice to don't modify the given one
	list := append([]corev1.Pod(nil), pods...)

	// shuffle the slice
	for i := len(list) - 1; i > 0; i-- {
		j := rand.Intn(i)
		list[i], list[j] = list[j], list[i]
	}

	// return the whole shuffled slice if the requested size is greater than the size of the slice
	if int(n) > len(list) {
		return list
	}

	return list[:n]
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

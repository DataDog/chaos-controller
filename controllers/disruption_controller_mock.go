// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NetworkConfigMock is a mock implementation of the NetworkConfig interface
type DisruptionReconcilerMock struct {
	mock.Mock
}

func (r DisruptionReconcilerMock) getMatchingPods(c client.Client, namespace string, selector labels.Set) (*corev1.PodList, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if len(selector) < 1 || selector == nil {
		return nil, errors.New("selector can't be an empty set")
	}

	// filter pods based on the nfi's label selector, and only consider those within the same namespace as the nfi
	pods := &corev1.PodList{}

	listOptions := &client.ListOptions{
		LabelSelector: selector.AsSelector(),
		Namespace:     namespace,
		//FieldSelector: fields.Set{"status.phase": "Running"}.AsSelector(),
	}

	// fetch pods from label selector
	err := c.List(context.Background(), pods, listOptions)
	if err != nil {
		return nil, err
	}

	return pods
}

// getPodsToCleanup returns the still-existing pods that were targeted by the disruption, according to the pod names in the instance status
func (r DisruptionReconcilerMock) getPodsToCleanup(instance *chaosv1beta1.Disruption) ([]*corev1.Pod, error) {
	podsToCleanup := make([]*corev1.Pod, 0, len(instance.Status.TargetPods))

	// check if each pod still exists; skip if it doesn't
	for _, podName := range instance.Status.TargetPods {
		// get the targeted pods names from the status
		podKey := types.NamespacedName{Name: podName, Namespace: instance.Namespace}
		p := &corev1.Pod{}
		err := r.Get(context.Background(), podKey, p)

		// skip if the pod doesn't exist anymore
		if errors.IsNotFound(err) {
			r.Log.Info("cleanup: pod no longer exists (skip)", "instance", instance.Name, "namespace", instance.Namespace, "name", podName)
			continue
		} else if err != nil {
			return nil, err
		}

		podsToCleanup = append(podsToCleanup, p)
	}

	return podsToCleanup, nil
}

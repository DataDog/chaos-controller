// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package targetselector

import (
	"context"
	"errors"
	"fmt"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RunningTargetSelector finds pods in Running Phase for applying network disruptions to a Kubernetes Cluster
type RunningTargetSelector struct{}

// GetMatchingPods returns a pods list containing all running pods matching the given label selector and namespace
func (r RunningTargetSelector) GetMatchingPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, error) {
	// get parsed selector
	selector, err := getLabelSelectorFromInstance(instance)
	if err != nil {
		return nil, fmt.Errorf("error getting label selector from disruption: %w", err)
	}

	// filter pods based on the label selector and namespace
	pods := &corev1.PodList{}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     instance.Namespace,
	}

	// fetch pods from label selector
	if err := c.List(context.Background(), pods, listOptions); err != nil {
		return nil, err
	}

	runningPods := &corev1.PodList{}

	for _, pod := range pods.Items {
		// if the disruption is applied on init, we only target pending pods with a running
		// chaos handler init container
		// otherwise, we only target running pods
		if instance.Spec.OnInit {
			hasChaosHandler := false

			// search for a potential running chaos handler init container
			for _, initContainerStatus := range pod.Status.InitContainerStatuses {
				if initContainerStatus.Name == "chaos-handler" && initContainerStatus.State.Running != nil {
					hasChaosHandler = true

					break
				}
			}

			if pod.Status.Phase == corev1.PodPending && hasChaosHandler {
				runningPods.Items = append(runningPods.Items, pod)
			}
		} else if pod.Status.Phase == corev1.PodRunning {
			runningPods.Items = append(runningPods.Items, pod)
		}
	}

	return runningPods, nil
}

// GetMatchingNodes returns a nodes list containing all nodes matching the given label selector
func (r RunningTargetSelector) GetMatchingNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, error) {
	// get parsed selector
	selector, err := getLabelSelectorFromInstance(instance)
	if err != nil {
		return nil, fmt.Errorf("error getting label selector from disruption: %w", err)
	}

	// filter nodes based on the label selector
	nodes := &corev1.NodeList{}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
	}

	// fetch nodes from label selector
	if err := c.List(context.Background(), nodes, listOptions); err != nil {
		return nil, err
	}

	runningNodes := &corev1.NodeList{}

	for _, node := range nodes.Items {
		// check if node is ready
		ready := false

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}

		if ready {
			runningNodes.Items = append(runningNodes.Items, node)
		}
	}

	return runningNodes, nil
}

// TargetIsHealthy returns an error if the given target is unhealthy or does not exist
func (r RunningTargetSelector) TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error {
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		var p corev1.Pod

		// check if target still exists
		if err := c.Get(context.Background(), types.NamespacedName{Name: target, Namespace: instance.Namespace}, &p); err != nil {
			return err
		}

		// check if pod is running
		if p.Status.Phase != corev1.PodRunning {
			return errors.New("pod is not Running")
		}
	case chaostypes.DisruptionLevelNode:
		var n corev1.Node
		if err := c.Get(context.Background(), client.ObjectKey{Name: target}, &n); err != nil {
			return err
		}

		// check if node is ready
		ready := false

		for _, condition := range n.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}

		if !ready {
			return errors.New("node is not Ready")
		}
	}

	return nil
}

// getLabelSelectorFromInstance crafts a label selector made of requirements from the given disruption instance
func getLabelSelectorFromInstance(instance *chaosv1beta1.Disruption) (labels.Selector, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if (len(instance.Spec.Selector) == 0 || instance.Spec.Selector == nil) && (len(instance.Spec.AdvancedSelector) == 0 || instance.Spec.AdvancedSelector == nil) {
		return nil, errors.New("selector can't be an empty set")
	}

	selector := labels.NewSelector()

	// add simple selectors by parsing them
	if instance.Spec.Selector != nil {
		req, err := labels.ParseToRequirements(instance.Spec.Selector.AsSelector().String())
		if err != nil {
			return nil, fmt.Errorf("error parsing given selector to requirements: %w", err)
		}

		selector = selector.Add(req...)
	}

	// add advanced selectors
	if instance.Spec.AdvancedSelector != nil {
		for _, req := range instance.Spec.AdvancedSelector {
			var op selection.Operator

			// parse the operator to convert it from one package to another
			switch req.Operator {
			case metav1.LabelSelectorOpIn:
				op = selection.In
			case metav1.LabelSelectorOpNotIn:
				op = selection.NotIn
			case metav1.LabelSelectorOpExists:
				op = selection.Exists
			case metav1.LabelSelectorOpDoesNotExist:
				op = selection.DoesNotExist
			default:
				return nil, fmt.Errorf("error parsing advanced selector operator %s: must be either In, NotIn, Exists or DoesNotExist", req.Operator)
			}

			// generate and add the requirement to the selector
			parsedReq, err := labels.NewRequirement(req.Key, op, req.Values)
			if err != nil {
				return nil, fmt.Errorf("error parsing given advanced selector to requirements: %w", err)
			}

			selector = selector.Add(*parsedReq)
		}
	}

	// if the disruption is supposed to be injected on pod init
	// then let's add a requirement to get pods having the matching label only
	if instance.Spec.OnInit {
		onInitRequirement, err := labels.NewRequirement(chaostypes.DisruptOnInitLabel, selection.Exists, []string{})
		if err != nil {
			return nil, fmt.Errorf("error adding the disrupt-on-init label requirement: %w", err)
		}

		selector.Add(*onInitRequirement)
	}

	return selector, nil
}

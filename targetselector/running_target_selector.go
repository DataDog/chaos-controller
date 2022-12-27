// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package targetselector

import (
	"context"
	"errors"
	"fmt"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// runningTargetSelector finds pods in Running Phase for applying network disruptions to a Kubernetes Cluster
type runningTargetSelector struct {
	controllerEnableSafeguards bool
	controllerNodeName         string
}

func NewRunningTargetSelector(controllerEnableSafeguards bool, controllerNodeName string) TargetSelector {
	return runningTargetSelector{
		controllerEnableSafeguards: controllerEnableSafeguards,
		controllerNodeName:         controllerNodeName,
	}
}

// GetMatchingPodsOverTotalPods returns a pods list containing all running pods matching the given label selector and namespace and the count of pods matching the selector
func (r runningTargetSelector) GetMatchingPodsOverTotalPods(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.PodList, int, error) {
	// get parsed selector
	selector, secondarySelector, err := GetLabelSelectorsFromInstance(instance)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting label selector from disruption: %w", err)
	}

	// filter pods based on the label selector and namespace
	pods := &corev1.PodList{}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
		Namespace:     instance.Namespace,
	}

	// fetch pods from label selector
	if err := c.List(context.Background(), pods, listOptions); err != nil {
		return nil, 0, err
	}

	runningPods := &corev1.PodList{}

	for _, pod := range pods.Items {
		// check the pod is already a disruption target
		isAlreadyATarget := false

		for target := range instance.Status.TargetInjections {
			if target == pod.Name {
				isAlreadyATarget = true

				break
			}
		}

		// apply controller safeguards if enabled
		if r.controllerEnableSafeguards {
			// skip the node running the controller if the disruption has a node failure in its spec
			if instance.Spec.NodeFailure != nil && pod.Spec.NodeName == r.controllerNodeName {
				continue
			}
		}

		// if the disruption is applied on init, we only target pending pods with a running (or terminated)
		// chaos handler init container
		// otherwise, we only target running pods
		if instance.Spec.OnInit {
			hasChaosHandler := false

			// search for a potential running chaos handler init container
			for _, initContainerStatus := range pod.Status.InitContainerStatuses {
				// If the container is the on init container named chaos handler and either in
				// - a Running state, blocking the execution of the target
				// - a Completed state, but already was targeted before and is being reevaluated because of dynamic targeting so we shouldn't remove the pod from the list of targets
				if initContainerStatus.Name == "chaos-handler" && (initContainerStatus.State.Running != nil || (isAlreadyATarget && initContainerStatus.State.Terminated != nil && initContainerStatus.State.Terminated.Reason == "Completed")) {
					hasChaosHandler = true

					break
				}
			}

			if hasChaosHandler && (pod.Status.Phase == corev1.PodPending || (pod.Status.Phase == corev1.PodRunning && isAlreadyATarget)) {
				runningPods.Items = append(runningPods.Items, pod)
			}
		} else if pod.Status.Phase == corev1.PodRunning {
			runningPods.Items = append(runningPods.Items, pod)
		}
	}

	if secondarySelector != nil {
		prunedRunningPods := &corev1.PodList{}
		nodes := &corev1.NodeList{}
		nodeNames := []string{}

		listOptions = &client.ListOptions{
			LabelSelector: secondarySelector,
		}

		// fetch nodes from label selector
		if err = c.List(context.Background(), nodes, listOptions); err != nil {
			return nil, 0, err
		}

		for _, node := range nodes.Items {
			nodeNames = append(nodeNames, node.Name)
		}

		for _, pod := range runningPods.Items {
			if utils.Contains(nodeNames, pod.Spec.NodeName) {
				prunedRunningPods.Items = append(prunedRunningPods.Items, pod)
			}
		}

		runningPods = prunedRunningPods
	}

	return runningPods, len(pods.Items), nil
}

// GetMatchingNodesOverTotalNodes returns a nodes list containing all nodes matching the given label selector and the count of nodes matching the selector
func (r runningTargetSelector) GetMatchingNodesOverTotalNodes(c client.Client, instance *chaosv1beta1.Disruption) (*corev1.NodeList, int, error) {
	// get parsed selector
	selector, secondarySelector, err := GetLabelSelectorsFromInstance(instance)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting label selector from disruption: %w", err)
	}

	// filter nodes based on the label selector
	nodes := &corev1.NodeList{}
	listOptions := &client.ListOptions{
		LabelSelector: selector,
	}

	// fetch nodes from label selector
	if err := c.List(context.Background(), nodes, listOptions); err != nil {
		return nil, 0, err
	}

	runningNodes := &corev1.NodeList{}

	for _, node := range nodes.Items {
		// apply controller safeguards if enabled
		if r.controllerEnableSafeguards {
			// skip the node running the controller
			if node.Name == r.controllerNodeName {
				continue
			}
		}

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

	if secondarySelector != nil {
		prunedRunningNodes := &corev1.NodeList{}
		pods := &corev1.PodList{}
		podNodeNames := []string{}

		listOptions = &client.ListOptions{
			LabelSelector: secondarySelector,
		}

		// fetch nodes from label selector
		if err = c.List(context.Background(), pods, listOptions); err != nil {
			return nil, 0, err
		}

		for _, pod := range pods.Items {
			podNodeNames = append(podNodeNames, pod.Spec.NodeName)
		}

		for _, node := range runningNodes.Items {
			if utils.Contains(podNodeNames, node.Name) {
				prunedRunningNodes.Items = append(prunedRunningNodes.Items, node)
			}
		}

		runningNodes = prunedRunningNodes
	}

	return runningNodes, len(nodes.Items), nil
}

// TargetIsHealthy returns an error if the given target is unhealthy or does not exist
func (r runningTargetSelector) TargetIsHealthy(target string, c client.Client, instance *chaosv1beta1.Disruption) error {
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

		// check if pod's node is gone in the case that this was a node failure
		if instance.Spec.NodeFailure != nil {
			var n corev1.Node
			if err := c.Get(context.Background(), client.ObjectKey{Name: p.Spec.NodeName}, &n); err != nil {
				return err
			}
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

// GetLabelSelectorsFromInstance crafts a label selector made of requirements from the given disruption instance
func GetLabelSelectorsFromInstance(instance *chaosv1beta1.Disruption) (labels.Selector, labels.Selector, error) {
	// we want to ensure we never run into the possibility of using an empty label selector
	if (len(instance.Spec.Selector) == 0 || instance.Spec.Selector == nil) && (len(instance.Spec.AdvancedSelector) == 0 || instance.Spec.AdvancedSelector == nil) {
		return nil, nil, errors.New("selector can't be an empty set")
	}

	selector := labels.NewSelector()
	secondarySelector := labels.NewSelector()

	// add simple selectors by parsing them
	if instance.Spec.Selector != nil {
		req, err := labels.ParseToRequirements(instance.Spec.Selector.AsSelector().String())
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing given selector to requirements: %w", err)
		}

		selector = selector.Add(req...)
	}

	// add advanced selectors
	if instance.Spec.AdvancedSelector != nil {
		for _, req := range instance.Spec.AdvancedSelector {
			parsedReq, err := advancedSelectorToSelector(req)
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing given advanced selector to requirements: %w", err)
			}

			selector = selector.Add(*parsedReq)
		}
	}

	// if the disruption is supposed to be injected on pod init
	// then let's add a requirement to get pods having the matching label only
	if instance.Spec.OnInit {
		onInitRequirement, err := labels.NewRequirement(chaostypes.DisruptOnInitLabel, selection.Exists, []string{})
		if err != nil {
			return nil, nil, fmt.Errorf("error adding the disrupt-on-init label requirement: %w", err)
		}

		selector.Add(*onInitRequirement)
	}

	if instance.Spec.SecondarySelector != nil {
		for _, s := range instance.Spec.SecondarySelector {
			// generate and add the requirement to the selector
			parsedReq, err := advancedSelectorToSelector(s)
			if err != nil {
				return nil, nil, fmt.Errorf("error parsing given advanced selector to requirements: %w", err)
			}

			secondarySelector = secondarySelector.Add(*parsedReq)
		}
	}

	return selector, secondarySelector, nil
}

func advancedSelectorToSelector(req metav1.LabelSelectorRequirement) (*labels.Requirement, error) {
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
	return labels.NewRequirement(req.Key, op, req.Values)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// podReplacementInjector describes a pod replacement injector
type podReplacementInjector struct {
	spec         v1beta1.PodReplacementSpec
	config       PodReplacementInjectorConfig
	k8sClientset kubernetes.Interface
	nodeName     string
	cordoned     bool
}

// PodReplacementInjectorConfig contains needed drivers to
// create a PodReplacementInjector
type PodReplacementInjectorConfig struct {
	Config
}

// NewPodReplacementInjector creates a PodReplacementInjector object with the given config
func NewPodReplacementInjector(spec v1beta1.PodReplacementSpec, config PodReplacementInjectorConfig) (Injector, error) {
	if config.K8sClient == nil {
		return nil, fmt.Errorf("k8sClient is required")
	}

	if config.Disruption.TargetNodeName == "" {
		return nil, fmt.Errorf("target node name is required")
	}

	return &podReplacementInjector{
		spec:         spec,
		config:       config,
		k8sClientset: config.K8sClient,
		nodeName:     config.Disruption.TargetNodeName,
		cordoned:     false,
	}, nil
}

func (i *podReplacementInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *podReplacementInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindPodReplacement
}

// Inject performs the pod replacement by cordoning the node, deleting PVCs, and killing the target pod
func (i *podReplacementInjector) Inject() error {
	ctx := context.Background()

	i.config.Log.Infow("starting pod replacement injection",
		"nodeName", i.nodeName,
		"deleteStorage", i.spec.DeleteStorage,
		"forceDelete", i.spec.ForceDelete,
		"gracePeriodSeconds", i.spec.GracePeriodSeconds,
	)

	// Step 1: Cordon the node
	if err := i.cordonNode(ctx); err != nil {
		return fmt.Errorf("failed to cordon node %s: %w", i.nodeName, err)
	}

	// Step 2: Get the target pod and check if we've already processed it
	targetPod, err := i.getTargetPod(ctx)
	if err != nil {
		return fmt.Errorf("failed to get target pod: %w", err)
	}

	i.config.Log.Infow("found target pod", "nodeName", i.nodeName, "podName", targetPod.Name, "podNamespace", targetPod.Namespace, "podUID", targetPod.UID)

	// Step 3: Delete PVCs if requested
	if i.spec.DeleteStorage {
		if err := i.deletePVCs(ctx, []corev1.Pod{*targetPod}); err != nil {
			return fmt.Errorf("failed to delete PVCs: %w", err)
		}
	}

	// Step 4: Delete the target pod
	if err := i.deletePods(ctx, []corev1.Pod{*targetPod}); err != nil {
		return fmt.Errorf("failed to delete target pod: %w", err)
	}

	i.config.Log.Infow("pod replacement injection completed successfully",
		"nodeName", i.nodeName,
	)

	return nil
}

// cordonNode cordons the specified node to prevent new pods from being scheduled
func (i *podReplacementInjector) cordonNode(ctx context.Context) error {
	if i.config.Disruption.DryRun {
		i.config.Log.Infow("dry-run: would cordon node", "nodeName", i.nodeName)
		return nil
	}

	// Get the node
	node, err := i.k8sClientset.CoreV1().Nodes().Get(ctx, i.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", i.nodeName, err)
	}

	if node.Spec.Unschedulable {
		i.config.Log.Infow("node is already cordoned", "nodeName", i.nodeName)
		i.cordoned = true

		return nil
	}

	// Cordon the node
	node.Spec.Unschedulable = true

	_, err = i.k8sClientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node %s: %w", i.nodeName, err)
	}

	i.cordoned = true
	i.config.Log.Infow("successfully cordoned node", "nodeName", i.nodeName)

	return nil
}

// getTargetPod retrieves the specific target pod for this disruption
func (i *podReplacementInjector) getTargetPod(ctx context.Context) (*corev1.Pod, error) {
	// For pod replacement, we want to target the specific pod that was selected
	// We can identify it using the target pod IP from the disruption config
	if i.config.Disruption.TargetPodIP == "" {
		return nil, fmt.Errorf("target pod IP is required for pod replacement disruption")
	}

	// List pods on the target node and find the one with matching IP
	podList, err := i.k8sClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", i.nodeName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", i.nodeName, err)
	}

	// Find the target pod by IP
	for _, pod := range podList.Items {
		if pod.Status.PodIP == i.config.Disruption.TargetPodIP &&
			pod.DeletionTimestamp == nil &&
			pod.Status.Phase != corev1.PodSucceeded &&
			pod.Status.Phase != corev1.PodFailed {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf("target pod with IP %s not found on node %s", i.config.Disruption.TargetPodIP, i.nodeName)
}

// deletePVCs deletes all PVCs associated with the pods
func (i *podReplacementInjector) deletePVCs(ctx context.Context, pods []corev1.Pod) error {
	pvcNames := make(map[string]string) // PVC name -> namespace

	// Collect PVC references from all pods
	for _, pod := range pods {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				pvcName := volume.PersistentVolumeClaim.ClaimName
				pvcNames[pvcName] = pod.Namespace
				i.config.Log.Infow("found PVC to delete", "pvcName", pvcName, "namespace", pod.Namespace, "podName", pod.Name)
			}
		}
	}

	if len(pvcNames) == 0 {
		i.config.Log.Infow("no PVCs found to delete")
		return nil
	}

	// Delete each PVC
	for pvcName, namespace := range pvcNames {
		if i.config.Disruption.DryRun {
			i.config.Log.Infow("dry-run: would delete PVC", "pvcName", pvcName, "namespace", namespace)
			continue
		}

		err := i.k8sClientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			i.config.Log.Errorw("failed to delete PVC", "pvcName", pvcName, "namespace", namespace, "error", err)
			return fmt.Errorf("failed to delete PVC %s in namespace %s: %w", pvcName, namespace, err)
		}

		if err == nil {
			i.config.Log.Infow("successfully deleted PVC", "pvcName", pvcName, "namespace", namespace)
		} else {
			i.config.Log.Infow("PVC not found (already deleted)", "pvcName", pvcName, "namespace", namespace)
		}
	}

	return nil
}

// deletePods deletes all pods with appropriate grace period
func (i *podReplacementInjector) deletePods(ctx context.Context, pods []corev1.Pod) error {
	deleteOptions := metav1.DeleteOptions{}

	if i.spec.GracePeriodSeconds != nil {
		deleteOptions.GracePeriodSeconds = i.spec.GracePeriodSeconds
	}

	if i.spec.ForceDelete {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	for _, pod := range pods {
		if i.config.Disruption.DryRun {
			i.config.Log.Infow("dry-run: would delete pod",
				"podName", pod.Name,
				"namespace", pod.Namespace,
				"gracePeriodSeconds", deleteOptions.GracePeriodSeconds,
			)

			continue
		}

		err := i.k8sClientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOptions)
		if err != nil && !apierrors.IsNotFound(err) {
			i.config.Log.Errorw("failed to delete pod", "podName", pod.Name, "namespace", pod.Namespace, "error", err)
			return fmt.Errorf("failed to delete pod %s in namespace %s: %w", pod.Name, pod.Namespace, err)
		}

		if err == nil {
			i.config.Log.Infow("successfully deleted pod",
				"podName", pod.Name,
				"namespace", pod.Namespace,
				"gracePeriodSeconds", deleteOptions.GracePeriodSeconds,
			)
		} else {
			i.config.Log.Infow("pod not found (already deleted)", "podName", pod.Name, "namespace", pod.Namespace)
		}
	}

	return nil
}

// UpdateConfig updates the injector configuration
func (i *podReplacementInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

// Clean performs cleanup by uncordoning the node if we cordoned it
func (i *podReplacementInjector) Clean() error {
	if !i.cordoned {
		i.config.Log.Infow("node was not cordoned by this injector, skipping uncordon", "nodeName", i.nodeName)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("dry-run: would uncordon node", "nodeName", i.nodeName)
		return nil
	}

	// Get the node
	node, err := i.k8sClientset.CoreV1().Nodes().Get(ctx, i.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s for cleanup: %w", i.nodeName, err)
	}

	if !node.Spec.Unschedulable {
		i.config.Log.Infow("node is already uncordoned", "nodeName", i.nodeName)
		return nil
	}

	// Uncordon the node
	node.Spec.Unschedulable = false

	_, err = i.k8sClientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to uncordon node %s during cleanup: %w", i.nodeName, err)
	}

	i.config.Log.Infow("successfully uncordoned node during cleanup", "nodeName", i.nodeName)

	return nil
}

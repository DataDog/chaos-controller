// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers

import (
	context "context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentHandler struct {
	Client client.Client

	log *zap.SugaredLogger
}

func NewDeploymentHandler(client client.Client, logger *zap.SugaredLogger) DeploymentHandler {
	return DeploymentHandler{
		Client: client,
		log:    logger,
	}
}

// OnAdd is a handler function for the add of a deployment
func (h DeploymentHandler) OnAdd(obj interface{}) {
	deployment, ok := obj.(*appsv1.Deployment)

	// If the object is not a deployment, do nothing
	if !ok {
		return
	}

	// If deployment doesn't have associated disruption rollout, do nothing
	hasDisruptionRollout, err := h.HasAssociatedDisruptionRollout(deployment)
	if err != nil {
		return
	}

	if !hasDisruptionRollout {
		return
	}

	initContainersHash, containersHash, err := HashPodSpec(&deployment.Spec.Template.Spec)
	if err != nil {
		return
	}

	err = h.UpdateDisruptionRolloutStatus(deployment, initContainersHash, containersHash)
	if err != nil {
		return
	}
}

// OnUpdate is a handler function for the update of a deployment
func (h DeploymentHandler) OnUpdate(oldObj, newObj interface{}) {
	// Convert oldObj and newObj to Deployment objects
	oldDeployment, okOldDeployment := oldObj.(*appsv1.Deployment)
	newDeployment, okNewDeployment := newObj.(*appsv1.Deployment)

	// If both old and new are not deployments, do nothing
	if !okOldDeployment || !okNewDeployment {
		return
	}

	// If deployment doesn't have associated disruption rollout, do nothing
	hasDisruptionRollout, err := h.HasAssociatedDisruptionRollout(newDeployment)
	if !hasDisruptionRollout || err != nil {
		return
	}

	// If containers have't changed, do nothing
	containersChanged, initContainersHash, containersHash, err := ContainersChanged(&oldDeployment.Spec.Template.Spec, &newDeployment.Spec.Template.Spec, h.log)
	if !containersChanged || err != nil {
		return
	}

	err = h.UpdateDisruptionRolloutStatus(newDeployment, initContainersHash, containersHash)
	if err != nil {
		return
	}
}

// OnDelete is a handler function for the delete of a deployment
func (h DeploymentHandler) OnDelete(_ interface{}) {
	// Do nothing on delete event
}

func (h DeploymentHandler) FetchAssociatedDisruptionRollouts(deployment *appsv1.Deployment) (*chaosv1beta1.DisruptionRolloutList, error) {
	indexedValue := "deployment" + "-" + deployment.Namespace + "-" + deployment.Name

	disruptionRollouts := &chaosv1beta1.DisruptionRolloutList{}
	err := h.Client.List(context.Background(), disruptionRollouts, client.MatchingFields{"targetResource": indexedValue})

	if err != nil {
		h.log.Errorw("unable to fetch DisruptionRollouts using index", "error", err, "indexedValue", indexedValue)
		return nil, err
	}

	return disruptionRollouts, nil
}

func (h DeploymentHandler) HasAssociatedDisruptionRollout(deployment *appsv1.Deployment) (bool, error) {
	disruptionRollouts, err := h.FetchAssociatedDisruptionRollouts(deployment)
	if err != nil {
		h.log.Errorw("unable to check for associated DisruptionRollout", "Deployment", deployment.Name, "error", err)
		return false, err
	}

	return len(disruptionRollouts.Items) > 0, nil
}

func (h DeploymentHandler) UpdateDisruptionRolloutStatus(deployment *appsv1.Deployment, initContainersHash, containersHash map[string]string) error {
	disruptionRollouts, err := h.FetchAssociatedDisruptionRollouts(deployment)
	if err != nil {
		return err
	}

	for _, dr := range disruptionRollouts.Items {
		dr.Status.LatestInitContainersHash = initContainersHash
		dr.Status.LatestContainersHash = containersHash
		dr.Status.LastContainerChangeTime = &metav1.Time{Time: time.Now()}

		err = h.Client.Status().Update(context.TODO(), &dr)
		if err != nil {
			h.log.Errorw("unable to update DisruptionRollout status", "DisruptionRollout", dr.Name, "error", err)
			return err
		}
	}

	return nil
}

func ContainersChanged(oldPodSpec, newPodSpec *corev1.PodSpec, log *zap.SugaredLogger) (bool, map[string]string, map[string]string, error) {
	oldInitContainersHash, oldContainersHash, err := HashPodSpec(oldPodSpec)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to hash old pod spec: %w", err)
	}

	newInitContainersHash, newContainersHash, err := HashPodSpec(newPodSpec)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to hash new pod spec: %w", err)
	}

	if HashesChanged(oldInitContainersHash, newInitContainersHash, log) ||
		HashesChanged(oldContainersHash, newContainersHash, log) {
		return true, newInitContainersHash, newContainersHash, nil
	}

	return false, oldInitContainersHash, oldContainersHash, nil
}

func Hash(container *corev1.Container) (string, error) {
	containerJSON, err := json.Marshal(*container)
	if err != nil {
		return "", fmt.Errorf("error serializing instance spec: %w", err)
	}

	return hex.EncodeToString(md5.New().Sum(containerJSON)), nil
}

func HashPodSpec(podSpec *corev1.PodSpec) (map[string]string, map[string]string, error) {
	initContainersHash, err := HashContainerList(&podSpec.InitContainers)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to hash initContainers: %w", err)
	}

	containersHash, err := HashContainerList(&podSpec.Containers)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to hash containers: %w", err)
	}

	return initContainersHash, containersHash, nil
}

func HashContainerList(containers *[]corev1.Container) (map[string]string, error) {
	containerHashes := make(map[string]string, len(*containers))

	for _, container := range *containers {
		hash, err := Hash(&container)
		if err != nil {
			return nil, fmt.Errorf("error hashing container %s: %v", container.Name, err)
		}

		containerHashes[container.Name] = hash
	}

	return containerHashes, nil
}

func HashesChanged(oldHashes, newHashes map[string]string, log *zap.SugaredLogger) bool {
	// Check if any hashes in oldHashes don't match the ones in newHashes
	for k, v := range oldHashes {
		newVal, ok := newHashes[k]
		if !ok {
			log.Infof("container %s is missing in new hashes", k)
			return true
		}

		if newVal != v {
			log.Infof("hash for container %s has changed", k)
			return true
		}
	}

	// Check if any hashes in newHashes don't match the ones in oldHashes
	for k, v := range newHashes {
		oldVal, ok := oldHashes[k]
		if !ok {
			log.Infof("container %s is missing in old hashes", k)
			return true
		}

		if oldVal != v {
			log.Infof("hash for container %s has changed", k)
			return true
		}
	}

	return false
}

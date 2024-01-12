// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package watchers

import (
	context "context"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentHandler struct {
	Client client.Client
	log    *zap.SugaredLogger
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

	// It would be more efficient to use label selectors,
	// however it would require a webhook to add those labels when new rollouts are created
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

		err = h.Client.Status().Update(context.Background(), &dr)
		if err != nil {
			h.log.Errorw("unable to update DisruptionRollout status", "DisruptionRollout", dr.Name, "error", err)
			return err
		}
	}

	return nil
}

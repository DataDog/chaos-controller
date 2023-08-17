// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
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

type StatefulSetHandler struct {
	Client client.Client
	log    *zap.SugaredLogger
}

func NewStatefulSetHandler(client client.Client, logger *zap.SugaredLogger) StatefulSetHandler {
	return StatefulSetHandler{
		Client: client,
		log:    logger,
	}
}

// OnAdd is a handler function for the add of a statefulset
func (h StatefulSetHandler) OnAdd(obj interface{}) {
	statefulset, ok := obj.(*appsv1.StatefulSet)

	// If the object is not a statefulset, do nothing
	if !ok {
		return
	}

	// If statefulset doesn't have associated disruption rollout, do nothing
	hasDisruptionRollout, err := h.HasAssociatedDisruptionRollout(statefulset)
	if err != nil {
		return
	}

	if !hasDisruptionRollout {
		return
	}

	initContainersHash, containersHash, err := HashPodSpec(&statefulset.Spec.Template.Spec)
	if err != nil {
		return
	}

	err = h.UpdateDisruptionRolloutStatus(statefulset, initContainersHash, containersHash)
	if err != nil {
		return
	}
}

// OnUpdate is a handler function for the update of a statefulset
func (h StatefulSetHandler) OnUpdate(oldObj, newObj interface{}) {
	// Convert oldObj and newObj to Deployment objects
	oldStatefulSet, okOldStatefulSet := oldObj.(*appsv1.StatefulSet)
	newStatefulSet, okNewStatefulSet := newObj.(*appsv1.StatefulSet)

	// If both old and new are not statefulsets, do nothing
	if !okOldStatefulSet || !okNewStatefulSet {
		return
	}

	// If statefulset doesn't have associated disruption rollout, do nothing
	hasDisruptionRollout, err := h.HasAssociatedDisruptionRollout(newStatefulSet)
	if !hasDisruptionRollout || err != nil {
		return
	}

	// If containers have't changed, do nothing
	containersChanged, initContainersHash, containersHash, err := ContainersChanged(&oldStatefulSet.Spec.Template.Spec, &newStatefulSet.Spec.Template.Spec, h.log)
	if !containersChanged || err != nil {
		return
	}

	err = h.UpdateDisruptionRolloutStatus(newStatefulSet, initContainersHash, containersHash)
	if err != nil {
		return
	}
}

// OnDelete is a handler function for the delete of a statefulset
func (h StatefulSetHandler) OnDelete(_ interface{}) {
	// Do nothing on delete event
}

func (h StatefulSetHandler) FetchAssociatedDisruptionRollouts(statefulset *appsv1.StatefulSet) (*chaosv1beta1.DisruptionRolloutList, error) {
	indexedValue := "statefulset" + "-" + statefulset.Namespace + "-" + statefulset.Name

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

func (h StatefulSetHandler) HasAssociatedDisruptionRollout(statefulset *appsv1.StatefulSet) (bool, error) {
	disruptionRollouts, err := h.FetchAssociatedDisruptionRollouts(statefulset)
	if err != nil {
		h.log.Errorw("unable to check for associated DisruptionRollout", "StatefulSet", statefulset.Name, "error", err)
		return false, err
	}

	return len(disruptionRollouts.Items) > 0, nil
}

func (h StatefulSetHandler) UpdateDisruptionRolloutStatus(statefulset *appsv1.StatefulSet, initContainersHash, containersHash map[string]string) error {
	disruptionRollouts, err := h.FetchAssociatedDisruptionRollouts(statefulset)
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

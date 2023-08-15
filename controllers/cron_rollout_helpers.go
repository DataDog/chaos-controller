package controllers

import (
	"context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getChildDisruptions retrieves disruptions associated with a resource by its label.
// Most of the time, this will return an empty list as disruptions are typically short-lived objects.
func getChildDisruptions(ctx context.Context, cl client.Client, log *zap.SugaredLogger, namespace, labelKey, labelVal string) (*chaosv1beta1.DisruptionList, error) {
	disruptions := &chaosv1beta1.DisruptionList{}
	labelSelector := labels.SelectorFromSet(labels.Set{labelKey: labelVal})

	if err := cl.List(ctx, disruptions, client.InNamespace(namespace), &client.ListOptions{LabelSelector: labelSelector}); err != nil {
		log.Errorw("unable to list Disruptions", "err", err)
		return disruptions, err
	}

	return disruptions, nil
}

// getTargetResource retrieves the specified target resource (Deployment or StatefulSet).
// It returns the target resource object and any error encountered during retrieval.
func getTargetResource(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (client.Object, error) {
	var targetObj client.Object

	switch targetResource.Kind {
	case "deployment":
		targetObj = &appsv1.Deployment{}
	case "statefulset":
		targetObj = &appsv1.StatefulSet{}
	}

	err := cl.Get(ctx, types.NamespacedName{
		Name:      targetResource.Name,
		Namespace: namespace,
	}, targetObj)

	return targetObj, err
}

// checkTargetResourceExists determines if the target resource exists.
// Returns a boolean indicating presence and an error if one occurs.
func checkTargetResourceExists(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (bool, error) {
	_, err := getTargetResource(ctx, cl, targetResource, namespace)

	if errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// getSelectors retrieves the labels of the specified target resource (Deployment or StatefulSet).
// Returns a set of labels to be used as Disruption selectors and an error if retrieval fails.
func getSelectors(ctx context.Context, cl client.Client, targetResource *chaosv1beta1.TargetResourceSpec, namespace string) (labels.Set, error) {
	targetObj, err := getTargetResource(ctx, cl, targetResource, namespace)
	if err != nil {
		return nil, err
	}

	labels := targetObj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	return labels, nil
}

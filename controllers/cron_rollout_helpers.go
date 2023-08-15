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
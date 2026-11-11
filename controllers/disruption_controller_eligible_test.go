// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// targetChaosPods is a ChaosPodService whose only implemented call lists the
// given chaos pods; getEligibleTargets calls nothing else.
type targetChaosPods struct {
	services.ChaosPodService
	pods []corev1.Pod
}

func (s targetChaosPods) GetChaosPodsOfDisruption(context.Context, *chaosv1beta1.Disruption, labels.Set) ([]corev1.Pod, error) {
	return s.pods, nil
}

// An injector pod that another disruption just created is Pending and has no
// container statuses yet. Its target is about to be disrupted, so a second
// disruption without allowDisruptedTargets must not select it.
func TestGetEligibleTargetsSkipsTargetsOfPendingInjectors(t *testing.T) {
	pending := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "chaos-network-1-abcde",
			Labels: map[string]string{chaostypes.DisruptionKindLabel: string(chaostypes.DisruptionKindNetworkDisruption)},
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}

	count := intstr.FromInt(1)
	instance := &chaosv1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{Name: "disk-pressure-2", Namespace: "team"},
		Spec: chaosv1beta1.DisruptionSpec{
			Count:        &count,
			Level:        chaostypes.DisruptionLevelPod,
			DiskPressure: &chaosv1beta1.DiskPressureSpec{Path: "/data"},
		},
	}

	reconciler := &DisruptionReconciler{
		log:             zap.NewNop().Sugar(),
		ChaosPodService: targetChaosPods{pods: []corev1.Pod{pending}},
	}

	eligible, err := reconciler.getEligibleTargets(context.Background(), instance, []string{"web-1"})
	if err != nil {
		t.Fatal(err)
	}

	if _, found := eligible["web-1"]; found {
		t.Fatal("web-1 was selected although a pending injector pod of another disruption targets it")
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"errors"
	"reflect"
	"testing"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type targetSelectionFailedListReader struct {
	client.Reader
	err error
}

func (r targetSelectionFailedListReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return r.err
}

func TestSelectTargetsPreservesTargetsWhenListingFails(t *testing.T) {
	for _, level := range []chaostypes.DisruptionLevel{chaostypes.DisruptionLevelPod, chaostypes.DisruptionLevelNode} {
		for _, count := range []intstr.IntOrString{intstr.FromInt(1), intstr.FromString("100%")} {
			t.Run(string(level)+"/"+count.String(), func(t *testing.T) {
				listErr := errors.New("API server unavailable")
				instance := &chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{Name: "ongoing", Namespace: "app"},
					Spec: chaosv1beta1.DisruptionSpec{
						Level:    level,
						Selector: map[string]string{"app": "web"},
						Count:    &count,
						Network:  &chaosv1beta1.NetworkDisruptionSpec{Drop: 10},
					},
					Status: chaosv1beta1.DisruptionStatus{
						InjectionStatus:      chaostypes.DisruptionInjectionStatusInjected,
						SelectedTargetsCount: 1,
						DesiredTargetsCount:  1,
						InjectedTargetsCount: 1,
						TargetInjections: chaosv1beta1.TargetInjections{
							"target": chaosv1beta1.TargetInjectorMap{
								chaostypes.DisruptionKindNetworkDisruption: {
									InjectorPodName: "existing-injector",
									InjectionStatus: chaostypes.DisruptionTargetInjectionStatusInjected,
								},
							},
						},
					},
				}
				before := instance.DeepCopy().Status
				r := &DisruptionReconciler{
					APIReader:      targetSelectionFailedListReader{err: listErr},
					TargetSelector: targetselector.NewRunningTargetSelector(false, "controller-node"),
					log:            zap.NewNop().Sugar(),
				}
				err := r.selectTargets(context.Background(), instance)
				if !errors.Is(err, listErr) {
					t.Errorf("selection failure must reach Reconcile for retry; got %v", err)
				}
				if !reflect.DeepEqual(instance.Status, before) {
					t.Errorf("a failed List changed the existing injection status: before=%#v after=%#v", before, instance.Status)
				}
			})
		}
	}
}

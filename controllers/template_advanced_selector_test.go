// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/targetselector"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTemplateAdvancedSelectorSurvivesTargetSelectors(t *testing.T) {
	for _, parentKind := range []string{"cron", "rollout"} {
		for _, targetKind := range []string{"deployment", "statefulset"} {
			for _, useExpression := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/expression=%t", parentKind, targetKind, useExpression), func(t *testing.T) {
					scheme := runtime.NewScheme()
					if err := chaos.AddToScheme(scheme); err != nil {
						t.Fatal(err)
					}
					if err := appsv1.AddToScheme(scheme); err != nil {
						t.Fatal(err)
					}
					selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "example"}}
					if useExpression {
						selector = &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"example"}}}}
					}
					metadata := metav1.ObjectMeta{Name: "app", Namespace: "ns"}
					podTemplate := corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "example"}},
						Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example:v1"}}},
					}
					var target client.Object
					if targetKind == "deployment" {
						target = &appsv1.Deployment{ObjectMeta: metadata, Spec: appsv1.DeploymentSpec{Selector: selector, Template: podTemplate}}
					} else {
						target = &appsv1.StatefulSet{ObjectMeta: metadata, Spec: appsv1.StatefulSetSpec{ServiceName: "app", Selector: selector, Template: podTemplate}}
					}
					count := intstr.FromInt(1)
					template := chaos.DisruptionSpec{
						Count: &count, Level: "pod", Duration: "1m",
						Network:          &chaos.NetworkDisruptionSpec{Drop: 100},
						AdvancedSelector: []metav1.LabelSelectorRequirement{{Key: "track", Operator: metav1.LabelSelectorOpIn, Values: []string{"canary"}}},
					}
					targetRef := chaos.TargetResourceSpec{Kind: targetKind, Name: target.GetName()}
					ownerMeta := metav1.ObjectMeta{Name: "scheduled", Namespace: "ns", UID: "owner"}
					var owner client.Object
					if parentKind == "cron" {
						owner = &chaos.DisruptionCron{ObjectMeta: ownerMeta, Spec: chaos.DisruptionCronSpec{Schedule: "@every 1h", TargetResource: targetRef, DisruptionTemplate: template}}
					} else {
						owner = &chaos.DisruptionRollout{ObjectMeta: ownerMeta, Spec: chaos.DisruptionRolloutSpec{TargetResource: targetRef, DisruptionTemplate: template}}
					}
					cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(target).Build()
					child, err := CreateDisruptionFromTemplate(context.Background(), cl, scheme, owner, &targetRef, &template, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
					if err != nil {
						t.Fatal(err)
					}
					effective, err := targetselector.GetLabelSelectorFromInstance(child)
					if err != nil {
						t.Fatal(err)
					}
					if !effective.Matches(labels.Set{"app": "example", "track": "canary"}) {
						t.Fatal("child does not select the intended canary target")
					}
					if effective.Matches(labels.Set{"app": "example", "track": "stable"}) {
						t.Fatalf("child selector %q discarded the template's canary restriction", effective.String())
					}
					if effective.Matches(labels.Set{"app": "unrelated", "track": "canary"}) {
						t.Fatal("child selects pods outside the target resource")
					}
				})
			}
		}
	}
}

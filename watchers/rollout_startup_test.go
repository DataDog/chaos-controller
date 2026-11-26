// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"context"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Rollout informer startup", func() {
	DescribeTable("only records new container changes", func(kind string, changed bool) {
		scheme := runtime.NewScheme()
		Expect(v1beta1.AddToScheme(scheme)).To(Succeed())
		podSpec := corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "app:v1"}}}
		initHash, containerHash, err := watchers.HashPodSpec(&podSpec)
		Expect(err).NotTo(HaveOccurred())
		observed := metav1.NewTime(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
		tested := metav1.NewTime(observed.Add(time.Minute))
		rollout := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "tested", Namespace: "ns"},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: kind, Name: "app"},
			},
			Status: v1beta1.DisruptionRolloutStatus{
				LatestInitContainersHash: initHash,
				LatestContainersHash:     containerHash,
				LastContainerChangeTime:  &observed,
				LastScheduleTime:         &tested,
			},
		}
		unobserved := rollout.DeepCopy()
		unobserved.Name = "unobserved"
		unobserved.Status = v1beta1.DisruptionRolloutStatus{}
		cl := fake.NewClientBuilder().WithScheme(scheme).
			WithStatusSubresource(&v1beta1.DisruptionRollout{}).
			WithObjects(rollout, unobserved).
			WithIndex(&v1beta1.DisruptionRollout{}, "targetResource", func(obj client.Object) []string {
				dr := obj.(*v1beta1.DisruptionRollout)
				return []string{watchers.TargetResourceIndexedValue(dr.Spec.TargetResource.Kind, dr.Namespace, dr.Spec.TargetResource.Name)}
			}).Build()
		if changed {
			podSpec.Containers[0].Image = "app:v2"
		}
		metadata := metav1.ObjectMeta{Name: "app", Namespace: "ns"}
		template := corev1.PodTemplateSpec{Spec: podSpec}
		if kind == "deployment" {
			handler := watchers.NewDeploymentHandler(cl, zap.NewNop().Sugar())
			handler.OnAdd(&appsv1.Deployment{ObjectMeta: metadata, Spec: appsv1.DeploymentSpec{Template: template}})
		} else {
			handler := watchers.NewStatefulSetHandler(cl, zap.NewNop().Sugar())
			handler.OnAdd(&appsv1.StatefulSet{ObjectMeta: metadata, Spec: appsv1.StatefulSetSpec{Template: template}})
		}
		current := &v1beta1.DisruptionRollout{}
		Expect(cl.Get(context.Background(), client.ObjectKeyFromObject(rollout), current)).To(Succeed())
		Expect(current.Status.LastScheduleTime.Time).To(BeTemporally("==", tested.Time))
		if changed {
			Expect(current.Status.LastContainerChangeTime).NotTo(BeNil())
			Expect(current.Status.LastContainerChangeTime.After(observed.Time)).To(BeTrue())
		} else {
			Expect(current.Status.LastContainerChangeTime.Time).To(BeTemporally("==", observed.Time))
		}
		_, expectedHash, err := watchers.HashPodSpec(&podSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(current.Status.LatestContainersHash).To(Equal(expectedHash))
		Expect(cl.Get(context.Background(), client.ObjectKeyFromObject(unobserved), current)).To(Succeed())
		Expect(current.Status.LastContainerChangeTime).NotTo(BeNil())
		Expect(current.Status.LatestContainersHash).To(Equal(expectedHash))
	},
		Entry("unchanged deployment", "deployment", false),
		Entry("deployment changed while offline", "deployment", true),
		Entry("unchanged statefulset", "statefulset", false),
		Entry("statefulset changed while offline", "statefulset", true),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/DataDog/chaos-controller/api/v1beta1"
)

var _ = Describe("TargetResourceIndexedValue", func() {
	It("does not collide for a namespace and name pair that share a hyphen", func() {
		// ns a-b + name c vs ns a + name b-c collided with the "-"-joined key
		Expect(TargetResourceIndexedValue("deployment", "a-b", "c")).ToNot(
			Equal(TargetResourceIndexedValue("deployment", "a", "b-c")))
	})

	It("keeps distinct kinds, namespaces and names distinguishable", func() {
		Expect(TargetResourceIndexedValue("deployment", "ns", "name")).ToNot(
			Equal(TargetResourceIndexedValue("statefulset", "ns", "name")))
		Expect(TargetResourceIndexedValue("deployment", "ns1", "name")).ToNot(
			Equal(TargetResourceIndexedValue("deployment", "ns2", "name")))
		Expect(TargetResourceIndexedValue("deployment", "ns", "name1")).ToNot(
			Equal(TargetResourceIndexedValue("deployment", "ns", "name2")))
	})
})

var _ = Describe("DeploymentHandler target resource index lookup", func() {
	It("only updates the rollout targeting the given deployment", func() {
		sch := runtime.NewScheme()
		Expect(v1beta1.AddToScheme(sch)).To(Succeed())

		// two rollouts whose "-"-joined keys used to collide:
		// (ns a-b, deployment c) and (ns a, deployment b-c)
		matching := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "matching", Namespace: "a-b", UID: types.UID("matching-uid")},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: "deployment", Name: "c"},
			},
		}
		other := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "a", UID: types.UID("other-uid")},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: "deployment", Name: "b-c"},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(sch).
			WithStatusSubresource(&v1beta1.DisruptionRollout{}).
			WithIndex(&v1beta1.DisruptionRollout{}, TargetResourceIndexKey, func(o client.Object) []string {
				dr := o.(*v1beta1.DisruptionRollout)
				return []string{TargetResourceIndexedValue(dr.Spec.TargetResource.Kind, dr.GetNamespace(), dr.Spec.TargetResource.Name)}
			}).
			WithObjects(matching, other).Build()

		handler := NewDeploymentHandler(k8sClient, zaptest.NewLogger(GinkgoT()).Sugar())

		old := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "a-b"},
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:1"}}},
			}},
		}
		new := old.DeepCopy()
		new.Spec.Template.Spec.Containers[0].Image = "nginx:2"

		handler.OnUpdate(old, new)

		updated := &v1beta1.DisruptionRollout{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "a-b", Name: "matching"}, updated)).To(Succeed())
		Expect(updated.Status.LatestContainersHash).To(HaveKey("app"))

		untouched := &v1beta1.DisruptionRollout{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "a", Name: "other"}, untouched)).To(Succeed())
		Expect(untouched.Status.LatestContainersHash).To(BeNil())
	})
})

var _ = Describe("StatefulSetHandler target resource index lookup", func() {
	It("only updates the rollout targeting the given statefulset", func() {
		sch := runtime.NewScheme()
		Expect(v1beta1.AddToScheme(sch)).To(Succeed())

		matching := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "matching", Namespace: "a-b", UID: types.UID("matching-uid")},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: "statefulset", Name: "c"},
			},
		}
		other := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "a", UID: types.UID("other-uid")},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: "statefulset", Name: "b-c"},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(sch).
			WithStatusSubresource(&v1beta1.DisruptionRollout{}).
			WithIndex(&v1beta1.DisruptionRollout{}, TargetResourceIndexKey, func(o client.Object) []string {
				dr := o.(*v1beta1.DisruptionRollout)
				return []string{TargetResourceIndexedValue(dr.Spec.TargetResource.Kind, dr.GetNamespace(), dr.Spec.TargetResource.Name)}
			}).
			WithObjects(matching, other).Build()

		handler := NewStatefulSetHandler(k8sClient, zaptest.NewLogger(GinkgoT()).Sugar())

		old := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "a-b"},
			Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx:1"}}},
			}},
		}
		new := old.DeepCopy()
		new.Spec.Template.Spec.Containers[0].Image = "nginx:2"

		handler.OnUpdate(old, new)

		updated := &v1beta1.DisruptionRollout{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "a-b", Name: "matching"}, updated)).To(Succeed())
		Expect(updated.Status.LatestContainersHash).To(HaveKey("app"))

		untouched := &v1beta1.DisruptionRollout{}
		Expect(k8sClient.Get(context.Background(), types.NamespacedName{Namespace: "a", Name: "other"}, untouched)).To(Succeed())
		Expect(untouched.Status.LatestContainersHash).To(BeNil())
	})
})

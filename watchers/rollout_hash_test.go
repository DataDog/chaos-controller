// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Rollout hash status integration", func() {
	DescribeTable("persists digests for regular and init containers, replacing legacy values", func(kind string) {
		ctx := context.Background()
		scheme := runtime.NewScheme()
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(v1beta1.AddToScheme(scheme)).To(Succeed())
		container := corev1.Container{
			Name: "app", Image: "nginx:latest",
			Env: []corev1.EnvVar{{Name: "TOKEN", Value: "synthetic-sensitive-value"}},
		}
		podSpec := corev1.PodSpec{
			Containers:     []corev1.Container{container},
			InitContainers: []corev1.Container{*container.DeepCopy()},
		}
		podSpec.InitContainers[0].Name = "init"
		rollout := &v1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "rollout", Namespace: "test"},
			Spec: v1beta1.DisruptionRolloutSpec{
				TargetResource: v1beta1.TargetResourceSpec{Kind: kind, Name: "workload"},
			},
		}
		cl := fake.NewClientBuilder().WithScheme(scheme).
			WithStatusSubresource(&v1beta1.DisruptionRollout{}).WithObjects(rollout).
			WithIndex(&v1beta1.DisruptionRollout{}, "targetResource", func(obj client.Object) []string {
				dr := obj.(*v1beta1.DisruptionRollout)
				return []string{dr.Spec.TargetResource.Kind + "-" + dr.Namespace + "-" + dr.Spec.TargetResource.Name}
			}).Build()
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "test"},
			Spec:       appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: podSpec}},
		}
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: deployment.ObjectMeta,
			Spec:       appsv1.StatefulSetSpec{Template: *deployment.Spec.Template.DeepCopy()},
		}
		// A fresh handler models an informer add event following a watcher restart.
		onAdd := func() {
			if kind == "deployment" {
				watchers.NewDeploymentHandler(cl, logger).OnAdd(deployment)
			} else {
				watchers.NewStatefulSetHandler(cl, logger).OnAdd(statefulSet)
			}
		}
		readStatus := func() v1beta1.DisruptionRolloutStatus {
			Expect(cl.Get(ctx, client.ObjectKeyFromObject(rollout), rollout)).To(Succeed())
			return rollout.Status
		}
		assertDigests := func() {
			status := readStatus()
			Expect(status.LatestContainersHash).To(HaveLen(1))
			Expect(status.LatestInitContainersHash).To(HaveLen(1))
			for _, hashes := range []map[string]string{status.LatestContainersHash, status.LatestInitContainersHash} {
				for _, hash := range hashes {
					Expect(hash).To(MatchRegexp(`^[0-9a-f]{32}$`))
					decoded, err := hex.DecodeString(hash)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(decoded)).NotTo(ContainSubstring(container.Env[0].Value))
				}
			}
			serialized, err := yaml.Marshal(rollout)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(serialized)).NotTo(ContainSubstring(container.Env[0].Value))
			Expect(string(serialized)).NotTo(ContainSubstring(hex.EncodeToString([]byte(container.Env[0].Value))))
		}
		onAdd()
		assertDigests()
		first := readStatus()
		onAdd()
		second := readStatus()
		Expect(second.LatestContainersHash).To(Equal(first.LatestContainersHash))
		Expect(second.LatestInitContainersHash).To(Equal(first.LatestInitContainersHash))

		// Reproduce the previous format exactly: JSON followed by MD5(empty).
		legacy := func(c corev1.Container) string {
			data, err := json.Marshal(c)
			Expect(err).NotTo(HaveOccurred())
			return hex.EncodeToString(md5.New().Sum(data))
		}
		seedLegacy := func() {
			rollout.Status.LatestContainersHash = map[string]string{"app": legacy(container)}
			rollout.Status.LatestInitContainersHash = map[string]string{"init": legacy(podSpec.InitContainers[0])}
			Expect(cl.Status().Update(ctx, rollout)).To(Succeed())
		}
		seedLegacy()
		onAdd()
		assertDigests()
		Expect(readStatus().LatestContainersHash).To(Equal(first.LatestContainersHash))
		seedLegacy()

		// An unchanged update preserves existing behavior; a real env change replaces both maps.
		if kind == "deployment" {
			h := watchers.NewDeploymentHandler(cl, logger)
			h.OnUpdate(deployment, deployment.DeepCopy())
			Expect(readStatus().LatestContainersHash["app"]).To(Equal(legacy(container)))
			updated := deployment.DeepCopy()
			updated.Spec.Template.Spec.Containers[0].Env[0].Value += "-changed"
			updated.Spec.Template.Spec.InitContainers[0].Env[0].Value += "-changed"
			h.OnUpdate(deployment, updated)
		} else {
			h := watchers.NewStatefulSetHandler(cl, logger)
			h.OnUpdate(statefulSet, statefulSet.DeepCopy())
			Expect(readStatus().LatestContainersHash["app"]).To(Equal(legacy(container)))
			updated := statefulSet.DeepCopy()
			updated.Spec.Template.Spec.Containers[0].Env[0].Value += "-changed"
			updated.Spec.Template.Spec.InitContainers[0].Env[0].Value += "-changed"
			h.OnUpdate(statefulSet, updated)
		}
		assertDigests()
		Expect(readStatus().LatestContainersHash).NotTo(Equal(first.LatestContainersHash))
		Expect(readStatus().LatestInitContainersHash).NotTo(Equal(first.LatestInitContainersHash))
	}, Entry("Deployment", "deployment"), Entry("StatefulSet", "statefulset"))
})

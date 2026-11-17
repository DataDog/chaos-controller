// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"encoding/hex"
	"os"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Requires an isolated test cluster running this revision with
// controller.disruptionRolloutEnabled=true. The workload never starts any pods.
var _ = Describe("Rollout hash E2E", Label("rollout-hash"), func() {
	It("stores only digests in API output after an environment value changes", func(ctx SpecContext) {
		if os.Getenv("RUN_ROLLOUT_HASH_E2E") != "true" {
			Skip("requires explicit opt-in and a rollout-enabled controller in an isolated test cluster")
		}
		const syntheticValue = "synthetic-sensitive-value"
		replicas := int32(0)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "rollout-hash", Namespace: namespace},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "rollout-hash"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "rollout-hash"}},
					Spec: corev1.PodSpec{
						Containers:     []corev1.Container{{Name: "app", Image: "nginx:latest", Env: []corev1.EnvVar{{Name: "TOKEN", Value: "initial"}}}},
						InitContainers: []corev1.Container{{Name: "init", Image: "busybox:latest", Env: []corev1.EnvVar{{Name: "TOKEN", Value: "initial"}}}},
					},
				},
			},
		}
		rollout := &chaosv1beta1.DisruptionRollout{
			ObjectMeta: metav1.ObjectMeta{Name: "rollout-hash", Namespace: namespace},
			Spec: chaosv1beta1.DisruptionRolloutSpec{
				TargetResource: chaosv1beta1.TargetResourceSpec{Kind: "deployment", Name: deployment.Name},
				DisruptionTemplate: chaosv1beta1.DisruptionSpec{
					DryRun: true, Count: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					Duration: "1s", Network: &chaosv1beta1.NetworkDisruptionSpec{Drop: 100},
				},
			},
		}
		Expect(k8sClient.Create(ctx, rollout)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) { Expect(k8sClient.Delete(ctx, rollout)).To(Succeed()) })
		Expect(k8sClient.Create(ctx, deployment)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) { Expect(k8sClient.Delete(ctx, deployment)).To(Succeed()) })
		Eventually(func(g Gomega) {
			g.Expect(APIReader.Get(ctx, client.ObjectKeyFromObject(rollout), rollout)).To(Succeed())
			if len(rollout.Status.LatestContainersHash) == 0 {
				// The workload add can precede observation of the rollout in the controller cache.
				g.Expect(APIReader.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)).To(Succeed())
				deployment.Spec.Template.Spec.Containers[0].Env[0].Value += "-retry"
				g.Expect(k8sClient.Update(ctx, deployment)).To(Succeed())
			}
			g.Expect(rollout.Status.LatestContainersHash["app"]).To(MatchRegexp(`^[0-9a-f]{32}$`))
			g.Expect(rollout.Status.LatestInitContainersHash["init"]).To(MatchRegexp(`^[0-9a-f]{32}$`))
		}).Within(k8sAPIServerResponseTimeout).Should(Succeed())
		oldApp := rollout.Status.LatestContainersHash["app"]
		oldInit := rollout.Status.LatestInitContainersHash["init"]
		Eventually(func() error {
			if err := APIReader.Get(ctx, client.ObjectKeyFromObject(deployment), deployment); err != nil {
				return err
			}
			deployment.Spec.Template.Spec.Containers[0].Env[0].Value = syntheticValue
			deployment.Spec.Template.Spec.InitContainers[0].Env[0].Value = syntheticValue
			return k8sClient.Update(ctx, deployment)
		}).Within(k8sAPIServerResponseTimeout).Should(Succeed())
		expectedApp, err := watchers.Hash(&deployment.Spec.Template.Spec.Containers[0])
		Expect(err).NotTo(HaveOccurred())
		expectedInit, err := watchers.Hash(&deployment.Spec.Template.Spec.InitContainers[0])
		Expect(err).NotTo(HaveOccurred())
		Eventually(func(g Gomega) {
			g.Expect(APIReader.Get(ctx, client.ObjectKeyFromObject(rollout), rollout)).To(Succeed())
			g.Expect(rollout.Status.LatestContainersHash["app"]).To(Equal(expectedApp))
			g.Expect(rollout.Status.LatestInitContainersHash["init"]).To(Equal(expectedInit))
			for name, hash := range map[string]string{
				"app":  rollout.Status.LatestContainersHash["app"],
				"init": rollout.Status.LatestInitContainersHash["init"],
			} {
				g.Expect(hash).To(MatchRegexp(`^[0-9a-f]{32}$`), name)
				decoded, err := hex.DecodeString(hash)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(string(decoded)).NotTo(ContainSubstring(syntheticValue))
			}
			g.Expect(rollout.Status.LatestContainersHash["app"]).NotTo(Equal(oldApp))
			g.Expect(rollout.Status.LatestInitContainersHash["init"]).NotTo(Equal(oldInit))
			output, err := yaml.Marshal(rollout)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(string(output)).NotTo(ContainSubstring(syntheticValue))
			g.Expect(string(output)).NotTo(ContainSubstring(hex.EncodeToString([]byte(syntheticValue))))
		}).Within(k8sAPIServerResponseTimeout).Should(Succeed())
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
)

var _ = Describe("User info immutability", func() {
	// Run only against the isolated development cluster used by this suite. The
	// caller must be allowed to impersonate this synthetic user and its groups.
	DescribeTable("rejects formatting collisions while allowing count updates", func(ctx SpecContext, oldGroups, newGroups []string) {
		config := rest.CopyConfig(restConfig)
		config.Impersonate = rest.ImpersonationConfig{
			UserName: "identity-test",
			// Retain development-cluster admin access for the synthetic user without
			// creating any RBAC resources. This must never target a shared cluster.
			Groups: append([]string{"system:masters"}, oldGroups...),
		}
		identityClient, err := client.New(config, client.Options{Scheme: k8sClient.Scheme()})
		Expect(err).NotTo(HaveOccurred())
		disruption := chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "identity-test-", Namespace: namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				DryRun: true, Duration: chaosv1beta1.DisruptionDuration(shortDisruptionDuration),
				Count:      &intstr.IntOrString{IntVal: 1},
				Selector:   map[string]string{"identity-test": "no-targets"},
				Unsafemode: &chaosv1beta1.UnsafemodeSpec{DisableAll: true},
				Network:    &chaosv1beta1.NetworkDisruptionSpec{Drop: 100},
			},
		}
		Expect(identityClient.Create(ctx, &disruption)).To(Succeed())
		DeferCleanup(DeleteDisruption, disruption)
		key := client.ObjectKeyFromObject(&disruption)
		Expect(APIReader.Get(ctx, key, &disruption)).To(Succeed())
		original, err := disruption.UserInfo()
		Expect(err).NotTo(HaveOccurred())
		Expect(original.Username).To(Equal("identity-test"))
		Expect(original.Groups).To(ContainElements(oldGroups))

		changed := disruption.DeepCopy()
		updated := original.DeepCopy()
		updated.Groups = nil
		replaced := false
		for _, group := range original.Groups {
			if group == "a" || group == "b" || group == "a b" {
				if !replaced {
					updated.Groups = append(updated.Groups, newGroups...)
					replaced = true
				}
				continue
			}
			updated.Groups = append(updated.Groups, group)
		}
		Expect(fmt.Sprintf("%v", *updated)).To(Equal(fmt.Sprintf("%v", original)))
		Expect(changed.SetUserInfo(*updated)).To(Succeed())
		err = k8sClient.Patch(ctx, changed, client.MergeFrom(&disruption))
		Expect(apierrors.IsForbidden(err)).To(BeTrue(), "expected admission rejection, got %v", err)
		Expect(err).To(MatchError(ContainSubstring("the user info annotation is immutable")))

		Expect(APIReader.Get(ctx, key, &disruption)).To(Succeed())
		actual, err := disruption.UserInfo()
		Expect(err).NotTo(HaveOccurred())
		Expect(actual).To(Equal(original))

		// A genuine API update and dynamic count changes must remain supported.
		Eventually(func(ctx SpecContext) error {
			if err := APIReader.Get(ctx, key, &disruption); err != nil {
				return err
			}
			return k8sClient.Update(ctx, disruption.DeepCopy())
		}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed())
		Expect(APIReader.Get(ctx, key, &disruption)).To(Succeed())
		changed = disruption.DeepCopy()
		changed.Spec.Count = &intstr.IntOrString{IntVal: 2}
		Expect(k8sClient.Patch(ctx, changed, client.MergeFrom(&disruption))).To(Succeed())
		Expect(APIReader.Get(ctx, key, &disruption)).To(Succeed())
		Expect(disruption.Spec.Count.IntVal).To(Equal(int32(2)))
	},
		Entry("merging group values", []string{"a", "b"}, []string{"a b"}),
		Entry("splitting group values", []string{"a b"}, []string{"a", "b"}),
	)
})

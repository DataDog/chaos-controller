// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("Legacy disruption updates", func() {
	var old, updated *Disruption

	BeforeEach(func() {
		old = makeValidNetworkDisruption()
		// Model a persisted spec that a newer validator no longer accepts, without
		// weakening validation or changing the rules used for new disruptions.
		old.Spec.Network.Drop = 101
		Expect(old.Spec.Validate()).To(HaveOccurred())
		controllerutil.AddFinalizer(old, chaostypes.DisruptionFinalizer)
		old.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		updated = old.DeepCopy()
		k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	})

	AfterEach(func() {
		k8sClient = nil
	})

	DescribeTable("allows finalizer removal with an unchanged invalid spec",
		func(ctx SpecContext, static bool) {
			old.Spec.StaticTargeting = static
			updated.Spec.StaticTargeting = static
			controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
			_, err := updated.ValidateUpdate(ctx, old, updated)
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("dynamic targeting", false),
		Entry("static targeting", true),
	)

	It("allows metadata updates while the spec is unchanged", func(ctx SpecContext) {
		updated.Labels = map[string]string{"example.com/note": "updated"}
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).ToNot(HaveOccurred())
	})

	It("allows finalizer removal even when the persisted count is invalid", func(ctx SpecContext) {
		old.Spec.Count = nil
		updated.Spec.Count = nil
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).ToNot(HaveOccurred())
	})

	It("still validates the entire spec when a dynamic count changes", func(ctx SpecContext) {
		updated.Spec.Count = &intstr.IntOrString{IntVal: 2}
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(MatchError(ContainSubstring("Drop")))
	})

	It("still rejects an invalid count change on an otherwise valid spec", func(ctx SpecContext) {
		old.Spec.Network.Drop = 100
		updated.Spec.Network.Drop = 100
		updated.Spec.Count = &intstr.IntOrString{IntVal: -1}
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(HaveOccurred())
	})

	It("still rejects immutable spec changes during finalizer removal", func(ctx SpecContext) {
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
		updated.Spec.Network.Drop = 100
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(MatchError(ContainSubstring("only a disruption spec's Count field can be updated")))
	})

	It("still protects finalizers when chaos pods exist", func(ctx SpecContext) {
		k8sClient = makek8sClientWithDisruptionPod()
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(MatchError(HavePrefix("unable to remove disruption finalizer")))
	})

	It("still protects user info with an unchanged invalid spec", func(ctx SpecContext) {
		Expect(updated.SetUserInfo(authv1.UserInfo{Username: "example-user"})).To(Succeed())
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(MatchError("the user info annotation is immutable"))
	})

	It("keeps the existing immutable-spec rejection for newly defaulted fields", func(ctx SpecContext) {
		previousDuration := defaultDuration
		defaultDuration = time.Hour
		DeferCleanup(func() { defaultDuration = previousDuration })
		old.Spec.Duration = ""
		updated.Spec.Duration = ""
		Expect(updated.Default(ctx, updated)).To(Succeed())
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
		_, err := updated.ValidateUpdate(ctx, old, updated)
		Expect(err).To(MatchError(ContainSubstring("only a disruption spec's Count field can be updated")))
	})

	DescribeTable("handles finalizer-removal AdmissionReviews through the HTTP webhook", func(changeCount, allowed bool) {
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)
		if changeCount {
			updated.Spec.Count = &intstr.IntOrString{IntVal: 2}
		}
		webhookScheme := runtime.NewScheme()
		Expect(AddToScheme(webhookScheme)).To(Succeed())
		oldJSON, err := json.Marshal(old)
		Expect(err).ToNot(HaveOccurred())
		updatedJSON, err := json.Marshal(updated)
		Expect(err).ToNot(HaveOccurred())
		review := admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
			Request: &admissionv1.AdmissionRequest{
				UID: "legacy-update-test", Operation: admissionv1.Update,
				Kind:     metav1.GroupVersionKind{Group: GroupVersion.Group, Version: GroupVersion.Version, Kind: DisruptionKind},
				Resource: metav1.GroupVersionResource{Group: GroupVersion.Group, Version: GroupVersion.Version, Resource: "disruptions"},
				Name:     old.Name, Namespace: old.Namespace,
				OldObject: runtime.RawExtension{Raw: oldJSON},
				Object:    runtime.RawExtension{Raw: updatedJSON},
			},
		}
		body, err := json.Marshal(review)
		Expect(err).ToNot(HaveOccurred())
		request := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		admission.WithValidator(webhookScheme, &Disruption{}).ServeHTTP(response, request)
		Expect(response.Code).To(Equal(http.StatusOK))
		var result admissionv1.AdmissionReview
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
		Expect(result.Response).ToNot(BeNil())
		Expect(result.Response.UID).To(Equal(review.Request.UID))
		Expect(result.Response.Allowed).To(Equal(allowed), "%s", response.Body.String())
	},
		Entry("unchanged invalid spec is admitted", false, true),
		Entry("count change still validates the invalid spec", true, false),
	)
})

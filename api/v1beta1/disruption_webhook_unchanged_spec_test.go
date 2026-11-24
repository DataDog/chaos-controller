// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disruption Webhook update of an unchanged spec", func() {
	It("lets the controller remove the finalizer of a disruption whose spec newer validation rejects", func(ctx SpecContext) {
		metricsSink = metricsnoop.New(logger)
		k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		DeferCleanup(func() { k8sClient = nil })

		// admitted by an older controller, rejected by the current validation
		stored := makeValidNetworkDisruption()
		stored.Spec.Network.Drop = 200
		Expect(stored.Spec.Validate()).To(HaveOccurred(), "precondition: the stored spec fails current validation")

		controllerutil.AddFinalizer(stored, chaostypes.DisruptionFinalizer)
		stored.DeletionTimestamp = &metav1.Time{Time: time.Now()}

		updated := stored.DeepCopy()
		controllerutil.RemoveFinalizer(updated, chaostypes.DisruptionFinalizer)

		_, err := updated.ValidateUpdate(ctx, stored, updated)
		Expect(err).ToNot(HaveOccurred())
	})

	It("still validates a spec whose count changes", func(ctx SpecContext) {
		metricsSink = metricsnoop.New(logger)

		stored := makeValidNetworkDisruption()
		stored.Spec.Network.Drop = 200

		updated := stored.DeepCopy()
		updated.Spec.Count.IntVal++

		_, err := updated.ValidateUpdate(ctx, stored, updated)
		Expect(err).To(HaveOccurred())
	})
})

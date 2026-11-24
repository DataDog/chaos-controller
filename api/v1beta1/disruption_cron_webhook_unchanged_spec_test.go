// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/mocks"
	metricsnoop "github.com/DataDog/chaos-controller/o11y/metrics/noop"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DisruptionCron Webhook update of an unchanged spec", func() {
	It("admits a metadata-only update of a cron whose template newer validation rejects", func(ctx SpecContext) {
		disruptionCronWebhookLogger = logger
		metricsSink = metricsnoop.New(logger)
		disruptionCronMetricsSink = metricsnoop.New(logger)
		recorder := mocks.NewEventRecorderMock(GinkgoT())
		recorder.EXPECT().AnnotatedEventf(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		disruptionCronWebhookRecorder = recorder
		DeferCleanup(func() { disruptionCronWebhookLogger = nil; disruptionCronWebhookRecorder = nil })

		// admitted by an older controller, rejected by the current validation
		stored := makeValidDisruptionCron()
		stored.Spec.DisruptionTemplate.Network = &NetworkDisruptionSpec{Drop: 200}
		Expect(stored.validateDisruptionCronSpec()).To(HaveOccurred(), "precondition: the stored template fails current validation")

		updated := stored.DeepCopy()
		updated.Labels = map[string]string{"touched": "true"}

		_, err := updated.ValidateUpdate(ctx, stored, updated)
		Expect(err).ToNot(HaveOccurred())
	})
})

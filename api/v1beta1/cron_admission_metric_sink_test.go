// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"context"
	"time"

	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
)

type cronAdmissionCountingSink struct {
	metrics.Sink
	created int
	updated int
	deleted int
}

func (s *cronAdmissionCountingSink) MetricValidationCreated([]string) error {
	s.created++
	return nil
}

func (s *cronAdmissionCountingSink) MetricValidationUpdated([]string) error {
	s.updated++
	return nil
}

func (s *cronAdmissionCountingSink) MetricValidationDeleted([]string) error {
	s.deleted++
	return nil
}

var _ = Describe("Cron admission metric routing", func() {
	DescribeTable("uses the cron sink for successful admission", func(operation string) {
		savedMetrics, savedCronMetrics := metricsSink, disruptionCronMetricsSink
		savedLog, savedRecorder := disruptionCronWebhookLogger, disruptionCronWebhookRecorder
		savedDeleteOnly, savedGroups := disruptionCronWebhookDeleteOnly, disruptionCronPermittedUserGroups
		savedFrequency := minimumCronFrequency
		DeferCleanup(func() {
			metricsSink, disruptionCronMetricsSink = savedMetrics, savedCronMetrics
			disruptionCronWebhookLogger, disruptionCronWebhookRecorder = savedLog, savedRecorder
			disruptionCronWebhookDeleteOnly, disruptionCronPermittedUserGroups = savedDeleteOnly, savedGroups
			minimumCronFrequency = savedFrequency
		})
		log := zap.NewNop().Sugar()
		mainSink := &cronAdmissionCountingSink{Sink: noop.New(log)}
		cronSink := &cronAdmissionCountingSink{Sink: noop.New(log)}
		metricsSink, disruptionCronMetricsSink = mainSink, cronSink
		disruptionCronWebhookLogger = log
		disruptionCronWebhookRecorder = record.NewFakeRecorder(10)
		disruptionCronWebhookDeleteOnly = false
		disruptionCronPermittedUserGroups = nil
		minimumCronFrequency = time.Second
		count := intstr.FromInt(1)
		cron := &DisruptionCron{
			ObjectMeta: metav1.ObjectMeta{Name: "schedule", Namespace: "ns"},
			Spec: DisruptionCronSpec{
				Schedule: "@every 1h", TargetResource: TargetResourceSpec{Kind: "deployment", Name: "app"},
				DisruptionTemplate: DisruptionSpec{Count: &count, Level: "pod", Duration: "1m", Network: &NetworkDisruptionSpec{Drop: 100}},
			},
		}
		want := [3]int{}
		var err error
		switch operation {
		case "create":
			_, err = cron.ValidateCreate(context.Background(), cron)
			want[0] = 1
		case "update":
			_, err = cron.ValidateUpdate(context.Background(), cron.DeepCopy(), cron)
			want[1] = 1
		case "delete":
			_, err = cron.ValidateDelete(context.Background(), cron)
			want[2] = 1
		}
		Expect(err).NotTo(HaveOccurred())
		Expect([3]int{cronSink.created, cronSink.updated, cronSink.deleted}).To(Equal(want))
		Expect([3]int{mainSink.created, mainSink.updated, mainSink.deleted}).To(Equal([3]int{}))
	},
		Entry("create", "create"),
		Entry("update", "update"),
		Entry("delete", "delete"),
	)
})

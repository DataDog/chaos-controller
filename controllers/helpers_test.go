// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Label Selector Validation", func() {
	Context("validating an empty label selector", func() {
		It("", func() {
			selector := labels.Set{}
			Expect(validateLabelSelector(selector.AsSelector())).ToNot(BeNil())
		})
	})
	Context("validating a good label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating special characters in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
})

var _ = Describe("Inject and CreatePods Trigger tests", func() {
	var creationTimestamp time.Time

	BeforeEach(func() {
		creationTimestamp = time.Now()
	})

	Context("TimeToCreatePods", func() {
		It("should return creationTimestamp if triggers is nil", func() {
			var triggers v1beta1.DisruptionTriggers

			Expect(TimeToCreatePods(&triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return creationTimestamp if triggers.createPods is nil", func() {
			triggers := &v1beta1.DisruptionTriggers{
				Inject: &v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
				CreatePods: nil,
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return createPods.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := &v1beta1.DisruptionTriggers{
				Inject: &v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
				CreatePods: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if createPods.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute * 5)
			triggers := &v1beta1.DisruptionTriggers{
				Inject: nil,
				CreatePods: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "5m",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(offsetTime))
		})
	})

	Context("TimeToInject", func() {
		It("should return creationTimestamp if triggers is nil", func() {
			var triggers v1beta1.DisruptionTriggers

			Expect(TimeToInject(&triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return triggers.createPods if triggers.inject is nil", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := &v1beta1.DisruptionTriggers{
				Inject: nil,
				CreatePods: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return inject.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := &v1beta1.DisruptionTriggers{
				Inject: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "1",
				},
				CreatePods: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "2m",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if inject.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute)
			triggers := &v1beta1.DisruptionTriggers{
				Inject: &v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "1m",
				},
				CreatePods: nil,
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(offsetTime))
		})
	})
})

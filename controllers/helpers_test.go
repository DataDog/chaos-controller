// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Label Selector Validation", func() {
	Context("validating an empty label selector", func() {
		It("", func() {
			selector := labels.Set{}
			Expect(validateLabelSelector(selector.AsSelector())).ToNot(Succeed())
		})
	})
	Context("validating a good label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(validateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating special characters in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(Succeed())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(Succeed())
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

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return creationTimestamp if triggers.createPods is nil", func() {
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return createPods.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					Offset: "15m",
				},
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToCreatePods(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if createPods.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute * 5)
			triggers := v1beta1.DisruptionTriggers{
				CreatePods: v1beta1.DisruptionTrigger{
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

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(creationTimestamp))
		})

		It("should return triggers.createPods if triggers.inject is nil", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return inject.notBefore if set", func() {
			notBefore := time.Now().Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					NotBefore: metav1.NewTime(notBefore),
					Offset:    "1",
				},
				CreatePods: v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "2m",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(notBefore))
		})

		It("should return a time after creationTimestamp if inject.offset is set", func() {
			offsetTime := creationTimestamp.Add(time.Minute)
			triggers := v1beta1.DisruptionTriggers{
				Inject: v1beta1.DisruptionTrigger{
					NotBefore: metav1.Time{},
					Offset:    "1m",
				},
			}

			Expect(TimeToInject(triggers, creationTimestamp)).Should(Equal(offsetTime))
		})
	})
})

var _ = DescribeTable(
	"disruptionTerminationStatus",
	func(disruption *disruptionBuilder, pods podsBuilder, expectTerminationStatus terminationStatus) {
		Expect(disruptionTerminationStatus(disruption.Build(), pods.Build())).To(Equal(expectTerminationStatus))
	},
	Entry(
		"not yet created disruption IS NOT terminated",
		newDisruptionBuilder().Reset(),
		nil,
		tsNotTerminated,
	),
	Entry(
		"1s before deadline, disruption IS NOT terminated",
		newDisruptionBuilder().WithCreation(time.Minute-time.Second),
		newPodsBuilder(),
		tsNotTerminated,
	),
	Entry(
		"1s after deadline, disruption IS definitively terminated",
		newDisruptionBuilder().WithCreation(time.Minute+time.Second),
		newPodsBuilder(),
		tsDefinitivelyTerminated,
	),
	Entry(
		"half duration disruption IS NOT terminated",
		newDisruptionBuilder(),
		newPodsBuilder(),
		tsNotTerminated,
	),
	Entry(
		"at deadline, disruption IS definitively terminated (however even ns before it is not)",
		newDisruptionBuilder().WithCreation(time.Minute),
		newPodsBuilder(),
		tsDefinitivelyTerminated,
	),
	Entry(
		"deleted disruption IS definitively terminated",
		newDisruptionBuilder().WithCreation(time.Minute).WithDeletion(),
		newPodsBuilder(),
		tsDefinitivelyTerminated,
	),
	Entry(
		"one chaos pod exited out of two IS NOT terminated",
		newDisruptionBuilder(),
		newPodsBuilder().One().Terminated().Parent(),
		tsNotTerminated,
	),
	Entry(
		"all chaos pods exited IS temporarily terminated",
		newDisruptionBuilder(),
		newPodsBuilder().One().Terminated().Parent().Two().Terminated().Parent(),
		tsTemporarilyTerminated,
	),
	Entry(
		"no pod injected is temporarily terminated",
		newDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusInjected),
		nil,
		tsTemporarilyTerminated,
	),
	Entry(
		"no pod partially injected is temporarily terminated",
		newDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusPartiallyInjected),
		nil,
		tsTemporarilyTerminated,
	),
	Entry(
		"no pod NOT injected is not terminated",
		newDisruptionBuilder().WithInjectionStatus(chaostypes.DisruptionInjectionStatusNotInjected),
		nil,
		tsNotTerminated,
	),
	Entry(
		"no pod initial injection status is not terminated",
		newDisruptionBuilder(),
		nil,
		tsNotTerminated,
	),
)

type disruptionBuilder struct {
	*chaosv1beta1.Disruption
	// we store action we want to perform instead of performing them right away because they are time sensititive
	// this enables us to ensure time.Now is as late as it can be without faking it (that we should do at some point)
	modifiers []func()
}

func newDisruptionBuilder() *disruptionBuilder {
	return (&disruptionBuilder{
		Disruption: &chaosv1beta1.Disruption{
			Spec: chaosv1beta1.DisruptionSpec{
				Duration: "1m", // per spec definition a valid disruption going to the reconcile loop MUST have a duration, let's not test wrong test cases
			},
		},
	}).WithCreation(30 * time.Second)
}

func (b *disruptionBuilder) Build() chaosv1beta1.Disruption {
	for _, modifier := range b.modifiers {
		modifier()
	}

	return *b.Disruption
}

func (b *disruptionBuilder) Reset() *disruptionBuilder {
	b.modifiers = nil

	return b
}

func (b *disruptionBuilder) WithCreation(past time.Duration) *disruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.CreationTimestamp = v1.NewTime(time.Now().Add(-past))
		})

	return b
}

func (b *disruptionBuilder) WithDeletion() *disruptionBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			v1t := v1.NewTime(time.Now())

			b.DeletionTimestamp = &v1t
		})

	return b
}

func (b *disruptionBuilder) WithInjectionStatus(status chaostypes.DisruptionInjectionStatus) *disruptionBuilder {
	b.Status.InjectionStatus = status

	return b
}

type podsBuilder []*podBuilder

type podBuilder struct {
	*corev1.Pod
	parent podsBuilder
}

func newPodsBuilder() podsBuilder {
	return podsBuilder{
		{
			Pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{},
						},
					},
				},
			},
		},
		{
			Pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{},
						},
					},
				},
			},
		},
	}
}

func (p podsBuilder) Build() []corev1.Pod {
	if p == nil {
		return nil
	}

	pods := make([]corev1.Pod, 0, len(p))

	for _, pod := range p {
		pods = append(pods, *pod.Pod)
	}

	return pods
}

func (p podsBuilder) Take(index int) *podBuilder {
	if p[index].parent == nil {
		p[index].parent = p
	}

	return p[index]
}

func (p podsBuilder) One() *podBuilder {
	return p.Take(0)
}

func (p podsBuilder) Two() *podBuilder {
	return p.Take(1)
}

func (p *podBuilder) Parent() podsBuilder {
	return p.parent
}

func (p *podBuilder) TerminatedWith(exitCode int32) *podBuilder {
	p.Pod.Status.ContainerStatuses[0].State.Terminated = &corev1.ContainerStateTerminated{
		ExitCode: exitCode,
	}

	return p
}

func (p *podBuilder) Terminated() *podBuilder {
	return p.TerminatedWith(0)
}

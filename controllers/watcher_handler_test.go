// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Cache Handler", func() {
	var (
		disruption v1beta1.Disruption
		targetPod  corev1.Pod
	)

	JustBeforeEach(func(ctx SpecContext) {
		disruption, targetPod, _ = InjectPodsAndDisruption(ctx, disruption, true)
		ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)
	})

	Context("verify events sent", func() {
		BeforeEach(func() {
			disruption = v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Annotations: map[string]string{v1beta1.SafemodeEnvironmentAnnotation: "lima"},
				},
				Spec: v1beta1.DisruptionSpec{
					DryRun: false,
					Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					Unsafemode: &v1beta1.UnsafemodeSpec{
						DisableAll: true,
					},
					StaticTargeting: false,
					Level:           chaostypes.DisruptionLevelPod,
					Network: &v1beta1.NetworkDisruptionSpec{
						Drop:    0,
						Corrupt: 0,
						Delay:   100,
					},
				},
			}
		})

		It("should target the pod", func(ctx SpecContext) {
			By("Ensuring that the injector pod has been created")
			ExpectChaosPods(ctx, disruption, 1)
		})

		It("should not fire any warning event on disruption", func(ctx SpecContext) {
			initialPodName := targetPod.Name

			By("deleting previously targeted pod")
			DeleteRunningPod(ctx, targetPod)

			By("waiting until disruption is considered NOT INJECTED")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusPausedInjected)

			By("creating a similar pod to target (same labels used by the disruption to target it, name will be different)")
			<-CreateRunningPod(ctx, targetPod)

			By("waiting until disruption is considered INJECTED on second pod")
			ExpectDisruptionStatus(ctx, disruption, chaostypes.DisruptionInjectionStatusInjected)

			By("deleting disruption")
			DeleteDisruption(ctx, disruption)

			By("disruption is now deleted, retrieving all namespace events")
			allNamespaceEvents := allNamespaceEvents(ctx)

			By("ensuring WARNING STATE of initial pod WAS NOT fired (we explicitely want such to be removed)")
			Expect(findEvent(v1beta1.EventTargetContainerWarningState, allNamespaceEvents, initialPodName)).To(BeZero())

			By("ensuring DISRUPTED event WAS fired for inital target")
			Expect(findEvent(v1beta1.EventDisrupted, allNamespaceEvents, initialPodName)).ToNot(BeZero())

			By("ensuring DISRUPTED event WAS fired for new target")
			Expect(findEvent(v1beta1.EventDisrupted, allNamespaceEvents, targetPod.Name)).ToNot(BeZero())
		})
	})
})

func allNamespaceEvents(ctx SpecContext) []corev1.Event {
	opts := client.ListOptions{
		Namespace: namespace,
	}
	eventList := corev1.EventList{}

	items := []corev1.Event{}
	for {
		Eventually(k8sClient.List).
			WithContext(ctx).WithArguments(&eventList, &opts).
			Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).
			Should(Succeed())
		items = append(items, eventList.Items...)

		if eventList.Continue == "" {
			break
		}

		opts.Continue = eventList.Continue
	}

	return items
}

func findEvent(eventKey v1beta1.DisruptionEventReason, events []corev1.Event, involvedObjectName string) corev1.Event {
	toFind := v1beta1.Events[eventKey]

	for _, event := range events {
		if toFind.Reason.MatchEventReason(event) && event.Type == toFind.Type && event.Source.Component == string(v1beta1.SourceDisruptionComponent) && event.InvolvedObject.Name == involvedObjectName {
			log.Infow("MATCHED", "event", event)
			return event
		} else {
			log.Infof("event: %s | %s %s %s %v", event.Reason, event.InvolvedObject.Name, event.Type, event.Source.Component, event.LastTimestamp.Time)
			log.Infow("NOT_MATCHED", "event", event, "to_find", toFind)
		}
	}

	return corev1.Event{}
}

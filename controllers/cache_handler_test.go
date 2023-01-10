// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package controllers

import (
	"fmt"
	"log"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func findEvent(disruptionTimestamp time.Time, toFind v1beta1.DisruptionEvent, events []v1.Event, involvedObjectName string) *v1.Event {
	for _, event := range events {
		if event.InvolvedObject.Name == involvedObjectName && disruptionTimestamp.Before(event.LastTimestamp.Time) {
			log.Printf("event: %s | %s %s %s", event.Reason, event.InvolvedObject.Name, event.Type, event.Source.Component)
		}
		if event.Reason == toFind.Reason && event.Type == toFind.Type && event.Source.Component == string(v1beta1.SourceDisruptionComponent) && event.InvolvedObject.Name == involvedObjectName {
			return &event
		}
	}

	return nil
}

var _ = Describe("Cache Handler verifications", func() {
	var disruption *v1beta1.Disruption
	var targetLabels map[string]string

	AfterEach(func() {
		// delete disruption resource
		_ = k8sClient.Delete(context.Background(), disruption)
		Eventually(func() error { return expectChaosPod(disruption, 0) }, timeout*2).Should(Succeed())
		Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
	})

	JustBeforeEach(func() {
		By("Creating disruption resource and waiting for injection to be done")
		Expect(k8sClient.Create(context.Background(), disruption)).To(BeNil())

		Eventually(func() error {
			// retrieve the previously created disruption
			d := v1beta1.Disruption{}
			if err := k8sClient.Get(context.Background(), instanceKey, &d); err != nil {
				return err
			}

			// check disruption injection status
			if d.Status.InjectionStatus != chaostypes.DisruptionInjectionStatusInjected {
				return fmt.Errorf("disruptions is not injected, current status is %s", d.Status.InjectionStatus)
			}

			return nil
		}, timeout).Should(Succeed())
	})

	Context("events sent verification", func() {
		BeforeEach(func() {
			targetLabels = targetPodA.Labels
			disruption = &v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: v1beta1.DisruptionSpec{
					DryRun: false,
					Count:  &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					Unsafemode: &v1beta1.UnsafemodeSpec{
						DisableAll: true,
					},
					StaticTargeting: false,
					Selector:        targetLabels,
					Level:           chaostypes.DisruptionLevelPod,
					Network: &v1beta1.NetworkDisruptionSpec{
						Drop:    0,
						Corrupt: 0,
						Delay:   100,
					},
				},
			}
		})

		It("should target the pod", func() {
			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return expectChaosPod(disruption, 1) }, timeout).Should(Succeed())
		})

		It("should not fire any warning event on disruption", func() {
			err := k8sClient.Delete(context.Background(), targetPodA)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(func() error {
				running, err := podsAreNotRunning(targetPodA)
				if err != nil {
					return err
				}

				if !running {
					return fmt.Errorf("target pods are running")
				}

				return nil
			}, timeout).Should(Succeed())

			targetPodA.ResourceVersion = ""
			Expect(k8sClient.Create(context.Background(), targetPodA)).To(BeNil())

			Eventually(func() error {
				running, err := podsAreRunning(targetPodA)
				if err != nil {
					return err
				}

				if !running {
					return fmt.Errorf("target pods are not running")
				}

				return nil
			}, timeout).Should(Succeed())

			eventList := v1.EventList{}

			err = k8sClient.List(context.Background(), &eventList, &client.ListOptions{
				Namespace: targetPodA.Namespace,
			})

			By("ensuring no error was thrown")
			Expect(err).To(BeNil())

			By("ensuring created event was fired")
			Eventually(func(g Gomega) {
				event := findEvent(disruption.CreationTimestamp.Time, v1beta1.Events[v1beta1.EventDisrupted], eventList.Items, targetPodA.Name)
				g.Expect(event).ShouldNot(BeNil())
			}).Should(Succeed())

			By("ensuring container target in warning state event was not fired")
			event := findEvent(disruption.CreationTimestamp.Time, v1beta1.Events[v1beta1.EventContainerWarningState], eventList.Items, targetPodA.Name)
			Expect(event).To(BeNil())
		})
	})
})

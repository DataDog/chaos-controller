// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package controllers

import (
	"fmt"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	clientsetv1beta1 "github.com/DataDog/chaos-controller/clientset/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
)

func setupDisruption(disruptionName, namespaceName string) v1beta1.Disruption {
	return v1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:        disruptionName,
			Namespace:   namespaceName,
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
}

func setupDisruptionCron(disruptionCronName, namespaceName string) v1beta1.DisruptionCron {
	return v1beta1.DisruptionCron{
		ObjectMeta: metav1.ObjectMeta{
			Name:        disruptionCronName,
			Namespace:   namespaceName,
			Annotations: map[string]string{v1beta1.SafemodeEnvironmentAnnotation: "lima"},
		},
		Spec: v1beta1.DisruptionCronSpec{
			Schedule: "*/15 * * * *",
			TargetResource: v1beta1.TargetResourceSpec{
				Kind: "deployment",
				Name: "test",
			},
			DisruptionTemplate: v1beta1.DisruptionSpec{
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
		},
	}
}

func createDisruption(ctx SpecContext, nsName string, dsName string) v1beta1.Disruption {
	disruption := setupDisruption(dsName, nsName)

	disruptionResult, _, _ := InjectPodsAndDisruption(ctx, disruption, true)
	ExpectDisruptionStatus(ctx, disruptionResult, chaostypes.DisruptionInjectionStatusInjected)

	return disruptionResult
}

func createDisruptionCron(ctx SpecContext, nsName string, dcName string) v1beta1.DisruptionCron {
	disruptionCron := setupDisruptionCron(dcName, nsName)

	Eventually(func(ctx SpecContext) error {
		return StopTryingNotRetryableKubernetesError(k8sClient.Create(ctx, &disruptionCron), true, false)
	}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed(), "Failed to create DisruptionCron")

	AddReportEntry(fmt.Sprintf("disruptioncron %s created at %v", disruptionCron.Name, disruptionCron.CreationTimestamp.Time), disruptionCron)

	DeferCleanup(func(ctx SpecContext, disruptionCron v1beta1.DisruptionCron) {
		Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&disruptionCron).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()), "Failed to delete DisruptionCron")
		Eventually(k8sClient.Get).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(types.NamespacedName{
			Namespace: disruptionCron.Namespace,
			Name:      disruptionCron.Name,
		}, &v1beta1.DisruptionCron{}).Should(WithTransform(errors.IsNotFound, BeTrue()), "DisruptionCron should be deleted")
	}, disruptionCron)

	return disruptionCron
}

var _ = Describe("Disruption Client", func() {
	var (
		clientset *clientsetv1beta1.Clientset
	)

	JustBeforeEach(func() {
		// Initialize the clientset before each test
		var err error
		clientset, err = clientsetv1beta1.NewForConfig(restConfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("List Method", func() {
		DescribeTable("should list disruptions correctly", func(ctx SpecContext, expectedDisruptionsCount int) {
			// Arrange
			namePrefix := "ds-list"
			for i := 1; i <= expectedDisruptionsCount; i++ {
				disruptionName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
				_ = createDisruption(ctx, namespace, disruptionName)
			}

			// Action
			ds, err := clientset.Chaos().Disruptions(namespace).List(ctx, metav1.ListOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while listing disruptions")

			var count int
			for _, item := range ds.Items {
				if strings.HasPrefix(item.Name, namePrefix) {
					count++
				}
			}
			Expect(count).To(Equal(expectedDisruptionsCount), "Mismatch in the number of expected disruptions")

		},
			Entry("when there are no disruptions in the cluster", 0, NodeTimeout(k8sAPIServerResponseTimeout)),
			Entry("when there is a single disruption in the cluster", 1, NodeTimeout(k8sAPIServerResponseTimeout)),
			//Entry("when there are three disruptions in the cluster", 3, NodeTimeout(k8sAPIServerResponseTimeout)), // TODO Skip("See CHAOSPLT-455: flaky test")
		)
	})

	Describe("Get Method", func() {
		DescribeTable("should retrieve a specific disruption successfully", func(ctx SpecContext, namePrefix string) {
			// Arrange
			disruptionName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
			_ = createDisruption(ctx, namespace, disruptionName)

			// Action
			d, err := clientset.Chaos().Disruptions(namespace).Get(ctx, disruptionName, metav1.GetOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while retrieving the disruption")
			Expect(d.Name).To(Equal(disruptionName), "Mismatch in the name of the retrieved disruption")
		},
			Entry("when a disruption exists in the cluster", "ds-get", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Create Method", func() {
		DescribeTable("should successfully create disruptions", func(ctx SpecContext, namePrefix string) {
			var (
				disruptionResult *v1beta1.Disruption
				err              error
			)
			// Arrange
			disruptionName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
			disruption := setupDisruption(disruptionName, namespace)
			disruption.Spec.Selector = map[string]string{"foo-foo": "bar-bar"}
			disruption.Spec.Duration = v1beta1.DisruptionDuration(lightCfg.Controller.DefaultDuration.String())

			// Action
			Eventually(func() error {
				disruptionResult, err = clientset.Chaos().Disruptions(namespace).Create(ctx, &disruption, metav1.CreateOptions{})
				return StopTryingNotRetryableKubernetesError(err, true, false)
			}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed(), "Error occurred while creating the disruption")

			DeferCleanup(DeleteDisruption, *disruptionResult)

			// Assert
			var fetchedDisruption v1beta1.Disruption
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: disruptionName, Namespace: namespace}, &fetchedDisruption)
			}, k8sAPIServerResponseTimeout, k8sAPIPotentialChangesEvery).Should(Succeed(), "Should eventually be able to retrieve the created disruption")
		},
			Entry("when creating a new disruption", "ds-create", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Delete Method", func() {
		DescribeTable("should successfully delete disruptions", func(ctx SpecContext, namePrefix string) {
			// Arrange
			disruptionName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
			_ = createDisruption(ctx, namespace, disruptionName)

			// Action
			err := clientset.Chaos().Disruptions(namespace).Delete(ctx, disruptionName, metav1.DeleteOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while deleting the disruption")

			Eventually(func() bool {
				var d v1beta1.Disruption
				err := k8sClient.Get(ctx, types.NamespacedName{Name: disruptionName, Namespace: namespace}, &d)
				return errors.IsNotFound(err)
			}, k8sAPIServerResponseTimeout, k8sAPIPotentialChangesEvery).Should(BeTrue(), "Disruption should be deleted from the cluster")

		},
			Entry("when deleting an existing disruption", "ds-delete", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})
	Describe("Watch Method", func() {
		var watcher watch.Interface

		JustBeforeEach(func() {
			// Create watcher for Disruptions
			var err error
			watcher, err = clientset.Chaos().Disruptions(namespace).Watch(suiteCtx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to start watching disruptions")

			DeferCleanup(watcher.Stop)
		})

		DescribeTable("should successfully capture events related to disruptions",
			func(ctx SpecContext, eventType watch.EventType, namePrefix string, configureDisruption func(ctx SpecContext, disruptionName string)) {
				// Arrange
				disruptionName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
				configureDisruption(ctx, disruptionName)

				// Assert
				Eventually(func() watch.Event {
					event := <-watcher.ResultChan()
					log.Debugw("received event from watcher", "type", event.Type, "object", event.Object)
					if event.Type == eventType {
						log.Debugw("received event matches expected event type", "expected", eventType, "received", event.Type, "object", event.Object)
						return event
					}
					return watch.Event{}
				}, k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(func(e watch.Event) bool {
					d, ok := e.Object.(*v1beta1.Disruption)
					return ok && d.Name == disruptionName && e.Type == eventType
				}, BeTrue()), "Expected to receive specific event type with correct disruption name")

			},
			Entry("when a disruption is added", watch.Added, "ds-watch-add", NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionName string) {
				_ = createDisruption(ctx, namespace, disruptionName)
			}),
			Entry("when a disruption is deleted", watch.Deleted, "ds-watch-delete", NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionName string) {
				disruption := createDisruption(ctx, namespace, disruptionName)

				Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&disruption).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()), "Failed to delete Disruption")
			}),
			Entry("when a disruption is updated", watch.Modified, "ds-watch-modify", NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionName string) {
				Skip("See CHAOSPLT-455: flaky test")
				_ = createDisruption(ctx, namespace, disruptionName)

				// Fetch the most up to date disruption
				var latestDisruption v1beta1.Disruption
				Eventually(k8sClient.Get).WithContext(ctx).WithArguments(types.NamespacedName{Name: disruptionName, Namespace: namespace}, &latestDisruption).Should(Succeed(), "Failed to fetch Disruption")

				// Update the disruption
				latestDisruption.Spec.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "30%"}

				// Update the disruption
				Eventually(k8sClient.Update).WithContext(ctx).WithArguments(&latestDisruption).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed(), "Failed to update Disruption")
			}),
		)
	})

})

var _ = Describe("DisruptionCron Client", func() {
	var (
		clientset *clientsetv1beta1.Clientset
	)

	JustBeforeEach(func() {
		// Initialize the clientset before each test
		var err error
		clientset, err = clientsetv1beta1.NewForConfig(restConfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("List Method", func() {
		DescribeTable("should list disruptioncrons correctly", func(ctx SpecContext, expectedDisruptionCronsCount int) {
			// Arrange
			namePrefix := "dc-list"
			for i := 1; i <= expectedDisruptionCronsCount; i++ {
				disruptionCronName := fmt.Sprintf("%s-%s", namePrefix, uuid.New().String())
				_ = createDisruptionCron(ctx, namespace, disruptionCronName)
			}

			// Action
			dcs, err := clientset.Chaos().DisruptionCrons(namespace).List(ctx, metav1.ListOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while listing disruptioncrons")

			var count int
			for _, item := range dcs.Items {
				if strings.HasPrefix(item.Name, namePrefix) {
					count++
				}
			}
			Expect(count).To(Equal(expectedDisruptionCronsCount), "Mismatch in the number of expected disruptions")
		},
			Entry("when there are no disruptioncrons in the cluster", 0, NodeTimeout(k8sAPIServerResponseTimeout)),
			Entry("when there is a single disruptioncron in the cluster", 1, NodeTimeout(k8sAPIServerResponseTimeout)),
			Entry("when there are multiple disruptioncrons in the cluster", 3, NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Get Method", func() {
		DescribeTable("should retrieve a specific disruptioncron successfully", func(ctx SpecContext) {
			// Arrange
			disruptionCronName := fmt.Sprintf("dicron-%s", uuid.New().String())
			_ = createDisruptionCron(ctx, namespace, disruptionCronName)

			// Action
			dc, err := clientset.Chaos().DisruptionCrons(namespace).Get(ctx, disruptionCronName, metav1.GetOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while retrieving the disruptioncron")
			Expect(dc.Name).To(Equal(disruptionCronName), "Mismatch in the name of the retrieved disruptioncron")

		},
			Entry("when a disruptioncron exists in the cluster", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Create Method", func() {
		DescribeTable("should successfully create disruptioncrons", func(ctx SpecContext) {
			// Arrange
			disruptionCronName := fmt.Sprintf("dicron-%s", uuid.New().String())
			disruptionCron := setupDisruptionCron(disruptionCronName, namespace)

			// Action
			dc, err := clientset.Chaos().DisruptionCrons(namespace).Create(ctx, &disruptionCron, metav1.CreateOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while creating the disruptioncron")
			Expect(dc.Name).To(Equal(disruptionCronName), "Mismatch in the name of the created disruptioncron")

			var fetchedDisruptionCron v1beta1.DisruptionCron
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: disruptionCronName, Namespace: namespace}, &fetchedDisruptionCron)
			}, k8sAPIServerResponseTimeout, k8sAPIPotentialChangesEvery).Should(Succeed(), "Should eventually be able to retrieve the created disruptionCron")

			Expect(fetchedDisruptionCron.Name).To(Equal(disruptionCronName), "Mismatch in the name of the fetched disruptionCron")

		},
			Entry("when creating a new disruptioncron", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Delete Method", func() {
		DescribeTable("should successfully delete disruptioncrons", func(ctx SpecContext) {
			// Arrange
			disruptionCronName := fmt.Sprintf("dicron-%s", uuid.New().String())
			_ = createDisruptionCron(ctx, namespace, disruptionCronName)

			// Action
			err := clientset.Chaos().DisruptionCrons(namespace).Delete(ctx, disruptionCronName, metav1.DeleteOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while deleting the disruptioncron")

			Eventually(func() bool {
				var dc v1beta1.DisruptionCron
				err := k8sClient.Get(ctx, types.NamespacedName{Name: disruptionCronName, Namespace: namespace}, &dc)
				return errors.IsNotFound(err)
			}, k8sAPIServerResponseTimeout, k8sAPIPotentialChangesEvery).Should(BeTrue(), "DisruptionCron should be deleted from the cluster")
		},
			Entry("when deleting an existing disruptioncron", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Watch Method", func() {
		var watcher watch.Interface

		JustBeforeEach(func() {
			// Create watcher for DisruptionCrons
			var err error
			watcher, err = clientset.Chaos().DisruptionCrons(namespace).Watch(suiteCtx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to start watching disruptioncrons")

			DeferCleanup(watcher.Stop)
		})

		DescribeTable("should successfully capture events related to disruptioncrons",
			func(ctx SpecContext, eventType watch.EventType, configureDisruptionCron func(ctx SpecContext, disruptionCronName string)) {
				// Arrange
				disruptionCronName := fmt.Sprintf("dicron-%s", uuid.New().String())
				configureDisruptionCron(ctx, disruptionCronName)

				// Assert
				Eventually(func() watch.Event {
					event := <-watcher.ResultChan()
					log.Debugw("received event from watcher", "type", event.Type, "object", event.Object)
					if event.Type == eventType {
						log.Debugw("received event matches expected event type", "expected", eventType, "received", event.Type, "object", event.Object)
						return event
					}
					return watch.Event{}
				}, k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(func(e watch.Event) bool {
					dc, ok := e.Object.(*v1beta1.DisruptionCron)
					return ok && dc.Name == disruptionCronName && e.Type == eventType
				}, BeTrue()), "Expected to receive specific event type with correct disruptioncron name")

			},
			Entry("when a disruptioncron is added", watch.Added, NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionCronName string) {
				_ = createDisruptionCron(ctx, namespace, disruptionCronName)
			}),
			Entry("when a disruptioncron is deleted", watch.Deleted, NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionCronName string) {
				disruptionCron := createDisruptionCron(ctx, namespace, disruptionCronName)

				Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&disruptionCron).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()), "Failed to delete DisruptionCron")
			}),
			Entry("when a disruptiocron is updated", watch.Modified, NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionCronName string) {
				Skip("See CHAOSPLT-455: flaky test")
				_ = createDisruptionCron(ctx, namespace, disruptionCronName)

				// Fetch the most up to date disruptioncron
				var latestDisruptionCron v1beta1.DisruptionCron
				Eventually(k8sClient.Get).WithContext(ctx).WithArguments(types.NamespacedName{Name: disruptionCronName, Namespace: namespace}, &latestDisruptionCron).Should(Succeed(), "Failed to fetch DisruptionCrib")

				// Update the disruptioncron
				latestDisruptionCron.Spec.DisruptionTemplate.Count = &intstr.IntOrString{Type: intstr.String, StrVal: "30%"}

				// Update the disruptioncron
				Eventually(k8sClient.Update).WithContext(ctx).WithArguments(&latestDisruptionCron).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed(), "Failed to update DisruptionCron")
			}),
		)
	})
})

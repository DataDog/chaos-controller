package controllers

import (
	"fmt"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	clientsetv1beta1 "github.com/DataDog/chaos-controller/clientset/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
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

func createDisruption(ctx SpecContext, nsName string, dsName string) v1beta1.Disruption {
	disruption := setupDisruption(dsName, nsName)

	disruptionResult, _, _ := InjectPodsAndDisruption(ctx, disruption, true)
	ExpectDisruptionStatus(ctx, disruptionResult, chaostypes.DisruptionInjectionStatusInjected)

	DeferCleanup(DeleteDisruption, disruptionResult)

	return disruptionResult
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
			namePrefix := "test-disruption-list"
			for i := 1; i <= expectedDisruptionsCount; i++ {
				disruptionName := fmt.Sprintf("%s%d", namePrefix, i)
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
			Entry("when there are three disruptions in the cluster", 3, NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Get Method", func() {
		DescribeTable("should retrieve a specific disruption successfully", func(ctx SpecContext, disruptionName string) {
			// Arrange
			_ = createDisruption(ctx, namespace, disruptionName)

			// Action
			d, err := clientset.Chaos().Disruptions(namespace).Get(ctx, disruptionName, metav1.GetOptions{})

			// Assert
			Expect(err).ShouldNot(HaveOccurred(), "Error occurred while retrieving the disruption")
			Expect(d.Name).To(Equal(disruptionName), "Mismatch in the name of the retrieved disruption")
		},
			Entry("when a disruption exists in the cluster", "test-disruption-get", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Create Method", func() {
		DescribeTable("should successfully create disruptions", func(ctx SpecContext, disruptionName string) {
			var (
				disruptionResult *v1beta1.Disruption
				err              error
			)
			// Arrange
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
			Entry("when creating a new disruption", "test-disruption-create", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Delete Method", func() {
		DescribeTable("should successfully delete disruptions", func(ctx SpecContext, disruptionName string) {
			// Arrange
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
			Entry("when deleting an existing disruption", "test-disruption-delete", NodeTimeout(k8sAPIServerResponseTimeout)),
		)
	})

	Describe("Watch Method", func() {
		DescribeTable("should successfully capture events related to disruptions",
			func(ctx SpecContext, eventType watch.EventType, disruptionName string, configureDisruption func(ctx SpecContext, disruptionName string)) {
				// Arrange
				configureDisruption(ctx, disruptionName)

				watcher, err := clientset.Chaos().Disruptions(namespace).Watch(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred(), "Failed to start watching disruptions")

				// Assert
				Eventually(func() watch.Event {
					select {
					case event := <-watcher.ResultChan():
						if event.Type == eventType {
							return event
						}
					default:
						return watch.Event{} // Return empty if no relevant event
					}
					return watch.Event{}
				}, k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(func(e watch.Event) bool {
					d, ok := e.Object.(*v1beta1.Disruption)
					return ok && d.Name == disruptionName && e.Type == eventType
				}, BeTrue()), "Expected to receive specific event type with correct disruption name")

			},
			Entry("when a disruption is added", watch.Added, "test-disruption-watch-add", NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionName string) {
				_ = createDisruption(ctx, namespace, disruptionName)
			}),
			Entry("when a disruption is deleted", watch.Deleted, "test-disruption-watch-delete", NodeTimeout(k8sAPIServerResponseTimeout), func(ctx SpecContext, disruptionName string) {
				disruption := createDisruption(ctx, namespace, disruptionName)

				Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&disruption).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()), "Failed to delete Disruption")
			}),
		)
	})

})

package controllers

import (
	"fmt"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/helpers"
	chaostypes "github.com/DataDog/chaos-fi-controller/types"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Disruption Controller", func() {
	var disruption *chaosv1beta1.Disruption
	var count int

	BeforeEach(func() {
		count = 1
		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Count:    &count,
				Selector: map[string]string{"foo": "bar"},
				NetworkFailure: &chaosv1beta1.NetworkFailureSpec{
					Host:        "127.0.0.1",
					Port:        80,
					Probability: 0,
					Protocol:    "tcp",
				},
			},
		}

		// patch
		monkey.Patch(helpers.GetContainerdID, func(pod *corev1.Pod) (string, error) {
			return "666", nil
		})
	})

	AfterEach(func() {
		k8sClient.Delete(context.Background(), disruption)
		monkey.UnpatchAll()
	})

	Context("nominal case", func() {
		It("should create the injection and cleanup pods", func() {
			By("Creating network failure resource")
			Expect(k8sClient.Create(context.Background(), disruption)).To(BeNil())

			By("Ensuring that the inject pod has been created")
			Eventually(func() error {
				l := corev1.PodList{}
				k8sClient.List(context.Background(), &l, &client.ListOptions{
					Namespace: "default",
					LabelSelector: labels.SelectorFromSet(map[string]string{
						chaostypes.TargetPodLabel: "foo-pod",
						chaostypes.PodModeLabel:   chaostypes.PodModeInject,
					}),
				})
				if len(l.Items) == 0 {
					return fmt.Errorf("empty list")
				}

				return nil
			}, timeout).Should(Succeed())

			By("Deleting the network failure resource")
			Expect(k8sClient.Delete(context.Background(), disruption)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(Succeed())

			By("Ensuring that the cleanup pod has been created")
			Eventually(func() error {
				l := corev1.PodList{}
				k8sClient.List(context.Background(), &l, &client.ListOptions{
					Namespace: "default",
					LabelSelector: labels.SelectorFromSet(map[string]string{
						chaostypes.TargetPodLabel: "foo-pod",
						chaostypes.PodModeLabel:   chaostypes.PodModeClean,
					}),
				})
				if len(l.Items) == 0 {
					return fmt.Errorf("empty list")
				}

				return nil
			}, timeout).Should(Succeed())

			By("Simulating the completion of the cleanup pod by removing the finalizer")
			Expect(k8sClient.Get(context.Background(), instanceKey, disruption)).To(BeNil())
			disruption.ObjectMeta.Finalizers = []string{}
			Eventually(func() error { return k8sClient.Update(context.Background(), disruption) }, timeout).Should(Succeed())

			By("Waiting for network failure resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, disruption) }, timeout).Should(MatchError("Disruption.chaos.datadoghq.com \"foo\" not found"))
		})
	})
})

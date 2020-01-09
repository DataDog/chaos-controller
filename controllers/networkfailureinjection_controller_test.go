package controllers

import (
	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/helpers"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	nfi = &chaosv1beta1.NetworkFailureInjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: chaosv1beta1.NetworkFailureInjectionSpec{
			Selector: map[string]string{"foo": "bar"},
			Failure: chaosv1beta1.NetworkFailureInjectionSpecFailure{
				Host:        "127.0.0.1",
				Port:        80,
				Probability: 0,
				Protocol:    "tcp",
			},
		},
	}
)

var _ = Describe("NetworkFailureInjection Controller", func() {
	var injectPodKey, cleanupPodKey types.NamespacedName
	var injectPod, cleanupPod *corev1.Pod

	BeforeEach(func() {
		injectPodKey = types.NamespacedName{Name: "foo-inject-foo-pod-pod", Namespace: "default"}
		cleanupPodKey = types.NamespacedName{Name: "foo-cleanup-foo-pod-pod", Namespace: "default"}
		injectPod = &corev1.Pod{}
		cleanupPod = &corev1.Pod{}

		logf.Log.Info("patching helpers.GetContainerdID")
		monkey.Patch(helpers.GetContainerdID, func(pod *corev1.Pod) (string, error) {
			return "666", nil
		})
	})

	AfterEach(func() {
		k8sClient.Delete(context.Background(), injectPod)
		k8sClient.Delete(context.Background(), cleanupPod)
		k8sClient.Delete(context.Background(), nfi)
		monkey.UnpatchAll()
	})

	Context("nominal case", func() {
		It("should create the injection and cleanup pods", func() {
			By("Creating network failure resource")
			Expect(k8sClient.Create(context.Background(), nfi)).To(BeNil())

			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return k8sClient.Get(context.Background(), injectPodKey, injectPod) }, timeout).Should(Succeed())

			By("Deleting the network failure resource")
			Expect(k8sClient.Delete(context.Background(), nfi)).To(BeNil())
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, nfi) }, timeout).Should(Succeed())

			By("Ensuring that the cleanup pod has been created")
			Eventually(func() error { return k8sClient.Get(context.Background(), cleanupPodKey, cleanupPod) }, timeout).Should(Succeed())

			By("Simulating the completion of the cleanup pod by removing the finalizer")
			Expect(k8sClient.Get(context.Background(), instanceKey, nfi)).To(BeNil())
			nfi.ObjectMeta.Finalizers = []string{}
			Expect(k8sClient.Update(context.Background(), nfi)).To(BeNil())

			By("Waiting for network failure resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, nfi) }, timeout).Should(MatchError("NetworkFailureInjection.chaos.datadoghq.com \"foo\" not found"))
		})
	})
})

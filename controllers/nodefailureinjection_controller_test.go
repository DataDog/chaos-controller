package controllers

import (
	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/api/v1beta1"
	"github.com/DataDog/chaos-fi-controller/datadog"
	"github.com/DataDog/datadog-go/statsd"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	nofi = &chaosv1beta1.NodeFailureInjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: chaosv1beta1.NodeFailureInjectionSpec{
			Selector: map[string]string{"foo": "bar"},
		},
	}
)

var _ = Describe("NodeFailureInjection Controller", func() {
	var injectPodKey types.NamespacedName
	var injectPod *corev1.Pod

	BeforeEach(func() {
		injectPodKey = types.NamespacedName{Name: "foo-foo-pod", Namespace: "default"}
		injectPod = &corev1.Pod{}

		logf.Log.Info("patching datadog instance")
		monkey.Patch(datadog.GetInstance, func() *statsd.Client {
			return nil
		})
	})

	AfterEach(func() {
		k8sClient.Delete(context.Background(), injectPod)
		k8sClient.Delete(context.Background(), nofi)
		monkey.UnpatchAll()
	})

	Context("nominal case", func() {
		It("should create the injection pod", func() {
			By("Creating node failure resource")
			Expect(k8sClient.Create(context.Background(), nofi)).To(BeNil())

			By("Ensuring that the inject pod has been created")
			Eventually(func() error { return k8sClient.Get(context.Background(), injectPodKey, injectPod) }, timeout).Should(Succeed())

			By("Deleting the node failure resource")
			Expect(k8sClient.Delete(context.Background(), nofi)).To(BeNil())

			By("Waiting for node failure resource to be deleted")
			Eventually(func() error { return k8sClient.Get(context.Background(), instanceKey, nofi) }, timeout).Should(MatchError("NodeFailureInjection.chaos.datadoghq.com \"foo\" not found"))
		})
	})
})

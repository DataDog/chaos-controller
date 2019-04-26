/*
Copyright 2019 Datadog.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package networkfailureinjection

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/datadog-go/statsd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var _ = Describe("Test Reconcile", func() {
	const timeout = time.Second * 30

	var (
		targetPod      *corev1.Pod
		otherTargetPod *corev1.Pod
		instance       *chaosv1beta1.NetworkFailureInjection

		// the inject pod for targetPod
		injectPod *corev1.Pod
		// the cleanup pod for targetPod
		cleanupPod *corev1.Pod

		// set on each invocation of a spec if it needs to be used
		childPod     *corev1.Pod
		selectedPods []corev1.Pod
		r            *ReconcileNetworkFailureInjection

		// request for instance, to pass to Reconcile
		nfiRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}

		// key for calling client.Get
		// key for instance
		nfiKey = types.NamespacedName{Name: "foo", Namespace: "default"}
		// key for injectPod
		injectPodKey = types.NamespacedName{Name: "foo-inject-foo-pod-pod", Namespace: "default"}
		// key for cleanup pod
		cleanupPodKey = types.NamespacedName{Name: "foo-cleanup-foo-pod-pod", Namespace: "default"}

		nodeName = "foo-node"
	)

	BeforeEach(func() {
		// Initialize objects to create
		targetPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-pod",
				Namespace: "default",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: corev1.PodSpec{
				NodeName:      nodeName,
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					corev1.Container{
						Name:  "foo-container",
						Image: "bash",
					},
				},
			},
		}

		otherTargetPod = &corev1.Pod{}
		targetPod.DeepCopyInto(otherTargetPod)
		otherTargetPod.Name = "foo-pod-2"

		injectPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-inject-foo-pod-pod",
				Namespace: targetPod.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						UID: "666",
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName:      nodeName,
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					corev1.Container{
						Name:  "foo-container",
						Image: "bash",
					},
				},
			},
		}

		cleanupPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-cleanup-foo-pod-pod",
				Namespace: targetPod.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						UID: "666",
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName:      nodeName,
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:  cleanupContainerName,
						Image: "bash",
					},
				},
			},
		}

		instance = &chaosv1beta1.NetworkFailureInjection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
				UID:       "666",
			},
			Spec: chaosv1beta1.NetworkFailureInjectionSpec{
				Selector: map[string]string{"foo": "bar"},
				Failure: chaosv1beta1.NetworkFailureInjectionSpecFailure{
					Host:        "127.0.0.1",
					Port:        666,
					Probability: 0,
					Protocol:    "tcp",
				},
			},
		}

		// os
		monkey.Patch(os.Getenv, func(key string) string {
			return "foo"
		})

		// statsd
		var statsdClient *statsd.Client
		monkey.Patch(statsd.New, func(addr string, options ...statsd.Option) (*statsd.Client, error) {
			return statsdClient, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(statsdClient), "Incr", func(client *statsd.Client, name string, tags []string, rate float64) error {
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(statsdClient), "Timing", func(client *statsd.Client, name string, value time.Duration, tags []string, rate float64) error {
			return nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(statsdClient), "Event", func(client *statsd.Client, e *statsd.Event) error {
			return nil
		})

		// ReconcileNetworkFailureInjection Reconciler
		monkey.PatchInstanceMethod(reflect.TypeOf(r), "GetContainerdID", func(r *ReconcileNetworkFailureInjection, pod *corev1.Pod) (string, error) {
			return "666", nil
		})

		monkey.PatchInstanceMethod(reflect.TypeOf(r), "SelectPodsForInjection", func(r *ReconcileNetworkFailureInjection, instance *chaosv1beta1.NetworkFailureInjection) (*corev1.PodList, error) {
			// Fail on empty label selectors
			if len(instance.Spec.Selector) < 1 || instance.Spec.Selector == nil {
				err := fmt.Errorf("nfi \"%s\" in namespace \"%s\" is missing a label selector", instance.Name, instance.Namespace)
				return nil, err
			}

			// Update the status
			for _, pod := range selectedPods {
				instance.Status.Pods = append(instance.Status.Pods, pod.Name)
				err := r.Update(context.Background(), instance)
				if err != nil {
					return nil, err
				}
			}
			return &corev1.PodList{Items: selectedPods}, nil
		})

		monkey.PatchInstanceMethod(reflect.TypeOf(r), "MakePod", func(r *ReconcileNetworkFailureInjection, instance *chaosv1beta1.NetworkFailureInjection, p *corev1.Pod, containerID, podType string) (*corev1.Pod, error) {
			return childPod, nil
		})

		childPod = nil
		selectedPods = nil
		r = nil
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("creating an nfi", func() {
		It("should correctly create an inject pod when a pod matching the label selector in the same namespace exists", func() {
			c = fakeClient(targetPod, instance)
			r = &ReconcileNetworkFailureInjection{
				Client:   c,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(5),
			}
			selectedPods = []corev1.Pod{*targetPod}
			childPod = injectPod

			_, err := r.Reconcile(nfiRequest)
			Expect(err).NotTo(HaveOccurred())

			r.Get(context.TODO(), nfiKey, instance)

			By("Ensuring the inject pod is created")
			p := &corev1.Pod{}
			Expect(r.Get(context.TODO(), injectPodKey, p)).NotTo(HaveOccurred())
			Expect(containsString(instance.Finalizers, cleanupFinalizer)).To(BeTrue(), "cleanup finalizer should be set on creation")
			Expect(p.OwnerReferences[0].UID == instance.UID).Should(BeTrue(), "inject pod's owner reference was set correctly")

			By("Ensuring the nfi's status.pods is updated with the target pod's name")
			Expect(containsString(instance.Status.Pods, targetPod.Name)).Should(BeTrue(), ".status.pods updated")

			By("Ensuring the status.injected field is set on successful creation of all injected pods")
			Expect(instance.Status.Injected == true).To(BeTrue())
		})

		It("should not create an inject pod if no pod matching the label selectors in the same namespace exists", func() {
			targetPod.Labels = map[string]string{}
			c = fakeClient(targetPod, instance)
			r = &ReconcileNetworkFailureInjection{
				Client:   c,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(5),
			}

			_, err := r.Reconcile(nfiRequest)
			Expect(err).NotTo(HaveOccurred())

			p := &corev1.Pod{}
			Expect(r.Get(context.TODO(), injectPodKey, p)).To(HaveOccurred())
		})
	})

	Describe("selecting target pods", func() {
		It("should not allow an empty label selector", func() {
			instance.Spec.Selector = map[string]string{}

			c = fakeClient(targetPod, instance)
			r = &ReconcileNetworkFailureInjection{
				Client:   c,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(5),
			}

			_, err := r.Reconcile(nfiRequest)
			Expect(err).Should(MatchError("nfi \"foo\" in namespace \"default\" is missing a label selector"))
		})

		// Random pod selection
		It("should only select numPodsToTarget when it is specified", func() {
			n := 1
			instance.Spec.NumPodsToTarget = &n
			selectedPods = []corev1.Pod{*targetPod}
			childPod = injectPod
			c = fakeClient(targetPod, instance)

			r = &ReconcileNetworkFailureInjection{
				Client:   c,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(5),
			}
			_, err := r.Reconcile(nfiRequest)
			Expect(err).NotTo(HaveOccurred())

			r.Get(context.TODO(), nfiKey, instance)
			Expect(len(instance.Status.Pods) == n).To(BeTrue(), "status should reflect numPodsToTarget")
		})
	})

	Describe("deleting an nfi", func() {
		It("should have its status updated and create a cleanup pod", func() {
			instance.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
			instance.Finalizers = append(instance.Finalizers, cleanupFinalizer)
			instance.Status.Pods = append(instance.Status.Pods, targetPod.Name)
			childPod = cleanupPod

			c = fakeClient(instance, targetPod)

			r = &ReconcileNetworkFailureInjection{
				Client:   c,
				scheme:   scheme.Scheme,
				recorder: record.NewFakeRecorder(5),
			}
			_, err := r.Reconcile(nfiRequest)
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the status.finalizing is set when a user tries to delete an nfi")
			r.Get(context.TODO(), nfiKey, instance)
			Expect(instance.Status.Finalizing).To(BeTrue())

			By("Ensuring a cleanup pod is created")
			c := &corev1.Pod{}
			Expect(r.Get(context.TODO(), cleanupPodKey, c)).NotTo(HaveOccurred())
		})
	})
})

func fakeClient(initObjs ...runtime.Object) client.Client {
	return fake.NewFakeClientWithScheme(scheme.Scheme, initObjs...)
}

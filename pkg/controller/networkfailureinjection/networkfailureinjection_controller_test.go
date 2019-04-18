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
	"fmt"
	"os"
	"reflect"
	"time"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/datadog-go/statsd"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

// TODO: add test for checking that nfi only targets pods in same ns

var _ = Describe("Test Reconcile", func() {
	const timeout = time.Second * 30

	var (
		// The pod the nfi will target
		targetPod *corev1.Pod
		// The nfi to create
		instance *chaosv1beta1.NetworkFailureInjection
		// The ReconcileNetworkFailureInjection to be used throughout tests due to patching
		r                  *ReconcileNetworkFailureInjection
		expectedNfiRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
		nfiKey             = types.NamespacedName{Name: "foo", Namespace: "default"}
		injectPodKey       = types.NamespacedName{Name: "foo-inject-foo-pod-pod", Namespace: "default"}
		cleanupPodKey      = types.NamespacedName{Name: "foo-cleanup-foo-pod-pod", Namespace: "default"}
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
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					corev1.Container{
						Name:  "foo-container",
						Image: "bash",
					},
				},
			},
		}

		instance = &chaosv1beta1.NetworkFailureInjection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
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

		// Use bash for image with Never RestartPolicy so the pods run to completion
		monkey.PatchInstanceMethod(reflect.TypeOf(r), "MakeInjectPod", func(r *ReconcileNetworkFailureInjection, instance *chaosv1beta1.NetworkFailureInjection, p *corev1.Pod, containerID string) (*corev1.Pod, error) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name + "-inject-" + p.Name + "-pod",
					Namespace: instance.Namespace,
				},
				Spec: corev1.PodSpec{
					NodeName:      p.Spec.NodeName,
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:  "chaos-fi-inject",
							Image: "bash",
						},
					},
				},
			}
			return pod, nil
		})

		monkey.PatchInstanceMethod(reflect.TypeOf(r), "MakeCleanupPod", func(r *ReconcileNetworkFailureInjection, instance *chaosv1beta1.NetworkFailureInjection, p *corev1.Pod, containerID string) (*corev1.Pod, error) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name + "-cleanup-" + p.Name + "-pod",
					Namespace: instance.Namespace,
				},
				Spec: corev1.PodSpec{
					NodeName:      p.Spec.NodeName,
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:  cleanupContainerName,
							Image: "bash",
							// Command: []string{"sleep 300"},
						},
					},
				},
			}
			return pod, nil
		})
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("creating an nfi targeting a pod", func() {
		// Note: If spec fails and cleanup in deferred functions fail,
		// gomega output will not be printed
		It("should create an inject and cleanup pod before deleting the nfi", func() {
			By("Setting up the manager and controller")
			var err error
			testMgr, err := manager.New(cfg, manager.Options{})
			Expect(err).NotTo(HaveOccurred())
			c = testMgr.GetClient()

			r = newReconciler(testMgr).(*ReconcileNetworkFailureInjection)
			recFn, requests := SetupTestReconcile(r)
			Expect(add(testMgr, recFn)).NotTo(HaveOccurred())

			stopMgr, mgrStopped := StartTestManager(testMgr)

			defer func() {
				close(stopMgr)
				mgrStopped.Wait()
			}()

			By("Creating a target pod for the nfi")
			err = c.Create(context.TODO(), targetPod)
			defer c.Delete(context.TODO(), targetPod)
			// The instance object may not be a valid object because it might be missing some required fields.
			// Please modify the instance object by adding required fields and then remove the following if statement.
			Expect(apierrors.IsInvalid(err)).NotTo(BeTrue(), "%v", err)
			Expect(err).NotTo(HaveOccurred())

			By("Creating an nfi targeting the created pod")
			err = c.Create(context.TODO(), instance)
			// Ensure we remove the nfi if anything fails, which requires the finalizer to be removed
			defer func() {
				defer GinkgoRecover()
				GinkgoWriter.Write([]byte("Cleanup: Ensuring the nfi is deleted since garbage collection isn't enabled in the test control plane\n"))
				c.Get(context.TODO(), nfiKey, instance)
				instance.Finalizers = removeString(instance.Finalizers, cleanupFinalizer)
				c.Update(context.TODO(), instance)
				Eventually(func() error {
					return c.Delete(context.TODO(), instance)
				}, timeout).
					Should(MatchError("networkfailureinjections.chaos.datadoghq.com \"foo\" not found"), "failed to delete nfi foo during cleanup")
				GinkgoWriter.Write([]byte("Cleanup: nfi was deleted\n"))
			}()
			// The instance object may not be a valid object because it might be missing some required fields.
			// Please modify the instance object by adding required fields and then remove the following if statement.
			Expect(apierrors.IsInvalid(err)).NotTo(BeTrue(), "%v", err)
			Expect(err).NotTo(HaveOccurred())

			By("Expecting the Reconciler to receive a Request for the nfi creation")
			Eventually(requests, timeout).Should(Receive(Equal(expectedNfiRequest)))

			By("Expecting an inject pod to be created for the targeted pod")
			injectPod := &corev1.Pod{}
			Eventually(func() error { return c.Get(context.TODO(), injectPodKey, injectPod) }, timeout).
				Should(Succeed())
			defer func() {
				defer GinkgoRecover()
				GinkgoWriter.Write([]byte("Cleanup: Ensuring the inject pod is deleted since garbage collection isn't enabled in the test control plane\n"))
				Eventually(func() error {
					return c.Delete(context.TODO(), injectPod)
				}, timeout).
					Should(MatchError("pods \"foo-inject-foo-pod-pod\" not found"))
				GinkgoWriter.Write([]byte("Cleanup: Inject pod was deleted\n"))
			}()

			By("Expecting the cleanup finalizer to be added to the nfi")
			Eventually(func() bool {
				nfi := &chaosv1beta1.NetworkFailureInjection{}
				err := c.Get(context.TODO(), nfiKey, nfi)
				if err != nil {
					return false
				}
				return containsString(nfi.Finalizers, cleanupFinalizer)
			}, timeout).Should(BeTrue(), "cleanup finalizer should be added to the nfi")

			// Delete the nfi and expect Reconcile to be called for the NFI deletion,
			// and expect the cleanup pod to be created
			By("Deleting the NFI")
			Expect(c.Delete(context.TODO(), instance)).NotTo(HaveOccurred())

			By("Expecting Reconcile to receive a Request for the NFI deletion")
			Eventually(requests, timeout).Should(Receive(Equal(expectedNfiRequest)))
			nfi := &chaosv1beta1.NetworkFailureInjection{}

			By("Expecting the nfi to not have been deleted (finalizer not yet removed)")
			Eventually(func() error { return c.Get(context.TODO(), nfiKey, nfi) }, timeout).
				Should(Succeed())

			By("Expecting the nfi's finalizing status to be set to true")
			Eventually(func() bool {
				c.Get(context.TODO(), nfiKey, nfi)
				return nfi.Status.Finalizing
			}, timeout).Should(BeTrue(), "nfi finalizing status should be true")

			By("Expecting a cleanup pod to be created for the target pod")
			cleanupPod := &corev1.Pod{}
			Eventually(func() error { return c.Get(context.TODO(), cleanupPodKey, cleanupPod) }, timeout).
				Should(Succeed())
			defer func() {
				defer GinkgoRecover()
				GinkgoWriter.Write([]byte("Cleanup: Ensuring the cleanup pod is deleted since garbage collection isn't enabled in the test control plane\n"))
				Eventually(func() error {
					return c.Delete(context.TODO(), cleanupPod)
				}, timeout).
					Should(MatchError("pods \"foo-cleanup-foo-pod-pod\" not found"))
				GinkgoWriter.Write([]byte("Cleanup: Cleanup pod was deleted\n"))
			}()

			By("Expecting the cleanup pod to complete")
			Eventually(func() error {
				cleanupPod := &corev1.Pod{}
				err = c.Get(context.TODO(), cleanupPodKey, cleanupPod)
				if err != nil {
					return err
				}
				if cleanupPod.Status.Phase != corev1.PodSucceeded {
					return fmt.Errorf("cleanup pod '%s' did not complete", cleanupPod.Name)
				}
				return nil
			}, timeout).Should(Succeed())

			By("Expecting Reconcile to receive a Request")
			Eventually(requests, timeout).Should(Receive(Equal(expectedNfiRequest)))

			By("Expecting the nfi to be deleted once the cleanup pod completes")
			Eventually(func() bool {
				err := c.Get(context.TODO(), nfiKey, instance)
				return apierrors.IsNotFound(err)
			}, timeout).Should(BeTrue(), "nfi should be deleted once cleanup pod completes")
		})
	})
})

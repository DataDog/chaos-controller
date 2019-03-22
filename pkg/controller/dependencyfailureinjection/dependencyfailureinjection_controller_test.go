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

package dependencyfailureinjection

import (
	"testing"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/onsi/gomega"
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

var (
	expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
	dfiKey          = types.NamespacedName{Name: "foo", Namespace: "default"}
	injectPodKey    = types.NamespacedName{Name: "foo-inject-foo-pod-pod", Namespace: "default"}
	cleanupPodKey   = types.NamespacedName{Name: "foo-cleanup-foo-pod-pod", Namespace: "default"}
)

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create a pod for the DependencyFailureInjection to target
	targetPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-pod",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
	err = c.Create(context.TODO(), targetPod)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), targetPod)

	// Create the DependencyFailureInjection object and expect the Reconcile request and inject pod to be created
	instance := &chaosv1beta1.DependencyFailureInjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: chaosv1beta1.DependencyFailureInjectionSpec{
			LabelSelector: "foo=bar",
			Failure: chaosv1beta1.DependencyFailureInjectionSpecFailure{
				Host:        "127.0.0.1",
				Port:        80,
				Probability: 0,
				Protocol:    "tcp",
			},
		},
	}
	err = c.Create(context.TODO(), instance)
	// The instance object may not be a valid object because it might be missing some required fields.
	// Please modify the instance object by adding required fields and then remove the following if statement.
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	injectPod := &corev1.Pod{}
	g.Eventually(func() error { return c.Get(context.TODO(), injectPodKey, injectPod) }, timeout).
		Should(gomega.Succeed())
	defer c.Delete(context.TODO(), injectPod)

	// Delete the DFI and expect Reconcile to be called for the DFI deletion,
	// and expect the cleanup pod to be created
	g.Expect(c.Delete(context.TODO(), instance)).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), dfiKey, instance) }, timeout).
		Should(gomega.Succeed())
	cleanupPod := &corev1.Pod{}
	g.Eventually(func() error { return c.Get(context.TODO(), cleanupPodKey, cleanupPod) }, timeout).
		Should(gomega.Succeed())
	defer c.Delete(context.TODO(), cleanupPod)

	// Manually delete DFI since GC isn't enabled in the test control plane
	g.Eventually(func() error { return c.Delete(context.TODO(), instance) }, timeout).
		Should(gomega.MatchError("DependencyFailureInjection \"foo\" not found"))

}

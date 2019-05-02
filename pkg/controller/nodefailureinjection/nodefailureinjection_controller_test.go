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

package nodefailureinjection

import (
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	chaosv1beta1 "github.com/DataDog/chaos-fi-controller/pkg/apis/chaos/v1beta1"
	"github.com/DataDog/chaos-fi-controller/pkg/datadog"
	"github.com/DataDog/chaos-fi-controller/pkg/helpers"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "foo", Namespace: "default"}}
var podKey = types.NamespacedName{Name: "foo-bar-abcd", Namespace: "default"}
var instanceKey = types.NamespacedName{Name: "foo", Namespace: "default"}

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	instance := &chaosv1beta1.NodeFailureInjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: chaosv1beta1.NodeFailureInjectionSpec{
			Selector: labels.Set{},
		},
	}

	// Patch
	defer monkey.UnpatchAll()
	var guard *monkey.PatchGuard
	// TODO: create a pod instead of patching
	monkey.Patch(helpers.GetMatchingPods, func(client.Client, string, labels.Set) (*corev1.PodList, error) {
		return &corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar-abcd",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						NodeName: "foo-node",
					},
				},
			},
		}, nil
	})
	guard = monkey.Patch(os.Getenv, func(key string) string {
		guard.Unpatch()
		defer guard.Restore()

		if key == helpers.ChaosFailureInjectionImageVariableName {
			return "foo:bar"
		}

		return os.Getenv(key)
	})
	monkey.Patch(datadog.GetInstance, func() *statsd.Client {
		return nil
	})

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

	// Create the NodeFailureInjection object and expect the Reconcile and pod to be created
	err = c.Create(context.TODO(), instance)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))

	pod := &corev1.Pod{}
	g.Eventually(func() error { return c.Get(context.TODO(), podKey, pod) }, timeout).
		Should(gomega.Succeed())
	g.Expect(c.Get(context.TODO(), instanceKey, instance)).NotTo(gomega.HaveOccurred())
	g.Expect(instance.Status.Injected).To(gomega.Equal(1))

	// Delete the pod and expect Reconcile to be called for pod deletion
	g.Expect(c.Delete(context.TODO(), pod)).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), podKey, pod) }, timeout).
		Should(gomega.Succeed())
	g.Expect(c.Get(context.TODO(), instanceKey, instance)).NotTo(gomega.HaveOccurred())
	g.Expect(instance.Status.Injected).To(gomega.Equal(1))

	// Manually delete pod since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), instance)).To(gomega.BeNil())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
	g.Eventually(func() error { return c.Get(context.TODO(), instanceKey, instance) }, timeout).
		Should(gomega.MatchError("NodeFailureInjection.chaos.datadoghq.com \"foo\" not found"))
	g.Expect(c.Delete(context.TODO(), pod)).To(gomega.BeNil())
}

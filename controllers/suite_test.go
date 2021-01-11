// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

/*

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

package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/metrics"
	metricstypes "github.com/DataDog/chaos-controller/metrics/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	timeout = time.Second * 10
)

var (
	cfg         *rest.Config
	k8sClient   client.Client
	k8sManager  ctrl.Manager
	testEnv     *envtest.Environment
	instanceKey types.NamespacedName
	targetPodA  *corev1.Pod
	targetPodB  *corev1.Pod
	targetPodC  *corev1.Pod
	targetPodD  *corev1.Pod
)

type fakeK8sClient struct {
	realClient client.Client
}

// Get adds a fake container ID to retrieved pods so injection and cleanup can be done
func (f fakeK8sClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	// load object
	err := f.realClient.Get(ctx, key, obj)
	if err != nil {
		return err
	}

	// try to convert given object into a pod
	if pod, ok := obj.(*corev1.Pod); ok {
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				ContainerID: "fakeID",
			},
		}
	}

	return nil
}
func (f fakeK8sClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return f.realClient.List(ctx, list, opts...)
}
func (f fakeK8sClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	return f.realClient.Create(ctx, obj, opts...)
}
func (f fakeK8sClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	return f.realClient.Delete(ctx, obj, opts...)
}
func (f fakeK8sClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return f.realClient.Update(ctx, obj, opts...)
}
func (f fakeK8sClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return f.realClient.Patch(ctx, obj, patch, opts...)
}
func (f fakeK8sClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return f.realClient.DeleteAllOf(ctx, obj, opts...)
}
func (f fakeK8sClient) Status() client.StatusWriter {
	return f.realClient.Status()
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "config", "crd", "bases")},
		KubeAPIServerFlags: append(envtest.DefaultKubeAPIServerFlags, "--allow-privileged"),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = chaosv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = chaosv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = chaosv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = chaosv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	k8sClient = fakeK8sClient{
		realClient: k8sManager.GetClient(),
	}

	ms, err := metrics.GetSink(metricstypes.SinkDriverNoop, metricstypes.SinkAppController)
	Expect(err).ToNot(HaveOccurred())

	err = (&DisruptionReconciler{
		Client:      k8sClient,
		Log:         ctrl.Log.WithName("controllers").WithName("Disruption"),
		Recorder:    k8sManager.GetEventRecorderFor("disruption-controller"),
		MetricsSink: ms,
		Scheme:      scheme.Scheme,
		PodTemplateSpec: corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "chaos",
					},
				},
			},
		},
		TargetSelector: MockTargetSelector{},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// start the manager and wait for a few seconds for it
	// to be ready to deal with watched resources
	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
	time.Sleep(5 * time.Second)

	close(done)
}, 60)

var _ = BeforeEach(func() {
	instanceKey = types.NamespacedName{Name: "foo", Namespace: "default"}
	targetPodA = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Image: "foo",
					Name:  "foo",
				},
			},
		},
	}
	targetPodB = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bar",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Image: "bar",
					Name:  "bar",
				},
			},
		},
	}
	targetPodC = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "car",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Image: "car",
					Name:  "car",
				},
			},
		},
	}
	targetPodD = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "far",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Image: "far",
					Name:  "far",
				},
			},
		},
	}

	By("Creating target pod")
	err := k8sClient.Create(context.Background(), targetPodA)
	if apierrors.IsInvalid(err) {
		logf.Log.Error(err, "failed to create object, got an invalid object error")
		return
	}
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Create(context.Background(), targetPodB)
	if apierrors.IsInvalid(err) {
		logf.Log.Error(err, "failed to create object, got an invalid object error")
		return
	}
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Create(context.Background(), targetPodC)
	if apierrors.IsInvalid(err) {
		logf.Log.Error(err, "failed to create object, got an invalid object error")
		return
	}
	Expect(err).NotTo(HaveOccurred())

	err = k8sClient.Create(context.Background(), targetPodD)
	if apierrors.IsInvalid(err) {
		logf.Log.Error(err, "failed to create object, got an invalid object error")
		return
	}
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterEach(func() {
	_ = k8sClient.Delete(context.Background(), targetPodA)
	_ = k8sClient.Delete(context.Background(), targetPodB)
	_ = k8sClient.Delete(context.Background(), targetPodC)
	_ = k8sClient.Delete(context.Background(), targetPodD)
})

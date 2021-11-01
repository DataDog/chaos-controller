// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	timeout = time.Second * 45
)

var (
	cfg         *rest.Config
	k8sClient   client.Client
	k8sManager  ctrl.Manager
	testEnv     *envtest.Environment
	instanceKey types.NamespacedName
	targetPodA  *corev1.Pod
	targetPodB  *corev1.Pod
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(done Done) {
	var err error

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "chart", "templates", "crds")},
	}

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

	// prepare and start manager
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())
	k8sClient = k8sManager.GetClient()
	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()

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
				{
					Image: "k8s.gcr.io/pause:3.4.1",
					Name:  "ctn1",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "foo",
							MountPath: "/mnt/foo",
						},
					},
				},
				{
					Image: "k8s.gcr.io/pause:3.4.1",
					Name:  "ctn2",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "bar",
							MountPath: "/mnt/bar",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "bar",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
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
				{
					Image: "k8s.gcr.io/pause:3.4.1",
					Name:  "ctn1",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "foo",
							MountPath: "/mnt/foo",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	By("Creating target pods")
	Expect(k8sClient.Create(context.Background(), targetPodA)).To(BeNil())
	Expect(k8sClient.Create(context.Background(), targetPodB)).To(BeNil())

	By("Waiting for target pods to be ready")
	Eventually(func() error {
		running, err := podsAreRunning(targetPodA, targetPodB)
		if err != nil {
			return err
		}

		if !running {
			return fmt.Errorf("target pods are not running")
		}

		return nil
	}, timeout).Should(Succeed())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")

	_ = k8sClient.Delete(context.Background(), targetPodA)
	_ = k8sClient.Delete(context.Background(), targetPodB)

	Expect(testEnv.Stop()).To(BeNil())
})

// podsAreRunning returns true when all the given pods have all their containers running
func podsAreRunning(pods ...*corev1.Pod) (bool, error) {
	for _, pod := range pods {
		var p corev1.Pod

		// retrieve pod
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &p); err != nil {
			return false, fmt.Errorf("error getting pod: %w", err)
		}

		// check the pod phase
		if p.Status.Phase != corev1.PodRunning {
			return false, nil
		}

		// check the pod containers statuses (pod phase can be running while all containers are not running)
		// we return false if at least one container in the pod is not running
		running := true
		for _, status := range p.Status.ContainerStatuses {
			if status.State.Running == nil {
				running = false

				break
			}
		}

		if !running {
			return false, nil
		}
	}

	return true, nil
}

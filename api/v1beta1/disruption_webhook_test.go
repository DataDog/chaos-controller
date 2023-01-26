// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	chaostypes "github.com/DataDog/chaos-controller/types"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testDisruptionName = "test-disruption"
)

var oldDisruption, newDisruption *Disruption

var _ = Describe("Disruption", func() {
	Context("ValidateUpdate", func() {
		BeforeEach(func() {
			oldDisruption = makeValidDisruption()
			newDisruption = oldDisruption.DeepCopy()
		})

		Describe("finalizer removal expectations", func() {
			BeforeEach(func() {
				k8sClient = makek8sClientWithDisruptionPod()
				controllerutil.AddFinalizer(oldDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				k8sClient = nil
			})

			When("disruption is deleting without associated disruption pods", func() {
				It("should succeed to remove finalizer (controller needs to be able to remove finalizer in such case)", func() {
					// override client to return NO pod
					k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

					oldDisruption.DeletionTimestamp = &metav1.Time{
						Time: time.Now(),
					}

					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
				})
			})

			When("disruption is running and has pods", func() {
				It("should fail to remove finalizer", func() {
					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
				})
			})

			When("disruption is deleting WITH associated disruption pods", func() {
				It("should fail to remove finalizer", func() {
					oldDisruption.DeletionTimestamp = &metav1.Time{
						Time: time.Now(),
					}

					err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("unable to remove disruption finalizer"))
				})
			})

			When("disruption did not had finalizer", func() {
				It("should be OK to stays without finalizer", func() {
					oldDisruption.Finalizers = nil

					err := newDisruption.ValidateUpdate(oldDisruption)
					Expect(err).Should(Succeed())
				})
			})
		})

		Describe("hash changes expectations", func() {
			When("nothing is updated", func() {
				It("should succeed", func() {
					Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
				})
			})

			When("count is updated", func() {
				BeforeEach(func() {
					newDisruption.Spec.Count = &intstr.IntOrString{IntVal: 2}
				})

				Context("DynamicTargeting (StaticTargeting=false)", func() {
					It("should succeed", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(Succeed())
					})
				})

				Context("StaticTargeting", func() {
					It("should fail", func() {
						oldDisruption.Spec.StaticTargeting = true
						newDisruption.Spec.StaticTargeting = true

						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})
			})

			When("StaticTargeting is updated", func() {
				When("static to dynamic", func() {
					BeforeEach(func() {
						oldDisruption.Spec.StaticTargeting = true
						newDisruption.Spec.StaticTargeting = false
					})

					It("should fail", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})

				When("dynamic to static", func() {
					BeforeEach(func() {
						oldDisruption.Spec.StaticTargeting = false
						newDisruption.Spec.StaticTargeting = true
					})

					It("should fail", func() {
						Expect(newDisruption.ValidateUpdate(oldDisruption)).Should(HaveOccurred())
					})
				})
			})
		})
	})
})

// makeValidDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidDisruption() *Disruption {
	return &Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDisruptionName,
			Namespace: chaosNamespace,
		},
		Spec: DisruptionSpec{
			Count: &intstr.IntOrString{
				IntVal: 1,
			},
			Network: &NetworkDisruptionSpec{
				Drop: 100,
			},
			Selector: labels.Set{
				"name":      "random",
				"namespace": "random",
			},
		},
	}
}

// makek8sClientWithDisruptionPod is a help that creates a k8sClient returning at least one valid pod associated with the Disruption created with makeValidDisruption
func makek8sClientWithDisruptionPod() client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "like-disruption-pod",
				Namespace: chaosNamespace,
				Labels: map[string]string{
					chaostypes.DisruptionNameLabel:      testDisruptionName,
					chaostypes.DisruptionNamespaceLabel: chaosNamespace,
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: "k8s.gcr.io/pause:3.4.1",
						Name:  "ctn1",
					},
				},
			},
		}).
		Build()
}

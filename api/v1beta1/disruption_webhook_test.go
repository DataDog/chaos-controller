// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1

import (
	"github.com/DataDog/chaos-controller/ddmark"
	"github.com/DataDog/chaos-controller/metrics/noop"
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/client-go/tools/record"
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
			oldDisruption = makeValidNetworkDisruption()
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

	Context("ValidateCreate", func() {
		Describe("general errors expectations", func() {
			BeforeEach(func() {
				err := ddmark.InitLibrary(EmbeddedChaosAPI, types.DDMarkChaoslibPrefix)
				Expect(err).ShouldNot(HaveOccurred())
				k8sClient = makek8sClientWithDisruptionPod()
				deleteOnly = false
			})

			JustBeforeEach(func() {
				newDisruption = makeValidNetworkDisruption()
				controllerutil.AddFinalizer(newDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				err := ddmark.CleanupLibraries(types.DDMarkChaoslibPrefix)
				Expect(err).ShouldNot(HaveOccurred())
				k8sClient = nil
				newDisruption = nil
			})

			When("disruption has delete-only mode enable", func() {
				It("should return an error which deny the creation of a new disruption", func() {
					// Arrange
					deleteOnly = true

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("the controller is currently in delete-only mode, you can't create new disruptions for now"))
				})
			})

			When("disruption has invalid name", func() {
				It("should return an error for an invalid d", func() {
					// Arrange
					newDisruption.Name = "invalid-name!"

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("invalid disruption name: found '!', expected: ',' or 'end of string'"))
				})
			})

			When("disruption using the onInit feature without the handler being enabled", func() {
				It("should return an error", func() {
					// Arrange
					newDisruption.Spec.OnInit = true

					// Action
					err := newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix("the chaos handler is disabled but the disruption onInit field is set to true, please enable the handler by specifying the --handler-enabled flag to the controller if you want to use the onInit feature"))
				})
			})

			When("disruption spec is invalid", func() {
				It("should return an error", func() {
					// Arrange
					invalidDisruption := newDisruption.DeepCopy()
					invalidDisruption.Spec.Selector = nil

					// Action
					err := invalidDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(Equal("1 error occurred:\n\t* Spec: either selector or advancedSelector field must be set\n\n"))
				})
			})

			When("ddmark return an error", func() {
				It("should catch this error and propagated it", func() {
					// Arrange
					err := ddmark.CleanupLibraries(types.DDMarkChaoslibPrefix)
					Expect(err).ShouldNot(HaveOccurred())

					// Action
					err = newDisruption.ValidateCreate()

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(Equal("1 error occurred:\n\t* ddmark:  validation_webhook: loaded classes are empty or not found\n\n"))
				})
			})
		})

		Describe("expectations with a disk failure disruption", func() {
			BeforeEach(func() {
				err := ddmark.InitLibrary(EmbeddedChaosAPI, types.DDMarkChaoslibPrefix)
				Expect(err).ShouldNot(HaveOccurred())

				k8sClient = makek8sClientWithDisruptionPod()
				recorder = record.NewFakeRecorder(1)
				metricsSink = noop.New()
				deleteOnly = false
				enableSafemode = true
			})

			JustBeforeEach(func() {
				newDisruption = makeValidDiskFailureDisruption()
				controllerutil.AddFinalizer(newDisruption, chaostypes.DisruptionFinalizer)
			})

			AfterEach(func() {
				err := ddmark.CleanupLibraries(types.DDMarkChaoslibPrefix)
				Expect(err).ShouldNot(HaveOccurred())
				k8sClient = nil
				newDisruption = nil
			})

			When("the disruption target a 'node'", func() {
				JustBeforeEach(func() {
					newDisruption.Spec.Level = chaostypes.DisruptionLevelNode
				})
				Context("with the '/' path", func() {
					It("should deny the usage of '/' path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = "/"

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\"."))
					})
				})
				Context("with the '  /  ' path", func() {
					It("should deny the usage of '   /   ' path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = "   /   "

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption targeting a node must not be \"/\"."))
					})
				})
				Context("with an empty path", func() {
					It("should deny the usage of an empty path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = ""

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption must not be empty."))
					})
				})
				Context("with a blank path", func() {
					It("should deny the usage of a blank path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = " "

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption must not be blank."))
					})
				})
				Context("safe-mode disabled", func() {
					It("should allow the '/' path", func() {
						// Arrange
						newDisruption.Spec.Unsafemode = &UnsafemodeSpec{
							DisableDiskFailurePath: true,
						}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})

			When("the disruption target a 'pod'", func() {
				JustBeforeEach(func() {
					newDisruption.Spec.Level = chaostypes.DisruptionLevelPod
				})
				Context("with the '/' path", func() {
					It("should allow the usage of this path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = "/"

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
				Context("with an empty path", func() {
					It("should deny the usage of an empty path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = ""

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption must not be empty."))
					})
				})
				Context("with a blank path", func() {
					It("should deny the usage of a blank path", func() {
						// Arrange
						newDisruption.Spec.DiskFailure.Path = " "

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(ContainSubstring("at least one of the initial safety nets caught an issue"))
						Expect(err.Error()).Should(ContainSubstring("the specified path for the disk failure disruption must not be blank."))
					})
				})
				Context("with the safe-mode disabled", func() {
					It("should allow the '/' path", func() {
						// Arrange
						newDisruption.Spec.Unsafemode = &UnsafemodeSpec{
							DisableDiskFailurePath: true,
						}

						// Action
						err := newDisruption.ValidateCreate()

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})
	})
})

// makeValidNetworkDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidNetworkDisruption() *Disruption {
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

// makeValidDiskFailureDisruption is a helper that constructs a valid Disruption suited for basic webhook validation testing
func makeValidDiskFailureDisruption() *Disruption {
	return &Disruption{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDisruptionName,
			Namespace: chaosNamespace,
		},
		Spec: DisruptionSpec{
			Count: &intstr.IntOrString{
				IntVal: 1,
			},
			Selector: labels.Set{
				"name":      "random",
				"namespace": "random",
			},
			DiskFailure: &DiskFailureSpec{
				Path: "/",
			},
		},
	}
}

// makek8sClientWithDisruptionPod is a help that creates a k8sClient returning at least one valid pod associated with the Disruption created with makeValidNetworkDisruption
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

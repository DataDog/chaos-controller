// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers_test

import (
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/types"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
)

var _ = Describe("Watcher factory", func() {
	var (
		disruption        chaosv1beta1.Disruption
		err               error
		eventRecorderMock record.EventRecorder
		noopSink          noop.Sink
		readerMock        mocks.ReaderMock
		watcherFactory    watchers.Factory
		watcher           watchers.Watcher
		cacheMock         *CacheMock
	)

	const watcherName = "name"

	BeforeEach(func() {
		cacheMock = &CacheMock{}
	})

	JustBeforeEach(func() {
		watcherFactory = watchers.NewWatcherFactory(logger, &noopSink, &readerMock, eventRecorderMock)
	})

	It("should not be nil", func() {
		Expect(watcherFactory).ToNot(BeNil())
	})

	When("NewChaosPodWatcher is called", func() {
		JustBeforeEach(func() {
			// Action
			watcher, err = watcherFactory.NewChaosPodWatcher(watcherName, &disruption, cacheMock)
		})

		Context("with a valid disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "disruption-name",
						Namespace: "namespace",
					},
				}
			})

			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should create the watcher with the expected name", func() {
				Expect(watcher.GetName()).Should(Equal(watcherName))
			})

			It("should have a valid k8s cache options", func() {
				expectedObjectSelector := k8scache.ObjectSelector{Label: labels.SelectorFromValidatedSet(map[string]string{
					chaostypes.DisruptionNameLabel:      disruption.Name,
					chaostypes.DisruptionNamespaceLabel: disruption.Namespace,
				})}

				By("having the same object selector")
				for _, selectorByObject := range watcher.GetConfig().CacheOptions.SelectorsByObject {
					Expect(selectorByObject).Should(Equal(expectedObjectSelector))
				}

				By("having the same namespace")
				Expect(watcher.GetConfig().CacheOptions.Namespace).Should(Equal(disruption.Namespace))
			})
		})

		Context("with an empty disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = chaosv1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "",
						Namespace: "",
					},
				}
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the disruption fields name and namespace of the ObjectMeta field are required"))
			})
		})
	})

	Describe("NewDisruptionTargetWatcher is called", func() {
		JustBeforeEach(func() {
			// Action
			watcher, err = watcherFactory.NewDisruptionTargetWatcher(watcherName, true, &disruption, cacheMock)
		})

		Context("with a valid disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = chaosv1beta1.Disruption{
					Spec: chaosv1beta1.DisruptionSpec{
						Selector: labels.Set{
							"Lorem": "ipsum",
						},
					},
				}
			})

			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			Context("with a node disruption level", func() {
				BeforeEach(func() {
					disruption.Spec.Level = types.DisruptionLevelNode
				})

				It("should not return an error", func() {
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("should create the watcher with the expected name", func() {
					Expect(watcher.GetName()).Should(Equal(watcherName))
				})
			})
		})

		Context("with an empty disruption", func() {
			BeforeEach(func() {
				// Arrange
				disruption = chaosv1beta1.Disruption{}
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(HavePrefix("could not create the name disruption target watcher. Error: error getting instance selector"))
			})
		})
	})
})

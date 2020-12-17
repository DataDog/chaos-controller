// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package controllers

import (
	"context"
	"os"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ChaosFailureInjectionImageVariableName is the name of the chaos failure injection image variable
	ChaosFailureInjectionImageVariableName = "CHAOS_INJECTOR_IMAGE"
)

type fakeClient struct {
	ListOptions []*client.ListOptions
}

func (f fakeClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return nil
}
func (f *fakeClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	for _, opt := range opts {
		if o, ok := opt.(*client.ListOptions); ok {
			f.ListOptions = append(f.ListOptions, o)
		}
	}
	if l, ok := list.(*corev1.PodList); ok {
		l.Items = mixedStatusPods
	} else if l, ok := list.(*corev1.NodeList); ok {
		l.Items = nodes
	}

	return nil
}
func (f fakeClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	return nil
}
func (f fakeClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	return nil
}
func (f fakeClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return nil
}
func (f fakeClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}
func (f fakeClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}
func (f fakeClient) Status() client.StatusWriter {
	return nil
}

var mixedStatusPods []corev1.Pod
var twoPods []corev1.Pod
var nodes []corev1.Node

var _ = Describe("Helpers", func() {
	var c fakeClient
	var image string
	var disruption *chaosv1beta1.Disruption
	var targetSelector RunningTargetSelector

	BeforeEach(func() {
		targetSelector = RunningTargetSelector{}

		c = fakeClient{}

		runningPod1 := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "genericRunningPod1",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}

		runningPod2 := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "genericRunningPod2",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}

		failedPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "genericFailedPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}

		mixedStatusPods = []corev1.Pod{
			*runningPod1,
			*runningPod2,
			*failedPod,
		}

		twoPods = []corev1.Pod{
			*runningPod1,
			*runningPod2,
		}

		// nodes list
		nodes = []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		}

		// misc
		image = "chaos-injector:latest"
		os.Setenv(ChaosFailureInjectionImageVariableName, image)

		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Selector: map[string]string{"foo": "bar"},
				NodeFailure: &chaosv1beta1.NodeFailureSpec{
					Shutdown: false,
				},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts:          []string{"127.0.0.1"},
					Port:           80,
					Protocol:       "tcp",
					Drop:           0,
					Corrupt:        0,
					Delay:          1000,
					BandwidthLimit: 10000,
				},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{},
				DiskPressure: &chaosv1beta1.DiskPressureSpec{
					Path:       "/mnt/foo",
					Throttling: chaosv1beta1.DiskPressureThrottlingSpec{},
				},
			},
		}
	})

	AfterEach(func() {
		os.Unsetenv(ChaosFailureInjectionImageVariableName)
	})

	Describe("GetMatchingPods", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				disruption.Namespace = ""
				disruption.Spec.Selector = nil

				_, err := targetSelector.GetMatchingPods(nil, disruption)
				Expect(err).NotTo(BeNil())
			})
		})
		Context("with non-empty label selector", func() {
			It("should pass given selector for the given namespace to the client", func() {
				ls := map[string]string{
					"app": "bar",
				}
				disruption.Namespace = "foo"
				disruption.Spec.Selector = ls

				_, err := targetSelector.GetMatchingPods(&c, disruption)
				Expect(err).To(BeNil())
				// Note: Namespace filter is not applied for results of the fakeClient.
				//       We instead test this functionality in the controller tests.
				Expect(c.ListOptions[0].Namespace).To(Equal("foo"))
				Expect(c.ListOptions[0].LabelSelector.Matches(labels.Set(ls))).To(BeTrue())
			})
			It("should return the pods list except for failed pod", func() {
				disruption.Namespace = ""
				disruption.Spec.Selector = map[string]string{
					"app": "bar",
				}

				r, err := targetSelector.GetMatchingPods(&c, disruption)
				numFailedPods := 1
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(mixedStatusPods) - numFailedPods))
			})
		})
	})

	Describe("GetMatchingNodes", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				disruption.Spec.Selector = nil
				_, err := targetSelector.GetMatchingNodes(&c, disruption)
				Expect(err).NotTo(BeNil())
			})
		})
		Context("with non-empty label selector", func() {
			It("should pass given selector to the client", func() {
				ls := map[string]string{"app": "bar"}
				disruption.Spec.Selector = ls
				_, err := targetSelector.GetMatchingNodes(&c, disruption)
				Expect(err).To(BeNil())
				Expect(c.ListOptions[0].LabelSelector.Matches(labels.Set(ls))).To(BeTrue())
			})
			It("should return the nodes list with no error", func() {
				disruption.Spec.Selector = map[string]string{"foo": "bar"}
				r, err := targetSelector.GetMatchingNodes(&c, disruption)
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(nodes)))
				Expect(r.Items[0].Name).To(Equal("foo"))
			})
		})
	})

	// "TargetIsHealthy"
})

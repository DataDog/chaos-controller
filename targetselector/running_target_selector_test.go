// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package targetselector

import (
	"context"
	"os"
	"reflect"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (f *fakeClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if key.Name == "runningPod" {
		objVal := reflect.ValueOf(obj)
		nodeVal := reflect.ValueOf(runningPod1)
		reflect.Indirect(objVal).Set(reflect.Indirect(nodeVal))
	} else if key.Name == "failedPod" {
		objVal := reflect.ValueOf(obj)
		nodeVal := reflect.ValueOf(failedPod)
		reflect.Indirect(objVal).Set(reflect.Indirect(nodeVal))
	} else if key.Name == "pendingPod" {
		objVal := reflect.ValueOf(obj)
		nodeVal := reflect.ValueOf(pendingPod)
		reflect.Indirect(objVal).Set(reflect.Indirect(nodeVal))
	} else if key.Name == "runningNode" {
		objVal := reflect.ValueOf(obj)
		nodeVal := reflect.ValueOf(runningNode)
		reflect.Indirect(objVal).Set(reflect.Indirect(nodeVal))
	} else if key.Name == "failedNode" {
		objVal := reflect.ValueOf(obj)
		nodeVal := reflect.ValueOf(failedNode)
		reflect.Indirect(objVal).Set(reflect.Indirect(nodeVal))
	}

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
		l.Items = justRunningNodes
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

var runningPod1 *corev1.Pod
var runningPod2 *corev1.Pod
var failedPod *corev1.Pod
var pendingPod *corev1.Pod

var mixedStatusPods []corev1.Pod
var twoPods []corev1.Pod

var runningNode *corev1.Node
var failedNode *corev1.Node

var justRunningNodes []corev1.Node
var mixedNodes []corev1.Node

var _ = Describe("Helpers", func() {
	var c fakeClient
	var image string
	var disruption *chaosv1beta1.Disruption
	var targetSelector RunningTargetSelector

	BeforeEach(func() {
		targetSelector = RunningTargetSelector{}

		c = fakeClient{}

		runningPod1 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "runningPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "foo",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		runningPod2 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anotherRunningPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "foo",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		failedPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failedPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}

		pendingPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pendingPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "chaos-handler",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		mixedStatusPods = []corev1.Pod{
			*runningPod1,
			*runningPod2,
			*failedPod,
			*pendingPod,
		}

		twoPods = []corev1.Pod{
			*runningPod1,
			*runningPod2,
		}

		runningNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "runningNode",
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
		}

		failedNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "failedNode",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		}

		justRunningNodes = []corev1.Node{
			*runningNode,
		}

		mixedNodes = []corev1.Node{
			*runningNode,
			*failedNode,
		}

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
				ContainerFailure: &chaosv1beta1.ContainerFailureSpec{
					Forced: false,
				},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
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
			BeforeEach(func() {
				disruption.Namespace = "foo"
				disruption.Spec.Selector = map[string]string{
					"app": "bar",
				}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"bar"},
					},
				}
			})

			It("should pass given selector for the given namespace to the client", func() {
				_, err := targetSelector.GetMatchingPods(&c, disruption)
				Expect(err).To(BeNil())
				// Note: Namespace filter is not applied for results of the fakeClient.
				//       We instead test this functionality in the controller tests.
				Expect(c.ListOptions[0].Namespace).To(Equal("foo"))
				Expect(c.ListOptions[0].LabelSelector.String()).To(Equal("app=bar,app,!app,app in (bar),app notin (bar)"))
			})

			It("should return the pods list except for failed pod", func() {
				disruption.Namespace = ""

				r, err := targetSelector.GetMatchingPods(&c, disruption)
				numExcludedPods := 2 // pending + failed pods
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(mixedStatusPods) - numExcludedPods))
			})
		})

		Context("with on init mode enabled", func() {
			BeforeEach(func() {
				disruption.Spec.OnInit = true
			})

			It("should match pending pods with init containers only", func() {
				r, err := targetSelector.GetMatchingPods(&c, disruption)
				Expect(err).To(BeNil())
				Expect(r.Items[0]).To(Equal(*pendingPod))
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
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{
					"app": "bar",
				}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"bar"},
					},
				}
			})

			It("should pass given selector to the client", func() {
				_, err := targetSelector.GetMatchingNodes(&c, disruption)
				Expect(err).To(BeNil())
				Expect(c.ListOptions[0].LabelSelector.String()).To(Equal("app=bar,app,!app,app in (bar),app notin (bar)"))
			})

			It("should return the nodes list with no error", func() {
				r, err := targetSelector.GetMatchingNodes(&c, disruption)

				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(justRunningNodes)))
				Expect(r.Items[0].Name).To(Equal("runningNode"))
			})
		})
	})

	Describe("TargetIsHealthy", func() {
		Context("with pod-level disruption spec", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{"foo": "bar"}
				disruption.Spec.Level = types.DisruptionLevelPod
			})

			It("should return no error for running pod", func() {
				err := targetSelector.TargetIsHealthy("runningPod", &c, disruption)
				Expect(err).To(BeNil())
			})
			It("should return error for failed pod", func() {
				err := targetSelector.TargetIsHealthy("failedPod", &c, disruption)
				Expect(err).ToNot(BeNil())
			})
		})

		Context("with node-level disruption spec", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{"foo": "bar"}
				disruption.Spec.Level = types.DisruptionLevelNode
			})

			It("should return an error for running node", func() {
				err := targetSelector.TargetIsHealthy("runnningNode", &c, disruption)
				Expect(err).ToNot(BeNil())
			})
			It("should return an error for failed node", func() {
				err := targetSelector.TargetIsHealthy("failedNode", &c, disruption)
				Expect(err).ToNot(BeNil())
			})
		})
	})
})

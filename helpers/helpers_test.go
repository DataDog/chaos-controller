// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package helpers_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/helpers"
	. "github.com/DataDog/chaos-controller/helpers"
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
	var owner corev1.Pod
	var ownedPod *corev1.Pod
	var image string

	BeforeEach(func() {
		c = fakeClient{}

		// owner pod
		owner = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "owner",
				UID:  "fakeUID",
			},
		}
		ownerRef := metav1.NewControllerRef(&owner, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})

		ownedPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "foo",
				Namespace:       "foo",
				OwnerReferences: []metav1.OwnerReference{*ownerRef},
			},
			Spec: corev1.PodSpec{
				NodeName: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		}
		runningPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "genericRunningPod",
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
			*runningPod,
			*failedPod,
			*ownedPod,
		}

		twoPods = []corev1.Pod{
			*runningPod,
			*ownedPod,
		}

		// nodes list
		nodes = []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Status: corev1.NodeStatus{
					Phase: corev1.NodeRunning,
				},
			},
		}

		// misc
		image = "chaos-injector:latest"
		os.Setenv(helpers.ChaosFailureInjectionImageVariableName, image)
	})

	AfterEach(func() {
		os.Unsetenv(helpers.ChaosFailureInjectionImageVariableName)
	})

	Describe("GetMatchingPods", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				_, err := GetMatchingPods(nil, "", nil)
				Expect(err).NotTo(BeNil())
			})
		})
		Context("with non-empty label selector", func() {
			It("should pass given selector for the given namespace to the client", func() {
				ns := "foo"
				ls := map[string]string{
					"app": "bar",
				}
				_, err := GetMatchingPods(&c, ns, ls)
				Expect(err).To(BeNil())
				// Note: Namespace filter is not applied for results of the fakeClient.
				//       We instead test this functionality in the controller tests.
				Expect(c.ListOptions[0].Namespace).To(Equal(ns))
				Expect(c.ListOptions[0].LabelSelector.Matches(labels.Set(ls))).To(BeTrue())
			})
			It("should return the pods list except for failed pod", func() {
				ls := map[string]string{
					"app": "bar",
				}
				r, err := GetMatchingPods(&c, "", ls)
				numFailedPods := 1
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(mixedStatusPods) - numFailedPods))
			})
		})
	})

	Describe("GetMatchingNodes", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				_, err := GetMatchingNodes(nil, nil)
				Expect(err).NotTo(BeNil())
			})
		})
		Context("with non-empty label selector", func() {
			It("should pass given selector to the client", func() {
				ls := map[string]string{
					"app": "bar",
				}
				_, err := GetMatchingNodes(&c, ls)
				Expect(err).To(BeNil())
				Expect(c.ListOptions[0].LabelSelector.Matches(labels.Set(ls))).To(BeTrue())
			})
			It("should return the nodes list with no error", func() {
				r, err := GetMatchingNodes(&c, map[string]string{"foo": "bar"})
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(nodes)))
				Expect(r.Items[0].Name).To(Equal("foo"))
			})
		})
	})

	Describe("GetOwnedPods", func() {
		It("should return the pod owned by owner", func() {
			r, err := GetOwnedPods(&c, &owner, nil)
			Expect(err).To(BeNil())
			Expect(len(r.Items)).To(Equal(1))
			Expect(r.Items[0]).To(Equal(*ownedPod))
		})
	})
})

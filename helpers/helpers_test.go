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
	l, _ := list.(*corev1.PodList)
	l.Items = pods
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

var pods []corev1.Pod

var _ = Describe("Helpers", func() {
	var c fakeClient
	var owner corev1.Pod
	var pod *corev1.Pod
	var image string

	BeforeEach(func() {
		c = fakeClient{}
		owner = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "owner",
				UID:  "fakeUID",
			},
		}
		ownerRef := metav1.NewControllerRef(&owner, schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "foo",
				Namespace:       "foo",
				OwnerReferences: []metav1.OwnerReference{*ownerRef},
			},
			Spec: corev1.PodSpec{
				NodeName: "bar",
			},
		}
		pods = []corev1.Pod{
			*pod,
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
			},
		}

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
				Expect(c.ListOptions[0].Namespace).To(Equal(ns))
				Expect(c.ListOptions[0].LabelSelector.Matches(labels.Set(ls))).To(BeTrue())
			})
			It("should return the pods list with no error", func() {
				r, err := GetMatchingPods(&c, "", map[string]string{"foo": "bar"})
				Expect(err).To(BeNil())
				Expect(len(r.Items)).To(Equal(len(pods)))
				Expect(r.Items[0].Name).To(Equal("foo"))
			})
		})
	})

	Describe("PickRandomPods", func() {
		Context("with n greater than pods list size", func() {
			It("should return the whole slice shuffled", func() {
				r := PickRandomPods(uint(len(pods)+1), pods)
				Expect(len(r)).To(Equal(len(pods)))
				Expect(r[0]).To(Equal(pods[1]))
				Expect(r[1]).To(Equal(pods[0]))
			})
		})
		Context("with n lower than pods list size", func() {
			It("should return a shuffled subslice", func() {
				r := PickRandomPods(1, pods)
				Expect(len(r)).To(Equal(1))
			})
		})
	})

	Describe("GetOwnedPods", func() {
		It("should return the pod owned by owner", func() {
			r, err := GetOwnedPods(&c, &owner, nil)
			Expect(err).To(BeNil())
			Expect(len(r.Items)).To(Equal(1))
			Expect(r.Items[0]).To(Equal(*pod))
		})
	})
})

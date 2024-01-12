// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

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

package v1beta1

import (
	"context"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// DisruptionInterface is the interface for the end client the user will use
type DisruptionInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.DisruptionList, error)
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.Disruption, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Create(ctx context.Context, disruption *v1beta1.Disruption) (*v1beta1.Disruption, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type disruptionClient struct {
	restClient rest.Interface
	ns         string
}

func (c *disruptionClient) List(ctx context.Context, opts metav1.ListOptions) (*v1beta1.DisruptionList, error) {
	result := v1beta1.DisruptionList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("disruptions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *disruptionClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.Disruption, error) {
	result := v1beta1.Disruption{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("disruptions").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *disruptionClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.
		Delete().
		Namespace(c.ns).
		Resource("disruptions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

func (c *disruptionClient) Create(ctx context.Context, disruption *v1beta1.Disruption) (*v1beta1.Disruption, error) {
	result := v1beta1.Disruption{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource("disruptions").
		Body(disruption).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *disruptionClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true

	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource("disruptions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

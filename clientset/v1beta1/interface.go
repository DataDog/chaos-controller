// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

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
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// DisruptionV1Beta1Interface is the interface that the client below matches
type DisruptionV1Beta1Interface interface {
	Disruptions(namespace string) DisruptionInterface
}

type DisruptionV1Beta1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*DisruptionV1Beta1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1beta1.GroupName, Version: v1beta1.APIVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &DisruptionV1Beta1Client{restClient: client}, nil
}

// Disruptions returns a client for interacting with disruptions
func (c *DisruptionV1Beta1Client) Disruptions(namespace string) DisruptionInterface {
	return &disruptionClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

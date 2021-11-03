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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = Describe("Label Selector Validation", func() {

	Context("validating an empty label selector", func() {
		It("", func() {
			selector := labels.Set{}
			Expect(validateLabelSelector(selector.AsSelector())).ToNot(BeNil())
		})
	})
	Context("validating a good label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "bar"}
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating special characters in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "”bar”"}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
	Context("validating too many quotes in label selector", func() {
		It("", func() {
			selector := labels.Set{"foo": "\"bar\""}
			//.AsSelector() should strip invalid characters
			Expect(validateLabelSelector(selector.AsSelector())).To(BeNil())
		})
	})
})

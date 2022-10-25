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

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var disruptionStatus *v1beta1.DisruptionStatus

var _ = Describe("Check if a target exist into DisruptionStatus targets list", func() {

	BeforeEach(func() {
		disruptionStatus = &v1beta1.DisruptionStatus{}
	})

	AfterEach(func() {
		disruptionStatus = nil
	})

	Context("with an empty target", func() {
		It("should return false", func() {
			target := ""
			Expect(disruptionStatus.HasTarget(target)).Should(BeFalse())
		})
	})

	Context("with an existing target", func() {
		It("should return true", func() {
			disruptionStatus.Targets = append(disruptionStatus.Targets, "test-1")
			Expect(disruptionStatus.HasTarget("test-1")).Should(BeTrue())
		})
	})

	Context("with an non existing target", func() {
		It("should return false", func() {
			disruptionStatus.Targets = append(disruptionStatus.Targets, "test-1")
			Expect(disruptionStatus.HasTarget("test-2")).Should(BeFalse())
		})
	})
})

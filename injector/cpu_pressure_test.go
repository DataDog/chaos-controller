// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Failure", func() {
	var (
		config     Config
		inj        Injector
		spec       v1beta1.CPUPressureSpec
		background *mocks.BackgroundProcessManagerMock
	)

	BeforeEach(func() {
		background = mocks.NewBackgroundProcessManagerMock(GinkgoT())

		// config
		config = Config{
			Log:         log,
			MetricsSink: ms,
		}

		// spec
		spec = v1beta1.CPUPressureSpec{}

		inj = NewCPUPressureInjector(config, spec, background, func(i int) []string { return []string{"do-not-care"} })
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
		})

		Context("user request to stress all the cores", func() {
			JustBeforeEach(func() {
				userRequestAll := intstr.FromString("100%")
				spec.Count = &userRequestAll
			})
		})

		Context("user request to stress half the cores", func() {
			JustBeforeEach(func() {
				userRequestHalf := intstr.FromString("50%")
				spec.Count = &userRequestHalf
			})

			// It("should call stresserManager track cores and get new core to apply pressure", func() {
			// 	// left empty as AfterEach 'AssertExpectations' check all this tests expectations
			// })
		})
	})
})

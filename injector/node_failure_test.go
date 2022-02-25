// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
)

var _ = Describe("Failure", func() {
	var (
		config NodeFailureInjectorConfig
		fw     FileWriterMock
		inj    Injector
		spec   v1beta1.NodeFailureSpec
	)

	BeforeEach(func() {
		fw = FileWriterMock{}
		fw.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config = NodeFailureInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
			},
			FileWriter: &fw,
		}

		spec = v1beta1.NodeFailureSpec{}

		// set mandatory environment variables
		os.Setenv(env.InjectorMountSysrq, "/mnt/sysrq")
		os.Setenv(env.InjectorMountSysrqTrigger, "/mnt/sysrq-trigger")
	})

	JustBeforeEach(func() {
		var err error
		inj, err = NewNodeFailureInjector(spec, config)

		Expect(err).To(BeNil())
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(BeNil())
			time.Sleep(time.Second * 11)
		})

		It("should enable the sysrq handler", func() {
			fw.AssertCalled(GinkgoT(), "Write", "/mnt/sysrq", mock.Anything, "1")
		})

		Context("with shutdown enabled", func() {
			BeforeEach(func() {
				spec.Shutdown = true
			})

			It("should write into the sysrq trigger file", func() {
				fw.AssertCalled(GinkgoT(), "Write", "/mnt/sysrq-trigger", mock.Anything, "o")
			})
		})

		Context("with shutdown disabled", func() {
			It("should write into the sysrq trigger file", func() {
				fw.AssertCalled(GinkgoT(), "Write", "/mnt/sysrq-trigger", mock.Anything, "c")
			})
		})

	})
})

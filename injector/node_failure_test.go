// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	. "github.com/DataDog/chaos-fi-controller/injector"
)

type fakeFileWriter struct {
	mock.Mock
}

func (fw *fakeFileWriter) Write(path string, mode os.FileMode, data string) error {
	args := fw.Called(path, mode, data)
	return args.Error(0)
}

var _ = Describe("Failure", func() {
	var (
		config NodeFailureInjectorConfig
		fw     fakeFileWriter
		inj    Injector
		spec   v1beta1.NodeFailureSpec
	)

	BeforeEach(func() {
		fw = fakeFileWriter{}
		fw.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config = NodeFailureInjectorConfig{
			FileWriter: &fw,
		}

		spec = v1beta1.NodeFailureSpec{}
	})

	JustBeforeEach(func() {
		inj = NewNodeFailureInjectorWithConfig("fake", spec, log, config)
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			inj.Inject()
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

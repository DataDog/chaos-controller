// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	. "github.com/DataDog/chaos-fi-controller/injector"
)

type fakeFileWriter struct {
	calls []fakeFileWriterCall
}

type fakeFileWriterCall struct {
	path string
	data string
}

func (fw *fakeFileWriter) Write(path string, _ os.FileMode, data string) error {
	fw.calls = append(fw.calls, fakeFileWriterCall{path: path, data: data})
	return nil
}

var _ = Describe("Failure", func() {
	var f NodeFailureInjector
	var fw fakeFileWriter
	BeforeEach(func() {
		fw = fakeFileWriter{
			calls: []fakeFileWriterCall{},
		}
		f = NodeFailureInjector{
			Injector: Injector{
				UID: "fake",
				Log: log,
			},
			Spec:       &v1beta1.NodeFailureSpec{},
			FileWriter: &fw,
		}
	})

	Describe("injection", func() {
		It("should write to the sysrq file", func() {
			f.Inject()
			Expect(len(fw.calls)).To(Equal(2))
			Expect(fw.calls[0].path).To(Equal("/mnt/sysrq"))
			Expect(fw.calls[0].data).To(Equal("1"))
			Expect(fw.calls[1].path).To(Equal("/mnt/sysrq-trigger"))
			Expect(fw.calls[1].data).To(Equal("c"))
		})
	})
})

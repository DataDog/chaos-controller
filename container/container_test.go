// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package container_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/container"
)

var _ = Describe("Container", func() {
	var (
		config  Config
		runtime *container.RuntimeMock
		ctn     Container
	)

	BeforeEach(func() {
		// runtime
		runtime = &container.RuntimeMock{}
		runtime.On("PID", mock.Anything).Return(uint32(666), nil)
		runtime.On("CgroupPath", mock.Anything).Return("/fake/cgroup/path", nil)
		runtime.On("Name", mock.Anything).Return("", nil)

		// config
		config = Config{
			Runtime: runtime,
		}
	})

	JustBeforeEach(func() {
		var err error
		ctn, err = NewWithConfig("containerd://fake", config)
		Expect(err).To(BeNil())
	})

	Describe("loading a container", func() {
		It("should return a container object with parsed info", func() {
			Expect(ctn.ID()).To(Equal("fake"))
			Expect(ctn.PID()).To(Equal(uint32(666)))
			Expect(ctn.CgroupPath()).To(Equal("/fake/cgroup/path"))
		})
	})
})

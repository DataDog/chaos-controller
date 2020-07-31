// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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
		config              Config
		rootns, containerns int
		netns               *container.NetnsMock
		runtime             *container.RuntimeMock
		cgroup              *container.CgroupMock
		ctn                 Container
		netnsGetFromPIDCall *mock.Call
	)

	BeforeEach(func() {
		// netns
		rootns = 1
		containerns = 2
		netns = &container.NetnsMock{
			Currentns: rootns,
			Fakens:    containerns,
		}
		netns.On("Set", mock.Anything).Return(nil)
		netns.On("GetCurrent").Return(netns.Currentns, nil)
		netnsGetFromPIDCall = netns.On("GetFromPID", mock.Anything).Return(netns.Fakens, nil)

		// runtime
		runtime = &container.RuntimeMock{}
		runtime.On("PID", mock.Anything).Return(uint32(666), nil)
		runtime.On("CgroupPath", mock.Anything).Return("/fake/cgroup/path", nil)

		// cgroup
		cgroup = &container.CgroupMock{}
		cgroup.On("JoinCPU", mock.Anything).Return(nil)

		// config
		config = Config{
			Netns:   netns,
			Runtime: runtime,
			Cgroup:  cgroup,
		}
	})

	JustBeforeEach(func() {
		var err error
		ctn, err = NewWithConfig("containerd://fake", config)
		Expect(err).To(BeNil())
	})

	Describe("loading a container", func() {
		It("should return a container object with a parsed ID", func() {
			Expect(ctn.ID()).To(Equal("fake"))
		})
	})

	Describe("entering and exiting the container network namespace", func() {
		It("should enter the container network namespace and leave it", func() {
			err := ctn.EnterNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(netns.Currentns).To(Equal(containerns))

			err = ctn.ExitNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(netns.Currentns).To(Equal(rootns))
		})
	})

	Describe("entering the container network namespace with a host network container", func() {
		BeforeEach(func() {
			netnsGetFromPIDCall.Return(rootns, nil)
		})

		It("should not enter the container network namespace if it is the same as the root", func() {
			err := ctn.EnterNetworkNamespace()
			Expect(err).To(Not(BeNil()))
		})
	})
})

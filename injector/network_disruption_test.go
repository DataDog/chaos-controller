// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
)

var _ = Describe("Failure", func() {
	var (
		ctn              container.ContainerMock
		inj              Injector
		config           NetworkConfigMock
		spec             v1beta1.NetworkDisruptionSpec
		cgroup           container.CgroupMock
		cgroupExistsCall *mock.Call
	)

	BeforeEach(func() {
		// cgroup
		cgroup = container.CgroupMock{}
		cgroup.On("Write", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		cgroupExistsCall = cgroup.On("Exists", mock.Anything).Return(true, nil)

		// container
		ctn = container.ContainerMock{}
		ctn.On("EnterNetworkNamespace").Return(nil)
		ctn.On("ExitNetworkNamespace").Return(nil)
		ctn.On("Cgroup").Return(&cgroup)

		// network disruption conf
		config = NetworkConfigMock{}
		config.On("AddNetem", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		config.On("AddOutputLimit", mock.Anything).Return()
		config.On("ApplyOperations").Return(nil)
		config.On("ClearOperations").Return(nil)

		spec = v1beta1.NetworkDisruptionSpec{
			Hosts:          []string{"testhost"},
			Port:           22,
			Drop:           5,
			Duplicate:      5,
			Corrupt:        1,
			Delay:          1000,
			Jitter:         100,
			BandwidthLimit: 10000,
		}
	})

	JustBeforeEach(func() {
		inj = NewNetworkDisruptionInjectorWithConfig("fake", spec, &ctn, log, ms, &config)
	})

	Describe("inj.Inject", func() {
		JustBeforeEach(func() {
			inj.Inject()
		})

		It("should enter and exit the container network namespace", func() {
			ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")
			ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")
		})

		It("should call AddNetem on its network disruption config", func() {
			config.AssertCalled(GinkgoT(), "AddNetem", time.Second, time.Second, spec.Drop, spec.Corrupt, spec.Duplicate)
		})

		It("should call AddOutputLimit on its network disruption config", func() {
			config.AssertCalled(GinkgoT(), "AddOutputLimit", uint(spec.BandwidthLimit))
		})

		It("should write 2:2 classid to the container net_cls cgroup", func() {
			cgroup.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0x00020002")
		})
	})

	Describe("inj.Clean", func() {
		JustBeforeEach(func() {
			inj.Clean()
		})

		It("should enter and exit the container network namespace", func() {
			ctn.AssertCalled(GinkgoT(), "EnterNetworkNamespace")
			ctn.AssertCalled(GinkgoT(), "ExitNetworkNamespace")
		})

		It("should call ClearOperations on its network disruption config", func() {
			config.AssertCalled(GinkgoT(), "ClearOperations")
		})

		It("should write the default classid value in the container net_cls cgroup", func() {
			cgroup.AssertCalled(GinkgoT(), "Exists", "net_cls")
			cgroup.AssertCalled(GinkgoT(), "Write", "net_cls", "net_cls.classid", "0x0")
		})

		When("net_cls cgroup does not exist anymore", func() {
			BeforeEach(func() {
				cgroupExistsCall.Return(false, nil)
			})

			It("should skip the cgroup cleanup", func() {
				cgroup.AssertNumberOfCalls(GinkgoT(), "Write", 0)
			})
		})
	})
})

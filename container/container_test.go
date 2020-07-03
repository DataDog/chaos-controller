// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	. "github.com/DataDog/chaos-controller/container"
)

// fake netns driver
type fakeNetns struct {
	mock.Mock

	currentns int
	fakens    int
}

func (f *fakeNetns) Set(ns int) error {
	f.currentns = ns
	args := f.Called(ns)

	return args.Error(0)
}
func (f *fakeNetns) GetCurrent() (int, error) {
	args := f.Called()

	return args.Int(0), args.Error(1)
}
func (f *fakeNetns) GetFromPID(pid uint32) (int, error) {
	args := f.Called(pid)

	return args.Int(0), args.Error(1)
}

// fake cgroup
type fakeCgroup struct {
	mock.Mock
}

func (f *fakeCgroup) JoinCPU() error {
	args := f.Called()

	return args.Error(0)
}

func (f *fakeCgroup) DiskThrottleRead(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

func (f *fakeCgroup) DiskThrottleWrite(identifier, bps int) error {
	args := f.Called(identifier, bps)

	return args.Error(0)
}

// fake runtime
type fakeRuntime struct {
	mock.Mock
}

func (f *fakeRuntime) PID(id string) (uint32, error) {
	args := f.Called(id)

	return args.Get(0).(uint32), args.Error(1)
}

func (f *fakeRuntime) CgroupPath(id string) (string, error) {
	args := f.Called(id)

	return args.String(0), args.Error(1)
}

func (f *fakeRuntime) HostPath(id, path string) (string, error) {
	args := f.Called(id, path)

	return args.String(0), args.Error(1)
}

// tests
var _ = Describe("Container", func() {
	var (
		config              Config
		rootns, containerns int
		netns               *fakeNetns
		runtime             *fakeRuntime
		cgroup              *fakeCgroup
		ctn                 Container
		netnsGetFromPIDCall *mock.Call
	)

	BeforeEach(func() {
		// netns
		rootns = 1
		containerns = 2
		netns = &fakeNetns{
			currentns: rootns,
			fakens:    containerns,
		}
		netns.On("Set", mock.Anything).Return(nil)
		netns.On("GetCurrent").Return(netns.currentns, nil)
		netnsGetFromPIDCall = netns.On("GetFromPID", mock.Anything).Return(netns.fakens, nil)

		// runtime
		runtime = &fakeRuntime{}
		runtime.On("PID", mock.Anything).Return(uint32(666), nil)
		runtime.On("CgroupPath", mock.Anything).Return("/fake/cgroup/path", nil)

		// cgroup
		cgroup = &fakeCgroup{}
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
			Expect(netns.currentns).To(Equal(containerns))

			err = ctn.ExitNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(netns.currentns).To(Equal(rootns))
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

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package container_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/DataDog/chaos-controller/container"
)

// fake netns driver
type fakeNetns struct {
	currentns int
	fakens    int
}

func (f *fakeNetns) Set(ns int) error {
	f.currentns = ns
	return nil
}
func (f fakeNetns) GetCurrent() (int, error) {
	return f.currentns, nil
}
func (f fakeNetns) GetFromPID(uint32) (int, error) {
	return f.fakens, nil
}

// fake runtime
type fakeRuntime struct{}

func (f fakeRuntime) PID(string) (uint32, error) {
	return 666, nil
}

// tests
var _ = Describe("Container", func() {
	var config Config
	var rootns, containerns int
	var netns fakeNetns

	BeforeEach(func() {
		rootns = 1
		containerns = 2
		netns = fakeNetns{
			currentns: rootns,
			fakens:    containerns,
		}
		config = Config{
			Netns:   &netns,
			Runtime: fakeRuntime{},
		}
	})

	Describe("loading a container", func() {
		It("should return a container object with a parsed ID", func() {
			c, err := NewWithConfig("containerd://fake", config)
			Expect(err).To(BeNil())
			Expect(c.ID()).To(Equal("fake"))
		})
	})

	Describe("entering and exiting the container network namespace", func() {
		It("should enter the container network namespace and leave it", func() {
			c, err := NewWithConfig("containerd://fake", config)
			Expect(err).To(BeNil())

			err = c.EnterNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(netns.currentns).To(Equal(containerns))

			err = c.ExitNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(netns.currentns).To(Equal(rootns))
		})

		It("should not enter the container network namespace if it is the same as the root", func() {
			netns.fakens = rootns
			c, err := NewWithConfig("containerd://fake", config)
			Expect(err).To(BeNil())

			err = c.EnterNetworkNamespace()
			Expect(err).To(Not(BeNil()))
		})
	})
})

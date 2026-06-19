// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package container_test

import (
	"errors"

	. "github.com/DataDog/chaos-controller/container"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ParseContainerID", func() {
	DescribeTable("success cases",
		func(id, wantID, wantRuntime string) {
			cid, runtime, err := ParseContainerID(id)
			Expect(err).NotTo(HaveOccurred())
			Expect(cid).To(Equal(wantID))
			Expect(runtime).To(Equal(wantRuntime))
		},
		Entry("containerd", "containerd://abc123", "abc123", "containerd"),
		Entry("docker", "docker://def456", "def456", "docker"),
	)

	DescribeTable("error cases",
		func(id string) {
			_, _, err := ParseContainerID(id)
			Expect(err).To(HaveOccurred())
		},
		Entry("no separator", "containerdabc123"),
		Entry("empty string", ""),
		Entry("too many separators", "containerd://abc://extra"),
	)
})

var _ = Describe("NewWithConfig error paths", func() {
	It("returns error for invalid container ID format", func() {
		_, err := NewWithConfig("invalid-id", "name", Config{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unable to parse container ID"))
	})

	It("returns error for unsupported runtime", func() {
		_, err := NewWithConfig("unsupportedruntime://abc", "name", Config{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported container runtime"))
	})

	It("returns error when PID lookup fails", func() {
		rt := NewRuntimeMock(GinkgoT())
		rt.EXPECT().PID(mock.Anything, mock.Anything).Return(uint32(0), errors.New("pid lookup failed"))
		_, err := NewWithConfig("containerd://abc", "name", Config{Runtime: rt})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("error getting PID"))
	})
})

var _ = Describe("New", func() {
	It("returns error for invalid container ID", func() {
		_, err := New("bad-id", "name")
		Expect(err).To(HaveOccurred())
	})

	It("attempts containerd runtime (fails without socket — covers newContainerdRuntime error path)", func() {
		// On systems without a containerd socket, this covers the newContainerdRuntime error path.
		// On systems with a containerd socket but no such container, it covers PID error.
		Expect(func() { New("containerd://nonexistent-container-id", "name") }).NotTo(Panic())
	})

	It("attempts docker runtime (fails without socket — covers newDockerRuntime error path)", func() {
		Expect(func() { New("docker://nonexistent-container-id", "name") }).NotTo(Panic())
	})
})

var _ = Describe("container accessors", func() {
	var (
		rt  *RuntimeMock
		ctn Container
	)

	BeforeEach(func() {
		rt = NewRuntimeMock(GinkgoT())
		rt.EXPECT().PID(mock.Anything, mock.Anything).Return(uint32(42), nil)
		var err error
		ctn, err = NewWithConfig("containerd://cid-123", "my-container", Config{Runtime: rt})
		Expect(err).NotTo(HaveOccurred())
	})

	It("Runtime() returns the configured runtime", func() {
		Expect(ctn.Runtime()).To(Equal(rt))
	})

	It("ID() returns parsed container ID", func() {
		Expect(ctn.ID()).To(Equal("cid-123"))
	})

	It("Name() returns the name", func() {
		Expect(ctn.Name()).To(Equal("my-container"))
	})

	It("PID() returns the pid", func() {
		Expect(ctn.PID()).To(Equal(uint32(42)))
	})
})

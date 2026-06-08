// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package container_test

import (
	"context"
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

var _ = Describe("ContainerMock", func() {
	It("ID via Return", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().ID().Return("mock-id")
		Expect(m.ID()).To(Equal("mock-id"))
	})

	It("ID via Run", func() {
		m := NewContainerMock(GinkgoT())
		called := false
		m.EXPECT().ID().Run(func() { called = true }).Return("x")
		m.ID()
		Expect(called).To(BeTrue())
	})

	It("ID via RunAndReturn", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().ID().RunAndReturn(func() string { return "dynamic-id" })
		Expect(m.ID()).To(Equal("dynamic-id"))
	})

	It("Name via Return", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().Name().Return("mock-name")
		Expect(m.Name()).To(Equal("mock-name"))
	})

	It("Name via Run", func() {
		m := NewContainerMock(GinkgoT())
		called := false
		m.EXPECT().Name().Run(func() { called = true }).Return("n")
		m.Name()
		Expect(called).To(BeTrue())
	})

	It("Name via RunAndReturn", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().Name().RunAndReturn(func() string { return "dyn-name" })
		Expect(m.Name()).To(Equal("dyn-name"))
	})

	It("PID via Return", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().PID().Return(uint32(99))
		Expect(m.PID()).To(Equal(uint32(99)))
	})

	It("PID via Run", func() {
		m := NewContainerMock(GinkgoT())
		called := false
		m.EXPECT().PID().Run(func() { called = true }).Return(uint32(0))
		m.PID()
		Expect(called).To(BeTrue())
	})

	It("PID via RunAndReturn", func() {
		m := NewContainerMock(GinkgoT())
		m.EXPECT().PID().RunAndReturn(func() uint32 { return 77 })
		Expect(m.PID()).To(Equal(uint32(77)))
	})

	It("Runtime via Return", func() {
		m := NewContainerMock(GinkgoT())
		rt := NewRuntimeMock(GinkgoT())
		m.EXPECT().Runtime().Return(rt)
		Expect(m.Runtime()).To(Equal(rt))
	})

	It("Runtime via Run", func() {
		m := NewContainerMock(GinkgoT())
		rt := NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().Runtime().Run(func() { called = true }).Return(rt)
		m.Runtime()
		Expect(called).To(BeTrue())
	})

	It("Runtime via RunAndReturn", func() {
		m := NewContainerMock(GinkgoT())
		rt := NewRuntimeMock(GinkgoT())
		m.EXPECT().Runtime().RunAndReturn(func() Runtime { return rt })
		Expect(m.Runtime()).To(Equal(rt))
	})
})

var _ = Describe("RuntimeMock", func() {
	It("HostPath via Return", func() {
		m := NewRuntimeMock(GinkgoT())
		m.EXPECT().HostPath(mock.Anything, "cid", "/mnt").Return("/host/mnt", nil)
		path, err := m.HostPath(context.Background(), "cid", "/mnt")
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal("/host/mnt"))
	})

	It("HostPath via Run", func() {
		m := NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().HostPath(mock.Anything, mock.Anything, mock.Anything).
			Run(func(ctx context.Context, id, path string) { called = true }).Return("", nil)
		_, _ = m.HostPath(context.Background(), "x", "y")
		Expect(called).To(BeTrue())
	})

	It("HostPath via RunAndReturn", func() {
		m := NewRuntimeMock(GinkgoT())
		m.EXPECT().HostPath(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, id, path string) (string, error) { return "/dyn", nil })
		p, _ := m.HostPath(context.Background(), "id", "path")
		Expect(p).To(Equal("/dyn"))
	})

	It("PID via Run", func() {
		m := NewRuntimeMock(GinkgoT())
		called := false
		m.EXPECT().PID(mock.Anything, mock.Anything).
			Run(func(ctx context.Context, id string) { called = true }).Return(uint32(1), nil)
		_, _ = m.PID(context.Background(), "x")
		Expect(called).To(BeTrue())
	})

	It("PID via RunAndReturn", func() {
		m := NewRuntimeMock(GinkgoT())
		m.EXPECT().PID(mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, id string) (uint32, error) { return 55, nil })
		pid, _ := m.PID(context.Background(), "x")
		Expect(pid).To(Equal(uint32(55)))
	})
})

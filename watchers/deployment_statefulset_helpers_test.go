// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/DataDog/chaos-controller/watchers"
)

func makeContainer(name, image string) corev1.Container {
	return corev1.Container{Name: name, Image: image}
}

func makePodSpec(initContainers, containers []corev1.Container) *corev1.PodSpec {
	return &corev1.PodSpec{
		InitContainers: initContainers,
		Containers:     containers,
	}
}

var _ = Describe("Hash", func() {
	It("returns non-empty hash for a container", func() {
		c := makeContainer("app", "nginx:latest")
		h, err := watchers.Hash(&c)
		Expect(err).NotTo(HaveOccurred())
		Expect(h).NotTo(BeEmpty())
	})

	It("returns same hash for identical containers", func() {
		c := makeContainer("app", "nginx:latest")
		h1, _ := watchers.Hash(&c)
		h2, _ := watchers.Hash(&c)
		Expect(h1).To(Equal(h2))
	})

	It("returns different hashes for different containers", func() {
		c1 := makeContainer("app", "nginx:1.0")
		c2 := makeContainer("app", "nginx:2.0")
		h1, _ := watchers.Hash(&c1)
		h2, _ := watchers.Hash(&c2)
		Expect(h1).NotTo(Equal(h2))
	})
})

var _ = Describe("HashContainerList", func() {
	It("returns empty map for empty list", func() {
		hashes, err := watchers.HashContainerList(&[]corev1.Container{})
		Expect(err).NotTo(HaveOccurred())
		Expect(hashes).To(BeEmpty())
	})

	It("returns hash per container name", func() {
		containers := []corev1.Container{
			makeContainer("app", "nginx:latest"),
			makeContainer("sidecar", "envoy:latest"),
		}
		hashes, err := watchers.HashContainerList(&containers)
		Expect(err).NotTo(HaveOccurred())
		Expect(hashes).To(HaveLen(2))
		Expect(hashes).To(HaveKey("app"))
		Expect(hashes).To(HaveKey("sidecar"))
	})
})

var _ = Describe("HashPodSpec", func() {
	It("returns hashes for both init containers and containers", func() {
		ps := makePodSpec(
			[]corev1.Container{makeContainer("init", "busybox")},
			[]corev1.Container{makeContainer("app", "nginx")},
		)
		initHashes, containerHashes, err := watchers.HashPodSpec(ps)
		Expect(err).NotTo(HaveOccurred())
		Expect(initHashes).To(HaveKey("init"))
		Expect(containerHashes).To(HaveKey("app"))
	})

	It("handles pod spec with no containers", func() {
		ps := makePodSpec(nil, nil)
		initHashes, containerHashes, err := watchers.HashPodSpec(ps)
		Expect(err).NotTo(HaveOccurred())
		Expect(initHashes).To(BeEmpty())
		Expect(containerHashes).To(BeEmpty())
	})
})

var _ = Describe("HashesChanged", func() {
	It("returns false when hashes are equal", func() {
		h := map[string]string{"app": "abc123"}
		Expect(watchers.HashesChanged(h, h, logger)).To(BeFalse())
	})

	It("returns true when a hash value changed", func() {
		old := map[string]string{"app": "abc123"}
		new := map[string]string{"app": "xyz789"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns true when a container is missing from new", func() {
		old := map[string]string{"app": "abc", "sidecar": "def"}
		new := map[string]string{"app": "abc"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns true when a container is added in new", func() {
		old := map[string]string{"app": "abc"}
		new := map[string]string{"app": "abc", "sidecar": "def"}
		Expect(watchers.HashesChanged(old, new, logger)).To(BeTrue())
	})

	It("returns false for two empty maps", func() {
		Expect(watchers.HashesChanged(map[string]string{}, map[string]string{}, logger)).To(BeFalse())
	})
})

var _ = Describe("ContainersChanged", func() {
	It("returns false when pod specs are identical", func() {
		ps := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx")})
		changed, _, _, err := watchers.ContainersChanged(ps, ps, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeFalse())
	})

	It("returns true when a container image changes", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:1.0")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:2.0")})
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns true when a container is added", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx"), makeContainer("sidecar", "envoy")})
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns true when an init container changes", func() {
		old := makePodSpec([]corev1.Container{makeContainer("init", "busybox:1.0")}, nil)
		new := makePodSpec([]corev1.Container{makeContainer("init", "busybox:2.0")}, nil)
		changed, _, _, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(changed).To(BeTrue())
	})

	It("returns new hashes when changed", func() {
		old := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:1.0")})
		new := makePodSpec(nil, []corev1.Container{makeContainer("app", "nginx:2.0")})
		_, _, newHashes, err := watchers.ContainersChanged(old, new, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(newHashes).To(HaveKey("app"))
	})
})

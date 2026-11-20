// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package container

import (
	"context"
	"os/exec"
	"strings"

	dockerlib "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func activeDockerHost() string {
	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func firstRunningDockerContainerID() string {
	out, err := exec.Command("docker", "ps", "-q", "--no-trunc").Output()
	if err != nil {
		return ""
	}

	lines := strings.Fields(string(out))
	if len(lines) == 0 {
		return ""
	}

	return lines[0]
}

func newTestDockerRuntime() (*dockerRuntime, bool) {
	opts := []dockerlib.Opt{dockerlib.FromEnv}

	if host := activeDockerHost(); host != "" {
		opts = append(opts, dockerlib.WithHost(host))
	}

	c, err := dockerlib.NewClientWithOpts(opts...)
	if err != nil {
		return nil, false
	}

	c.NegotiateAPIVersion(context.Background())

	// Verify connectivity: ping should not return a connection error.
	if _, err := c.Ping(context.Background()); err != nil {
		return nil, false
	}

	return &dockerRuntime{client: c}, true
}

var _ = Describe("dockerRuntime (integration)", func() {
	var (
		rt  *dockerRuntime
		cid string
	)

	BeforeEach(func() {
		cid = firstRunningDockerContainerID()
		if cid == "" {
			Skip("no running docker containers available")
		}

		var ok bool

		rt, ok = newTestDockerRuntime()
		if !ok {
			Skip("docker not accessible")
		}
	})

	It("PID returns non-zero PID for running container", func() {
		pid, err := rt.PID(context.Background(), cid)
		Expect(err).NotTo(HaveOccurred())
		Expect(pid).To(BeNumerically(">", 0))
	})

	It("HostPath returns empty string for non-existent mount path", func() {
		path, err := rt.HostPath(context.Background(), cid, "/nonexistent/mount/path")
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(BeEmpty())
	})

	It("HostPath returns error for non-existent container", func() {
		_, err := rt.HostPath(context.Background(), "nonexistent-container-id-xyz", "/path")
		Expect(err).To(HaveOccurred())
	})

	It("HostPath returns host path for existing mount destination", func() {
		mountDest := getContainerFirstMountDest(cid)
		if mountDest == "" {
			Skip("container has no mounts")
		}

		path, err := rt.HostPath(context.Background(), cid, mountDest)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).NotTo(BeEmpty())
	})
})

func getContainerFirstMountDest(cid string) string {
	out, err := exec.Command("docker", "inspect", cid, "--format", "{{range .Mounts}}{{.Destination}}{{end}}").Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

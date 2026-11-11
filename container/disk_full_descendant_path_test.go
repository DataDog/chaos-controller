// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package container

import (
	"context"
	"io"
	"net/http"
	"strings"

	dockerlib "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type diskFullMountsTransport struct {
	mounts string
}

func (transport diskFullMountsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"Mounts":` + transport.mounts + `}`)),
		Request:    request,
	}, nil
}

var _ = DescribeTable("disk-full descendant host path resolution",
	func(mounts, targetPath, expectedHostPath string) {
		client, err := dockerlib.NewClientWithOpts(
			dockerlib.WithHost("tcp://daemon.invalid"),
			dockerlib.WithHTTPClient(&http.Client{Transport: diskFullMountsTransport{mounts: mounts}}),
			dockerlib.WithVersion("1.45"),
		)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(client.Close)

		runtime := dockerRuntime{client: client}
		hostPath, err := runtime.HostPath(context.Background(), "target", targetPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(hostPath).To(Equal(expectedHostPath))
	},
	Entry("exact destination",
		`[{"Source":"/host/data","Destination":"/data"}]`,
		"/data", "/host/data"),
	Entry("directory beneath a mount",
		`[{"Source":"/host/data","Destination":"/data"}]`,
		"/data/logs", "/host/data/logs"),
	Entry("deepest mount wins even when its parent appears later",
		`[{"Source":"/host/logs","Destination":"/data/logs"},{"Source":"/host/data","Destination":"/data"}]`,
		"/data/logs/app", "/host/logs/app"),
	Entry("a string prefix without a path boundary does not match",
		`[{"Source":"/host/data","Destination":"/data"}]`,
		"/database", ""),
	Entry("root mount contains descendants",
		`[{"Source":"/host/root","Destination":"/"}]`,
		"/data", "/host/root/data"),
	Entry("the last equally specific mount retains precedence",
		`[{"Source":"/host/old","Destination":"/data"},{"Source":"/host/new","Destination":"/data"}]`,
		"/data/file", "/host/new/file"),
)

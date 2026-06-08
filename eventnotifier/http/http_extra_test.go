// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/eventnotifier/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP GetNotifierName", func() {
	It("returns http driver name", func() {
		n := &Notifier{}
		Expect(n.GetNotifierName()).To(Equal(string(types.NotifierDriverHTTP)))
	})
})

var _ = Describe("splitHeaders", func() {
	It("parses valid key:value headers", func() {
		result, err := splitHeaders([]string{"Content-Type:application/json", "X-Foo:bar"})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveKeyWithValue("Content-Type", "application/json"))
		Expect(result).To(HaveKeyWithValue("X-Foo", "bar"))
	})

	It("skips empty header strings", func() {
		result, err := splitHeaders([]string{"", "X-Foo:bar"})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
	})

	It("returns error for invalid header without colon", func() {
		_, err := splitHeaders([]string{"invalid-no-colon"})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid headers"))
	})

	It("returns error for header with too many colons", func() {
		_, err := splitHeaders([]string{"a:b:c"})
		Expect(err).To(HaveOccurred())
	})

	It("returns empty map for nil input", func() {
		result, err := splitHeaders(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})
})

var _ = Describe("New error and headers paths", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "http-notifier-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	It("creates notifier with auth token provider when AuthURL is set", func() {
		notifier, err := New(types.NotifiersCommonConfig{}, Config{
			Disruption: DisruptionConfig{Enabled: true, URL: "http://localhost/disruption"},
			AuthURL:    "http://localhost/auth",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(notifier).NotTo(BeNil())
		Expect(notifier.authTokenProvider).NotTo(BeNil())
	})

	It("returns error when headers file does not exist", func() {
		_, err := New(types.NotifiersCommonConfig{}, Config{
			HeadersFilepath: "/nonexistent/headers.txt",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("headers file not found"))
	})

	It("loads headers from file", func() {
		path := filepath.Join(tmpDir, "headers.txt")
		Expect(os.WriteFile(path, []byte("X-Custom:value"), 0o644)).To(Succeed())
		notifier, err := New(types.NotifiersCommonConfig{}, Config{
			HeadersFilepath: path,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(notifier.headers).To(HaveKeyWithValue("X-Custom", "value"))
	})

	It("returns error for invalid headers in file", func() {
		path := filepath.Join(tmpDir, "bad-headers.txt")
		Expect(os.WriteFile(path, []byte("invalid-no-colon"), 0o644)).To(Succeed())
		_, err := New(types.NotifiersCommonConfig{}, Config{
			HeadersFilepath: path,
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid headers in headers file"))
	})

	It("returns error for invalid auth headers", func() {
		_, err := New(types.NotifiersCommonConfig{}, Config{
			AuthURL:     "http://localhost/auth",
			AuthHeaders: []string{"invalid-no-colon"},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid headers for auth"))
	})
})

var _ = Describe("NewBearerAuthTokenProvider", func() {
	It("creates a provider without error", func() {
		p := NewBearerAuthTokenProvider(&http.Client{}, "http://localhost/auth", nil, "")
		Expect(p).NotTo(BeNil())
	})
})

var _ = Describe("bearerAuthTokenProvider.AuthToken", func() {
	It("returns token from response body when TokenPath is empty", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("my-token"))
		}))
		DeferCleanup(srv.Close)

		p := NewBearerAuthTokenProvider(srv.Client(), srv.URL, nil, "")
		token, err := p.AuthToken(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("my-token"))
	})

	It("returns error when server responds with non-2xx status", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		DeferCleanup(srv.Close)

		p := NewBearerAuthTokenProvider(srv.Client(), srv.URL, nil, "")
		_, err := p.AuthToken(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unexpected status code"))
	})

	It("returns error for invalid URL", func() {
		p := NewBearerAuthTokenProvider(&http.Client{}, "://invalid", nil, "")
		_, err := p.AuthToken(context.Background())
		Expect(err).To(HaveOccurred())
	})

	It("extracts token from JSON body using TokenPath", func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"secret"}`))
		}))
		DeferCleanup(srv.Close)

		p := NewBearerAuthTokenProvider(srv.Client(), srv.URL, nil, "access_token")
		token, err := p.AuthToken(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("secret"))
	})

	It("adds request headers when configured", func() {
		var receivedHeader string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeader = r.Header.Get("X-Auth")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("tok"))
		}))
		DeferCleanup(srv.Close)

		p := NewBearerAuthTokenProvider(srv.Client(), srv.URL, map[string]string{"X-Auth": "secret"}, "")
		_, err := p.AuthToken(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(receivedHeader).To(Equal("secret"))
	})
})

var _ = Describe("bearerAuthTokenProviderMock Run/RunAndReturn", func() {
	It("AuthToken Run callback is invoked", func() {
		m := NewBearerAuthTokenProviderMock(GinkgoT())
		called := false
		m.EXPECT().AuthToken(context.Background()).Run(func(ctx context.Context) { called = true }).Return("", nil)
		_, _ = m.AuthToken(context.Background())
		Expect(called).To(BeTrue())
	})

	It("AuthToken RunAndReturn works", func() {
		m := NewBearerAuthTokenProviderMock(GinkgoT())
		m.EXPECT().AuthToken(context.Background()).RunAndReturn(func(ctx context.Context) (string, error) { return "tok", nil })
		token, err := m.AuthToken(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(token).To(Equal("tok"))
	})
})

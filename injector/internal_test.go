// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getUpstreamDNSFromResolvConf", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "resolv-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	writeResolv := func(content string) string {
		path := filepath.Join(tmpDir, "resolv.conf")
		Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
		return path
	}

	It("returns fallback DNS when file does not exist", func() {
		result := getUpstreamDNSFromResolvConf("/nonexistent/path/resolv.conf")
		Expect(result).To(Equal("8.8.8.8:53"))
	})

	It("returns fallback DNS when file has no nameserver entries", func() {
		path := writeResolv("# only a comment\nsearch example.com\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("8.8.8.8:53"))
	})

	It("parses single IPv4 nameserver and appends port", func() {
		path := writeResolv("nameserver 1.2.3.4\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("1.2.3.4:53"))
	})

	It("parses multiple nameservers as comma-separated list", func() {
		path := writeResolv("nameserver 1.1.1.1\nnameserver 8.8.8.8\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("1.1.1.1:53,8.8.8.8:53"))
	})

	It("wraps IPv6 nameserver in brackets with port", func() {
		path := writeResolv("nameserver ::1\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("[::1]:53"))
	})

	It("passes through already-bracketed IPv6 nameserver", func() {
		path := writeResolv("nameserver [::1]:53\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("[::1]:53"))
	})

	It("skips empty lines and comments", func() {
		path := writeResolv("\n# comment\n  \nnameserver 9.9.9.9\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("9.9.9.9:53"))
	})

	It("ignores malformed nameserver lines with no IP", func() {
		path := writeResolv("nameserver\n")
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("8.8.8.8:53"))
	})

	It("returns fallback DNS when scanner returns error (via a too-long line)", func() {
		// bufio.Scanner returns ErrTooLong when a line exceeds the buffer.
		// Default buffer is 64KB. Create a line that exceeds it.
		longLine := "nameserver " + strings.Repeat("x", 65*1024) + "\n"
		path := writeResolv(longLine)
		// The scanner will error; function should fall back to default DNS
		Expect(getUpstreamDNSFromResolvConf(path)).To(Equal("8.8.8.8:53"))
	})
})

var _ = Describe("standardFileWriter.Write", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "filewriter-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	It("writes data to file in normal mode", func() {
		fw := standardFileWriter{dryRun: false}
		path := filepath.Join(tmpDir, "test.txt")
		Expect(fw.Write(path, 0o644, "hello")).To(Succeed())
		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("hello"))
	})

	It("is a no-op in dry-run mode", func() {
		fw := standardFileWriter{dryRun: true}
		path := filepath.Join(tmpDir, "test.txt")
		Expect(fw.Write(path, 0o644, "hello")).To(Succeed())
		_, err := os.Stat(path)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("returns error for invalid path", func() {
		fw := standardFileWriter{dryRun: false}
		err := fw.Write("/nonexistent/dir/file.txt", 0o644, "data")
		Expect(err).To(HaveOccurred())
	})
})

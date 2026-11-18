// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package disk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("disk struct", func() {
	It("Major returns the stored major number", func() {
		d := disk{major: 8, source: "/dev/sda"}
		Expect(d.Major()).To(Equal(8))
	})

	It("Minor returns the stored minor number", func() {
		d := disk{major: 259, minor: 3, source: "/dev/nvme1n1"}
		Expect(d.Minor()).To(Equal(3))
	})

	It("Source returns the stored source path", func() {
		d := disk{major: 8, source: "/dev/sda"}
		Expect(d.Source()).To(Equal("/dev/sda"))
	})
})

var _ = Describe("FromPath", func() {
	var path, sysfs, bin, device string

	// Use shell builtins so discovery tests need neither real devices nor GNU coreutils.
	command := func(name, output string, status int) {
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\nexit %d\n", output, status)
		Expect(os.WriteFile(filepath.Join(bin, name), []byte(script), 0700)).To(Succeed())
	}

	blockDevice := func(id string, partition bool) {
		device = filepath.Join(path, "sys", "devices", "disk")
		if partition {
			device = filepath.Join(device, "disk1")
		}
		Expect(os.MkdirAll(device, 0700)).To(Succeed())
		target, err := filepath.Rel(sysfs, device)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Symlink(target, filepath.Join(sysfs, id))).To(Succeed())
		if partition {
			Expect(os.WriteFile(filepath.Join(device, "partition"), []byte("1\n"), 0600)).To(Succeed())
		}
	}

	BeforeEach(func() {
		path = GinkgoT().TempDir()
		sysfs = filepath.Join(path, "sys", "dev", "block")
		bin = filepath.Join(path, "bin")
		Expect(os.MkdirAll(sysfs, 0700)).To(Succeed())
		Expect(os.Mkdir(bin, 0700)).To(Succeed())
		GinkgoT().Setenv("PATH", bin)
		command("df", "Filesystem\n/dev/test-disk\n", 0)
		command("ls", "brw-rw---- 1 root disk 8, 17 Jul  7 09:51 /dev/test-disk\n", 0)
	})

	It("returns error for non-existent path", func() {
		_, err := FromPath("/this/path/does/not/exist/chaos-controller-test")
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("retains whole-device identifiers",
		func(major, minor int, separator string) {
			blockDevice(fmt.Sprintf("%d:%d", major, minor), false)
			command("ls", fmt.Sprintf("brw-rw----%s1%sroot%sdisk%s%d,%s%d%sJul  7 09:51 /dev/test-disk\n", separator, separator, separator, separator, major, separator, minor, separator), 0)
			informer, err := fromPath(path, sysfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(informer.Major()).To(Equal(major))
			Expect(informer.Minor()).To(Equal(minor))
			Expect(informer.Source()).To(Equal("/dev/test-disk"))
		},
		Entry("minor zero", 8, 0, " "),
		Entry("NVMe with padded spaces", 259, 3, "   "),
		Entry("second SCSI disk with tabs", 8, 16, "\t"),
	)

	DescribeTable("resolves partitions to the parent disk",
		func(major, minor, parentMajor, parentMinor int) {
			blockDevice(fmt.Sprintf("%d:%d", major, minor), true)
			command("ls", fmt.Sprintf("brw-rw---- 1 root disk %d, %d Jul  7 09:51 /dev/test-disk\n", major, minor), 0)
			parentID := fmt.Sprintf("%d:%d\n", parentMajor, parentMinor)
			Expect(os.WriteFile(filepath.Join(filepath.Dir(device), "dev"), []byte(parentID), 0600)).To(Succeed())
			informer, err := fromPath(path, sysfs)
			Expect(err).NotTo(HaveOccurred())
			Expect(informer.Major()).To(Equal(parentMajor))
			Expect(informer.Minor()).To(Equal(parentMinor))
			Expect(informer.Source()).To(Equal("/dev/test-disk"))
		},
		Entry("first SCSI disk", 8, 1, 8, 0),
		Entry("second SCSI disk", 8, 17, 8, 16),
		Entry("NVMe", 259, 7, 259, 3),
		Entry("different parent major", 259, 7, 8, 16),
	)

	DescribeTable("rejects malformed device output",
		func(output string) {
			command("ls", output, 0)
			_, err := fromPath(path, sysfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected"))
		},
		Entry("empty", ""),
		Entry("regular file", "-rw-rw---- 1 root disk 8, 17 Jul 7 09:51 /dev/test-disk"),
		Entry("character device", "crw-rw---- 1 root disk 8, 17 Jul 7 09:51 /dev/test-disk"),
		Entry("missing comma", "brw-rw---- 1 root disk 8 17 Jul 7 09:51 /dev/test-disk"),
		Entry("missing major", "brw-rw---- 1 root disk , 17 Jul 7 09:51 /dev/test-disk"),
		Entry("invalid major", "brw-rw---- 1 root disk x, 17 Jul 7 09:51 /dev/test-disk"),
		Entry("missing minor", "brw-rw---- 1 root disk 8, Jul 7 09:51 /dev/test-disk"),
		Entry("invalid minor", "brw-rw---- 1 root disk 8, x Jul 7 09:51 /dev/test-disk"),
		Entry("negative minor", "brw-rw---- 1 root disk 8, -1 Jul 7 09:51 /dev/test-disk"),
	)

	DescribeTable("reports command failures",
		func(name string) {
			command(name, "command failed", 1)
			_, err := fromPath(path, sysfs)
			Expect(err).To(MatchError(ContainSubstring("error executing " + name + " command")))
		},
		Entry("df", "df"),
		Entry("ls", "ls"),
	)

	DescribeTable("rejects invalid df output",
		func(output, message string) {
			command("df", output, 0)
			_, err := fromPath(path, sysfs)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
		Entry("missing source", "Filesystem\n", "unexpected df output"),
		Entry("scanner error", "Filesystem\n"+strings.Repeat("x", 65536), "error reading df output"),
	)

	It("reports missing sysfs devices", func() {
		_, err := fromPath(path, sysfs)
		Expect(err).To(MatchError(ContainSubstring("error resolving block device 8:17")))
	})

	It("reports errors checking the partition marker", func() {
		blockDevice("8:17", false)
		Expect(os.Symlink("partition", filepath.Join(device, "partition"))).To(Succeed())
		_, err := fromPath(path, sysfs)
		Expect(err).To(MatchError(ContainSubstring("error checking block device partition")))
	})

	It("reports missing parent identifiers", func() {
		blockDevice("8:17", true)
		_, err := fromPath(path, sysfs)
		Expect(err).To(MatchError(ContainSubstring("error reading parent block device")))
	})

	It("reports unreadable parent identifiers", func() {
		blockDevice("8:17", true)
		Expect(os.Mkdir(filepath.Join(filepath.Dir(device), "dev"), 0700)).To(Succeed())
		_, err := fromPath(path, sysfs)
		Expect(err).To(MatchError(ContainSubstring("error reading parent block device")))
	})

	DescribeTable("rejects malformed parent identifiers",
		func(id string) {
			blockDevice("8:17", true)
			Expect(os.WriteFile(filepath.Join(filepath.Dir(device), "dev"), []byte(id), 0600)).To(Succeed())
			_, err := fromPath(path, sysfs)
			Expect(err).To(MatchError(ContainSubstring("error reading parent block device")))
		},
		Entry("empty", ""),
		Entry("missing separator", "8"),
		Entry("extra separator", "8:16:0"),
		Entry("invalid major", "x:16"),
		Entry("invalid minor", "8:x"),
		Entry("missing minor", "8:"),
		Entry("negative major", "-8:16"),
		Entry("overflow", "8:4294967296"),
	)
})

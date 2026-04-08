// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/types"
)

var _ = Describe("DiskFull", func() {
	var (
		config DiskFullInjectorConfig
		inj    Injector
		spec   v1beta1.DiskFullSpec
		tmpDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "chaos-diskfull-test-*")
		Expect(err).ToNot(HaveOccurred())

		// env vars — set mount host to empty so hostPath = tmpDir directly
		os.Setenv(env.InjectorMountHost, "")

		// config — node level to avoid needing container runtime mock
		config = DiskFullInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
				Disruption: chaosapi.DisruptionArgs{
					Level:          types.DisruptionLevelNode,
					DisruptionName: "test-disruption",
				},
			},
		}

		spec = v1beta1.DiskFullSpec{
			Path:     tmpDir,
			Capacity: "95%",
		}
	})

	AfterEach(func() {
		os.Unsetenv(env.InjectorMountHost)
		os.RemoveAll(tmpDir)
	})

	Describe("NewDiskFullInjector", func() {
		It("should create an injector successfully", func() {
			var err error
			inj, err = NewDiskFullInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(inj).ToNot(BeNil())
		})

		It("should return an error when the path does not exist", func() {
			spec.Path = "/nonexistent/path/that/does/not/exist"
			inj, err := NewDiskFullInjector(spec, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
			Expect(inj).To(BeNil())
		})

		It("should return an error when mount host env var is not set", func() {
			os.Unsetenv(env.InjectorMountHost)
			inj, err := NewDiskFullInjector(spec, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(env.InjectorMountHost))
			Expect(inj).To(BeNil())
		})

		It("should return the correct disruption kind", func() {
			var err error
			inj, err = NewDiskFullInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(inj.GetDisruptionKind())).To(Equal("disk-full"))
		})
	})

	Describe("Inject", func() {
		JustBeforeEach(func() {
			var err error
			inj, err = NewDiskFullInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(inj).ToNot(BeNil())
		})

		Context("with a small allocation that fits in available space", func() {
			BeforeEach(func() {
				// Compute a remaining value that will only allocate 1Mi.
				// remaining = available - 1Mi, so bytesToFill = available - remaining = 1Mi.
				var stat syscall.Statfs_t
				err := syscall.Statfs(tmpDir, &stat)
				Expect(err).ToNot(HaveOccurred())

				availableBytes := stat.Bavail * uint64(stat.Bsize)
				// Leave (available - 2Mi) as remaining, so we allocate ~2Mi minus safety floor = ~1Mi
				targetRemaining := availableBytes - 2*1024*1024
				spec.Capacity = ""
				spec.Remaining = formatBytes(targetRemaining)
			})

			It("should create a ballast file", func() {
				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
				info, statErr := os.Stat(ballastPath)
				Expect(statErr).ToNot(HaveOccurred())
				Expect(info.Size()).To(BeNumerically(">", 0))
			})
		})

		Context("with remaining larger than available space", func() {
			BeforeEach(func() {
				spec.Capacity = ""
				spec.Remaining = "999Ti"
			})

			It("should skip injection without error", func() {
				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
				_, statErr := os.Stat(ballastPath)
				Expect(os.IsNotExist(statErr)).To(BeTrue())
			})
		})

		Context("with dry-run mode", func() {
			BeforeEach(func() {
				config.Disruption.DryRun = true
			})

			It("should not create a ballast file", func() {
				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
				_, statErr := os.Stat(ballastPath)
				Expect(os.IsNotExist(statErr)).To(BeTrue())
			})
		})
	})

	Describe("Inject and Clean round trip", func() {
		It("should create and then remove the ballast file", func() {
			// Compute a remaining value that allocates only ~1Mi
			var stat syscall.Statfs_t
			err := syscall.Statfs(tmpDir, &stat)
			Expect(err).ToNot(HaveOccurred())

			availableBytes := stat.Bavail * uint64(stat.Bsize)
			targetRemaining := availableBytes - 2*1024*1024
			spec.Capacity = ""
			spec.Remaining = formatBytes(targetRemaining)

			inj, err := NewDiskFullInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())

			err = inj.Inject()
			Expect(err).ToNot(HaveOccurred())

			ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
			_, statErr := os.Stat(ballastPath)
			Expect(statErr).ToNot(HaveOccurred())

			err = inj.Clean()
			Expect(err).ToNot(HaveOccurred())

			_, statErr = os.Stat(ballastPath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
	})

	Describe("Clean", func() {
		JustBeforeEach(func() {
			var err error
			inj, err = NewDiskFullInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(inj).ToNot(BeNil())
		})

		Context("when ballast file exists", func() {
			BeforeEach(func() {
				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
				err := os.WriteFile(ballastPath, []byte("ballast"), 0644)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should remove the ballast file", func() {
				err := inj.Clean()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-disruption")
				_, statErr := os.Stat(ballastPath)
				Expect(os.IsNotExist(statErr)).To(BeTrue())
			})
		})

		Context("when ballast file does not exist", func() {
			It("should succeed without error (idempotent)", func() {
				err := inj.Clean()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

// formatBytes formats a byte count as a string suitable for resource.ParseQuantity
func formatBytes(bytes uint64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%dGi", bytes/(1024*1024*1024))
	}

	if bytes >= 1024*1024 {
		return fmt.Sprintf("%dMi", bytes/(1024*1024))
	}

	return fmt.Sprintf("%d", bytes)
}

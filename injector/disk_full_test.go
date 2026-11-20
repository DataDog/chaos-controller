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
	"github.com/DataDog/chaos-controller/fallocate"
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
					Level:               types.DisruptionLevelNode,
					DisruptionName:      "test-disruption",
					DisruptionNamespace: "test-namespace",
					TargetName:          "test-node",
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

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				info, statErr := os.Stat(ballastPath)
				Expect(statErr).ToNot(HaveOccurred())
				Expect(info.Size()).To(BeNumerically(">", 0))
			})
		})

		Context("with capacity mode", func() {
			BeforeEach(func() {
				// Use capacity mode: fill to 95% — allocate a tiny amount relative to the disk
				spec.Capacity = "95%"
				spec.Remaining = ""
			})

			It("should create a ballast file when disk is below target capacity", func() {
				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
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

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
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

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				_, statErr := os.Stat(ballastPath)
				Expect(os.IsNotExist(statErr)).To(BeTrue())
			})
		})

		Context("adversarial: floor does not clamp when remaining is exactly the floor", func() {
			// remaining=1Mi is legal (webhook floor is strictly < 1Mi) and must fill
			// exactly (available-1Mi) bytes with no extra clamping by the runtime floor.
			// To stay host-safe we compute the *expected* fill from statfs and only
			// assert the ballast size, not the post-fill statfs of the whole FS.
			BeforeEach(func() {
				spec.Capacity = ""
				spec.Remaining = "1Mi"
			})

			It("should create a ballast sized to available-1Mi", func() {
				var stat syscall.Statfs_t
				Expect(syscall.Statfs(tmpDir, &stat)).To(Succeed())

				availableBytes := stat.Bavail * uint64(stat.Bsize)
				if availableBytes <= 1024*1024 {
					Skip("tmpdir filesystem has <=1Mi available; floor-skip path would trigger instead")
				}

				// Guard against allocating an unreasonable amount on a developer
				// machine: only run the fill on filesystems with modest free space.
				if availableBytes > 2*1024*1024*1024 {
					Skip("tmpdir filesystem has >2Gi free; refusing to fill down to 1Mi on a shared host filesystem")
				}

				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				info, statErr := os.Stat(ballastPath)
				Expect(statErr).ToNot(HaveOccurred())
				// fill = available - 1Mi (floor equals remaining here, no double-clamp)
				Expect(info.Size()).To(BeNumerically("~", availableBytes-1024*1024, 4096))
			})
		})

		Context("adversarial: AllowNoFloor with remaining 0 does not clamp", func() {
			// With unsafeMode the floor is skipped entirely: remaining=0 must fill
			// the entire available space. Again size-asserted only on small FSes.
			BeforeEach(func() {
				spec.Capacity = ""
				spec.Remaining = "0"
				config.AllowNoFloor = true
			})

			It("should create a ballast sized to the full available space", func() {
				var stat syscall.Statfs_t
				Expect(syscall.Statfs(tmpDir, &stat)).To(Succeed())

				availableBytes := stat.Bavail * uint64(stat.Bsize)
				if availableBytes <= 1024*1024 {
					Skip("tmpdir filesystem has <=1Mi available; nothing meaningful to fill")
				}

				if availableBytes > 2*1024*1024*1024 {
					Skip("tmpdir filesystem has >2Gi free; refusing to fill entirely on a shared host filesystem")
				}

				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				info, statErr := os.Stat(ballastPath)
				Expect(statErr).ToNot(HaveOccurred())
				// fill = available - 0 = available (no floor applied)
				Expect(info.Size()).To(BeNumerically("~", availableBytes, 4096))
			})
		})

		Context("adversarial: runtime skip when volume already past target", func() {
			BeforeEach(func() {
				// Fill the tmpdir with a tiny preexisting file, then use a capacity
				// target that the current usage already exceeds — injection must
				// skip cleanly without touching the disk further.
				spec.Capacity = "1%"
				spec.Remaining = ""
			})

			It("should skip injection without creating a ballast file when used >= 1%", func() {
				var stat syscall.Statfs_t
				Expect(syscall.Statfs(tmpDir, &stat)).To(Succeed())

				totalBytes := stat.Blocks * uint64(stat.Bsize)
				availableBytes := stat.Bavail * uint64(stat.Bsize)
				usedBytes := totalBytes - min(availableBytes, totalBytes)

				// On a filesystem whose real usage is below 1% (pristine tmpfs),
				// create a small preexisting file to push usage over the target.
				ballast := filepath.Join(tmpDir, ".preexisting-fill")
				if usedBytes*100 < totalBytes {
					needed := totalBytes/100 - usedBytes + 1024*1024
					f, err := os.Create(ballast)
					Expect(err).ToNot(HaveOccurred())
					defer f.Close()
					Expect(fallocate.Fallocate(f, 0, int64(needed))).To(Succeed())
				}

				err := inj.Inject()
				Expect(err).ToNot(HaveOccurred())

				chaosBallast := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				_, statErr := os.Stat(chaosBallast)
				Expect(os.IsNotExist(statErr)).To(BeTrue())

				if _, err := os.Stat(ballast); err == nil {
					Expect(os.Remove(ballast)).To(Succeed())
				}
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

			ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
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
				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
				err := os.WriteFile(ballastPath, []byte("ballast"), 0644)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should remove the ballast file", func() {
				err := inj.Clean()
				Expect(err).ToNot(HaveOccurred())

				ballastPath := filepath.Join(tmpDir, ".chaos-diskfull-test-namespace-test-disruption-test-node")
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

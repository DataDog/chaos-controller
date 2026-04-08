// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/fallocate"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// minFreeSpaceBytes is the safety floor enforced unless unsafemode overrides it (1Mi)
	minFreeSpaceBytes = 1024 * 1024
	// ballastFilePrefix is the prefix for ballast files created by the disk full injector
	ballastFilePrefix = ".chaos-diskfull-"
	// EBPFDiskFullWriteCmd is the name of the eBPF binary for write syscall interception
	EBPFDiskFullWriteCmd = "bpf-disk-full-write"
)

type diskFullInjector struct {
	spec        v1beta1.DiskFullSpec
	config      DiskFullInjectorConfig
	hostPath    string
	ballastPath string
}

// DiskFullInjectorConfig is the disk full injector config
type DiskFullInjectorConfig struct {
	Config
	// CmdFactory is required when WriteSyscall is configured (for launching the eBPF binary)
	CmdFactory command.Factory
	// ProcessManager is required when WriteSyscall is configured
	ProcessManager process.Manager
	// BPFConfigInformer is required when WriteSyscall is configured
	BPFConfigInformer ebpf.ConfigInformer
}

// NewDiskFullInjector creates a disk full injector with the given config
func NewDiskFullInjector(spec v1beta1.DiskFullSpec, config DiskFullInjectorConfig) (Injector, error) {
	path := spec.Path

	// get root mount path
	mountHost, ok := os.LookupEnv(env.InjectorMountHost)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountHost)
	}

	// get path from container info if we target a pod
	if config.Disruption.Level == types.DisruptionLevelPod {
		var err error

		path, err = config.TargetContainer.Runtime().HostPath(config.TargetContainer.ID(), spec.Path)
		if err != nil {
			return nil, fmt.Errorf("error resolving host path for disk full disruption: %w", err)
		}

		if len(path) == 0 {
			config.Log.Warnf("could not apply injector on container: %s; %s not found on this targeted container.", config.TargetContainer.Name(), spec.Path)
			return nil, nil
		}
	}

	hostPath := filepath.Clean(mountHost + path)

	// validate path exists
	if _, err := os.Stat(hostPath); err != nil {
		return nil, fmt.Errorf("target path %s does not exist: %w", hostPath, err)
	}

	// initialize eBPF dependencies when writeSyscall is configured
	if spec.WriteSyscall != nil {
		if config.CmdFactory == nil {
			config.CmdFactory = command.NewFactory(config.Disruption.DryRun)
		}

		if config.ProcessManager == nil {
			config.ProcessManager = process.NewManager(config.Disruption.DryRun)
		}

		if config.BPFConfigInformer == nil {
			var err error

			config.BPFConfigInformer, err = ebpf.NewConfigInformer(config.Log, config.Disruption.DryRun, nil, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("could not create an instance of eBPF config informer for the disk full disruption: %w", err)
			}
		}
	}

	ballastPath := filepath.Join(hostPath, ballastFilePrefix+config.Disruption.DisruptionName)

	return &diskFullInjector{
		spec:        spec,
		config:      config,
		hostPath:    hostPath,
		ballastPath: ballastPath,
	}, nil
}

func (i *diskFullInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *diskFullInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskFull
}

func (i *diskFullInjector) Inject() error {
	// Phase 1: Volume fill
	if err := i.injectVolumeFill(); err != nil {
		return err
	}

	// Phase 2: Optional eBPF write syscall interception
	if i.spec.WriteSyscall != nil {
		if err := i.injectWriteSyscall(); err != nil {
			return err
		}
	}

	return nil
}

func (i *diskFullInjector) injectVolumeFill() error {
	// get filesystem stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs(i.hostPath, &stat); err != nil {
		return fmt.Errorf("error getting filesystem stats for %s: %w", i.hostPath, err)
	}

	// Note: on Linux, Blocks/Bavail are in units of Frsize (fragment size), not Bsize.
	// On ext4/xfs (the common case), Bsize == Frsize. We use Bsize here for Darwin
	// compatibility in tests. The injector runs on Linux where this is correct for
	// standard filesystems.
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	// Bavail excludes space reserved for root (~5% on ext4), so we may slightly
	// underestimate bytes to fill. This is the safe direction.
	availableBytes := stat.Bavail * uint64(stat.Bsize)

	bytesToFill, err := i.computeBytesToFill(totalBytes, availableBytes)
	if err != nil {
		return fmt.Errorf("error computing bytes to fill: %w", err)
	}

	// enforce 1Mi safety floor
	if availableBytes > minFreeSpaceBytes && bytesToFill > availableBytes-minFreeSpaceBytes {
		bytesToFill = availableBytes - minFreeSpaceBytes
		i.config.Log.Infow("clamped fill size to enforce 1Mi safety floor",
			"bytesToFill", bytesToFill,
			"availableBytes", availableBytes,
		)
	}

	if bytesToFill <= 0 {
		i.config.Log.Infow("volume already at or past target fill level, skipping injection",
			"totalBytes", totalBytes,
			"availableBytes", availableBytes,
		)

		return nil
	}

	if i.config.Disruption.DryRun {
		i.config.Log.Infow("dry-run: would create ballast file",
			"ballastPath", i.ballastPath,
			"bytesToFill", bytesToFill,
		)

		return nil
	}

	i.config.Log.Infow("injecting disk full disruption",
		"path", i.hostPath,
		"ballastPath", i.ballastPath,
		"bytesToFill", bytesToFill,
		"totalBytes", totalBytes,
		"availableBytes", availableBytes,
	)

	// Create ballast file and allocate space using fallocate syscall.
	// On Linux, this uses fallocate(2) which is instant (metadata-only).
	// Falls back to writing zeros if the filesystem doesn't support fallocate.
	file, err := os.Create(i.ballastPath)
	if err != nil {
		return fmt.Errorf("error creating ballast file %s: %w", i.ballastPath, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			i.config.Log.Warnw("failed to close ballast file", "error", err)
		}
	}()

	if err := fallocate.Fallocate(file, 0, int64(bytesToFill)); err != nil {
		// Clean up partial file on failure
		if removeErr := os.Remove(i.ballastPath); removeErr != nil {
			i.config.Log.Warnw("failed to clean up partial ballast file", "error", removeErr)
		}

		return fmt.Errorf("error allocating disk space: %w", err)
	}

	i.config.Log.Infow("disk full disruption injected successfully",
		"ballastPath", i.ballastPath,
		"bytesToFill", bytesToFill,
	)

	return nil
}

func (i *diskFullInjector) injectWriteSyscall() error {
	if err := i.config.BPFConfigInformer.ValidateRequiredSystemConfig(); err != nil {
		return fmt.Errorf("the disk full write syscall interception needs a kernel supporting eBPF programs: %w", err)
	}

	if !i.config.BPFConfigInformer.GetMapTypes().HavePerfEventArrayMapType {
		return fmt.Errorf("the disk full write syscall interception needs the perf event array map type, but the current kernel does not support this type of map")
	}

	pid := 0
	if i.config.Disruption.Level == types.DisruptionLevelPod {
		pid = int(i.config.TargetContainer.PID())
	}

	exitCode := i.spec.WriteSyscall.GetExitCodeInt()

	probability := "100"
	if i.spec.WriteSyscall.Probability != "" {
		probability = strings.TrimSuffix(i.spec.WriteSyscall.Probability, "%")
	}

	args := []string{
		"-process", strconv.Itoa(pid),
		"-exit-code", strconv.Itoa(exitCode),
		"-probability", probability,
	}

	i.config.Log.Infow("starting eBPF write syscall interception",
		"pid", pid,
		"exitCode", i.spec.WriteSyscall.ExitCode,
		"probability", probability,
	)

	cmd := i.config.CmdFactory.NewCmd(context.Background(), EBPFDiskFullWriteCmd, args)

	bgCmd := command.NewBackgroundCmd(cmd, i.config.Log, i.config.ProcessManager)
	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("unable to run the eBPF disk full write interception: %w", err)
	}

	return nil
}

func (i *diskFullInjector) computeBytesToFill(totalBytes, availableBytes uint64) (uint64, error) {
	if i.spec.Capacity != "" {
		percentStr := strings.TrimSuffix(i.spec.Capacity, "%")

		percent, err := strconv.Atoi(percentStr)
		if err != nil {
			return 0, fmt.Errorf("invalid capacity percentage %q: %w", i.spec.Capacity, err)
		}

		usedBytes := totalBytes - availableBytes
		targetUsed := totalBytes * uint64(percent) / 100

		if targetUsed <= usedBytes {
			return 0, nil
		}

		return targetUsed - usedBytes, nil
	}

	if i.spec.Remaining != "" {
		qty, err := resource.ParseQuantity(i.spec.Remaining)
		if err != nil {
			return 0, fmt.Errorf("invalid remaining quantity %q: %w", i.spec.Remaining, err)
		}

		remainingTarget := uint64(qty.Value())
		if availableBytes <= remainingTarget {
			return 0, nil
		}

		return availableBytes - remainingTarget, nil
	}

	return 0, fmt.Errorf("either capacity or remaining must be set")
}

func (i *diskFullInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *diskFullInjector) Clean() error {
	i.config.Log.Infow("cleaning disk full disruption", "ballastPath", i.ballastPath)

	if err := os.Remove(i.ballastPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			i.config.Log.Infow("ballast file already removed", "ballastPath", i.ballastPath)
			return nil
		}

		return fmt.Errorf("error removing ballast file %s: %w", i.ballastPath, err)
	}

	i.config.Log.Infow("disk full disruption cleaned successfully", "ballastPath", i.ballastPath)

	return nil
}

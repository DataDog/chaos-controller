// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/command"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/env" //nolint:depguard
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/types"
)

type DiskFailureInjector struct {
	spec   v1beta1.DiskFailureSpec
	config DiskFailureInjectorConfig
}

// DiskFailureInjectorConfig is the disk pressure injector config
type DiskFailureInjectorConfig struct {
	Config
	CmdFactory        command.Factory
	ProcessManager    process.Manager
	BPFConfigInformer ebpf.ConfigInformer
	PidNsInumReader   func(pid int) (uint64, error)
	// PidNsSharedChecker reports whether the given PID namespace inode is shared
	// by processes belonging to different containers (shareProcessNamespace: true).
	// The BPF filter scopes by namespace inode, so a shared namespace would
	// disrupt ALL containers in the pod, not just the targeted one.
	PidNsSharedChecker func(targetPID int, nsInum uint64) (bool, error)
}

const EBPFDiskFailureCmd = "bpf-disk-failure"

// NewDiskFailureInjector creates a disk failure injector with the given config
func NewDiskFailureInjector(spec v1beta1.DiskFailureSpec, config DiskFailureInjectorConfig) (Injector, error) {
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
			return nil, fmt.Errorf("could not create an instance of eBPF config informer for the disk failure disruption: %w", err)
		}
	}

	return &DiskFailureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *DiskFailureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *DiskFailureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskFailure
}

func (i *DiskFailureInjector) Inject() error {
	if err := i.config.BPFConfigInformer.ValidateRequiredSystemConfig(); err != nil {
		return fmt.Errorf("the disk failure needs a kernel supporting eBPF programs: %w", err)
	}

	if !i.config.BPFConfigInformer.GetMapTypes().HavePerfEventArrayMapType {
		return fmt.Errorf("the disk failure needs the perf event array map type, but the current kernel does not support this type of map")
	}

	pidNsInum := uint64(0)

	if i.config.Disruption.Level == types.DisruptionLevelPod {
		if i.config.PidNsInumReader == nil {
			mountProc, ok := os.LookupEnv(env.InjectorMountProc)
			if !ok {
				return fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountProc)
			}

			i.config.PidNsInumReader = func(pid int) (uint64, error) {
				var stat syscall.Stat_t

				path := fmt.Sprintf("%s%d/ns/pid", mountProc, pid)
				if err := syscall.Stat(path, &stat); err != nil {
					return 0, fmt.Errorf("cannot read PID namespace inode for pid %d from %s: %w", pid, path, err)
				}

				return stat.Ino, nil
			}

			// Set default shared-namespace detector alongside the inode reader.
			// Both are derived from the same mountProc, so they are initialized together.
			if i.config.PidNsSharedChecker == nil {
				i.config.PidNsSharedChecker = makePidNsSharedChecker(mountProc, i.config.PidNsInumReader)
			}
		}

		pid := int(i.config.TargetContainer.PID())

		var err error

		pidNsInum, err = i.config.PidNsInumReader(pid)
		if err != nil {
			return fmt.Errorf("unable to resolve PID namespace inode: %w", err)
		}

		// Refuse pod-level disk failure when the container shares the host PID
		// namespace (hostPID: true). In that case pidNsInum equals the host's PID
		// namespace inode, so the eBPF filter would match every process on the node.
		hostInum, err := i.config.PidNsInumReader(1)
		if err != nil {
			return fmt.Errorf("unable to resolve host PID namespace inode: %w", err)
		}

		if pidNsInum == hostInum {
			return fmt.Errorf("pod-level disk failure is not supported for containers sharing the host PID namespace (hostPID: true)")
		}

		// Refuse when the namespace is shared between containers (shareProcessNamespace: true).
		// The BPF filter matches by PID namespace inode, so it would disrupt all containers
		// in the shared namespace, not just the targeted one.
		if i.config.PidNsSharedChecker != nil {
			shared, err := i.config.PidNsSharedChecker(pid, pidNsInum)
			if err != nil {
				return fmt.Errorf("unable to check for shared PID namespace: %w", err)
			}

			if shared {
				return fmt.Errorf("pod-level disk failure is not supported for containers using shareProcessNamespace")
			}
		}
	}

	exitCode := 0

	if i.spec.OpenatSyscall != nil {
		exitCode = i.spec.OpenatSyscall.GetExitCodeInt()
	}

	for _, path := range i.spec.Paths {
		args := []string{"-pid-ns-inum", strconv.FormatUint(pidNsInum, 10)}

		if path != "" {
			args = append(args, "-path", path)
		}

		if exitCode != 0 {
			args = append(args, "-exit-code", fmt.Sprintf("%v", exitCode))
		}

		args = append(args, "-probability", strings.TrimSuffix(i.spec.Probability, "%"))

		cmd := i.config.CmdFactory.NewCmd(context.Background(), EBPFDiskFailureCmd, args)

		bgCmd := command.NewBackgroundCmd(cmd, i.config.Log, i.config.ProcessManager)
		if err := bgCmd.Start(); err != nil {
			return fmt.Errorf("unable to run the eBPF disk failure: %w", err)
		}
	}

	return nil
}

func (i *DiskFailureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *DiskFailureInjector) Clean() error {
	return nil
}

// makePidNsSharedChecker returns a checker that detects shareProcessNamespace by
// scanning /proc for processes sharing the same PID namespace inode that belong
// to a different container (identified by a different container cgroup scope).
func makePidNsSharedChecker(mountProc string, inumReader func(int) (uint64, error)) func(int, uint64) (bool, error) {
	return func(targetPID int, targetInum uint64) (bool, error) {
		targetCgroup, err := readCgroupPath(mountProc, targetPID)
		if err != nil {
			return false, fmt.Errorf("cannot read cgroup for pid %d: %w", targetPID, err)
		}

		entries, err := os.ReadDir(mountProc)
		if err != nil {
			return false, fmt.Errorf("cannot read %s: %w", mountProc, err)
		}

		for _, e := range entries {
			pid, err := strconv.Atoi(e.Name())
			if err != nil || pid == targetPID {
				continue
			}

			inum, err := inumReader(pid)
			if err != nil || inum != targetInum {
				continue
			}

			cgroup, err := readCgroupPath(mountProc, pid)
			if err != nil {
				continue
			}

			// Two processes are in the same container if any of their cgroup
			// hierarchy paths is equal or one is a subdirectory of the other
			// (delegated/nested cgroups). This handles both the systemd cgroup
			// driver (.scope suffix) and the cgroupfs driver (/kubepods/.../<id>
			// directories) without needing to parse driver-specific formats.
			if !sameContainerCgroup(cgroup, targetCgroup) {
				return true, nil
			}
		}

		return false, nil
	}
}

// sameContainerCgroup reports whether two /proc/<pid>/cgroup file contents
// belong to the same container. Two processes are in the same container when
// any of their cgroup hierarchy paths is equal or one is a subdirectory of
// the other (delegated/nested cgroups inside the container).
//
// This handles both the systemd cgroup driver (paths ending with .scope) and
// the cgroupfs driver (/kubepods/.../<container-id> directories) without
// requiring driver-specific parsing.
func sameContainerCgroup(cgroupA, cgroupB string) bool {
	pathsA := extractCgroupPaths(cgroupA)
	pathsB := extractCgroupPaths(cgroupB)

	// If no non-root paths were found in either file (e.g. all controllers
	// at "/" on an unusual cgroup v1 layout), we cannot distinguish containers
	// via cgroup paths. Treat conservatively as the same container to avoid
	// falsely detecting shareProcessNamespace and blocking valid injections.
	if len(pathsA) == 0 || len(pathsB) == 0 {
		return true
	}

	for _, a := range pathsA {
		for _, b := range pathsB {
			if a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") {
				return true
			}
		}
	}

	return false
}

func extractCgroupPaths(content string) []string {
	var paths []string

	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		path := parts[2]
		// Skip root-only paths ("/"). On cgroup v1 nodes, unused controllers
		// (e.g. rdma) set every process's path to "/". Including these would
		// cause sameContainerCgroup to match unrelated containers via the common
		// "/" prefix before ever reaching container-specific hierarchies.
		if path == "/" {
			continue
		}

		paths = append(paths, path)
	}

	return paths
}

func readCgroupPath(mountProc string, pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("%s%d/cgroup", mountProc, pid))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

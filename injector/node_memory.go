// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/env"
	"github.com/prometheus/procfs"
)

// readNodeMemoryStats reads total and used node memory from one meminfo snapshot.
// MemAvailable excludes reclaimable memory from usage, unlike MemFree.
func readNodeMemoryStats() (int64, int64, error) {
	mountProc := os.Getenv(env.InjectorMountProc)
	if mountProc == "" {
		mountProc = procfs.DefaultMountPoint
	}

	nodeProc, err := procfs.NewFS(mountProc)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open node procfs: %w", err)
	}

	info, err := nodeProc.Meminfo()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to read node meminfo: %w", err)
	}

	if info.MemTotalBytes == nil || info.MemAvailableBytes == nil {
		return 0, 0, fmt.Errorf("meminfo must include MemTotal and MemAvailable")
	}

	total, available := *info.MemTotalBytes, *info.MemAvailableBytes
	if total == 0 || available > total {
		return 0, 0, fmt.Errorf("invalid node memory statistics: total=%d, available=%d", total, available)
	}

	return int64(total), int64(total - available), nil
}

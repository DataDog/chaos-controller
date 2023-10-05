// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/types"
)

type diskPressureInjector struct {
	spec   v1beta1.DiskPressureSpec
	config DiskPressureInjectorConfig
}

// DiskPressureInjectorConfig is the disk pressure injector config
type DiskPressureInjectorConfig struct {
	Config
	Informer disk.Informer
}

// Possible throttle modes enum
type diskPressureThrottleMode int

const (
	diskPressureThrottleModeRead diskPressureThrottleMode = iota
	diskPressureThrottleModeWrite
)

const diskPressureBlkioControllerName = "blkio"

// NewDiskPressureInjector creates a disk pressure injector with the given config
func NewDiskPressureInjector(spec v1beta1.DiskPressureSpec, config DiskPressureInjectorConfig) (Injector, error) {
	var err error

	path := spec.Path

	// get root mount path
	mountHost, ok := os.LookupEnv(env.InjectorMountHost)
	if !ok {
		return nil, fmt.Errorf("environment variable %s doesn't exist", env.InjectorMountHost)
	}

	// get path from container info if we target a pod
	if config.Disruption.Level == types.DisruptionLevelPod {
		// get host path from mount path
		path, err = config.TargetContainer.Runtime().HostPath(config.TargetContainer.ID(), spec.Path)
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		if err == nil && len(path) == 0 {
			config.Log.Warnf("could not apply injector on container: %s; %s not found on this targeted container.", config.TargetContainer.Name(), spec.Path)
			return nil, nil
		}
	}

	if config.Informer == nil {
		informer, err := disk.FromPath(filepath.Clean(mountHost + path))
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		config.Informer = informer
	}

	return &diskPressureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i *diskPressureInjector) TargetName() string {
	return i.config.TargetName()
}

func (i *diskPressureInjector) GetDisruptionKind() types.DisruptionKindName {
	return types.DisruptionKindDiskPressure
}

func (i *diskPressureInjector) Inject() error {
	// add read throttle
	if i.spec.Throttling.ReadBytesPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeRead), i.formatThrottle(*i.spec.Throttling.ReadBytesPerSec, diskPressureThrottleModeRead)); err != nil {
			return fmt.Errorf("error throttling disk read: %w", err)
		}

		i.config.Log.Infow("read throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.ReadBytesPerSec)
	}

	// add write throttle
	if i.spec.Throttling.WriteBytesPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWrite), i.formatThrottle(*i.spec.Throttling.WriteBytesPerSec, diskPressureThrottleModeWrite)); err != nil {
			return fmt.Errorf("error throttling disk write: %w", err)
		}

		i.config.Log.Infow("write throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.WriteBytesPerSec)
	}

	return nil
}

func (i *diskPressureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *diskPressureInjector) Clean() error {
	// clean read throttle
	i.config.Log.Infow("cleaning disk read throttle", "device", i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeRead), i.formatThrottle(0, diskPressureThrottleModeRead)); err != nil {
		return fmt.Errorf("error cleaning read disk throttle: %w", err)
	}

	// clean write throttle
	i.config.Log.Infow("cleaning disk write throttle", "device", i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWrite), i.formatThrottle(0, diskPressureThrottleModeWrite)); err != nil {
		return fmt.Errorf("error cleaning write disk throttle: %w", err)
	}

	return nil
}

// formatThrottle formats the throttle data to write to the IO throttling cgroup file
// depending on the cgroup version
func (i *diskPressureInjector) formatThrottle(throttle int, mode diskPressureThrottleMode) string {
	// cgroups v2 io.max file used for throttling expects a slightly different data format
	// example: 252:0 rbps=1024 wbps=max
	if i.config.Cgroup.IsCgroupV2() {
		// resetting the throttling value must be done by setting
		// the value to "max" instead of "0" in cgroups v1
		sThrottle := ""
		if throttle == 0 {
			sThrottle = "max"
		} else {
			sThrottle = strconv.Itoa(throttle)
		}

		// the file can be used to configure both read and write throttling (both iops and bps too)
		// to set that value, it is now a key/value pair (rbps for read throttling, wbps for write throttling)
		switch mode {
		case diskPressureThrottleModeRead:
			return fmt.Sprintf("%d:0 rbps=%s", i.config.Informer.Major(), sThrottle)
		case diskPressureThrottleModeWrite:
			return fmt.Sprintf("%d:0 wbps=%s", i.config.Informer.Major(), sThrottle)
		default:
			return "" // should never be used
		}
	}

	// cgroups v1 throttling format is much simple and only takes the bps value
	// example: 252:0 1024
	return fmt.Sprintf("%d:0 %d", i.config.Informer.Major(), throttle)
}

// getThrottleFilename returns the filename to use to write IO read/write throttling data to
// depending on the cgroup version
func (i *diskPressureInjector) getThrottleFilename(mode diskPressureThrottleMode) string {
	// cgroups v2 uses a single file to handle all kind of IO throttling
	// read and write, iops and bps
	if i.config.Cgroup.IsCgroupV2() {
		return "io.max"
	}

	// cgroups v1 uses separate files for both the mode and the unit
	// - read and bps
	// - write and bps
	// - read and iops (unused here)
	// - write and iops (unused here)
	switch mode {
	case diskPressureThrottleModeRead:
		return "blkio.throttle.read_bps_device"
	case diskPressureThrottleModeWrite:
		return "blkio.throttle.write_bps_device"
	}

	return "" // should never be used
}

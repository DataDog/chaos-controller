// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/o11y/tags"
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
	diskPressureThrottleModeReadBps diskPressureThrottleMode = iota
	diskPressureThrottleModeWriteBps
	diskPressureThrottleModeReadIops
	diskPressureThrottleModeWriteIops
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// get host path from mount path
		path, err = config.TargetContainer.Runtime().HostPath(ctx, config.TargetContainer.ID(), spec.Path)
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		if len(path) == 0 {
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
	// add read bytes-per-second throttle
	if i.spec.Throttling.ReadBytesPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeReadBps), i.formatThrottle(*i.spec.Throttling.ReadBytesPerSec, diskPressureThrottleModeReadBps)); err != nil {
			return fmt.Errorf("error throttling disk read: %w", err)
		}

		i.config.Log.Infow("read throttling injected", tags.DeviceKey, i.config.Informer.Source(), tags.BpsKey, *i.spec.Throttling.ReadBytesPerSec)
	}

	// add write bytes-per-second throttle
	if i.spec.Throttling.WriteBytesPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWriteBps), i.formatThrottle(*i.spec.Throttling.WriteBytesPerSec, diskPressureThrottleModeWriteBps)); err != nil {
			return fmt.Errorf("error throttling disk write: %w", err)
		}

		i.config.Log.Infow("write throttling injected", tags.DeviceKey, i.config.Informer.Source(), tags.BpsKey, *i.spec.Throttling.WriteBytesPerSec)
	}

	// add read iops throttle
	if i.spec.Throttling.ReadIOPSPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeReadIops), i.formatThrottle(*i.spec.Throttling.ReadIOPSPerSec, diskPressureThrottleModeReadIops)); err != nil {
			return fmt.Errorf("error throttling disk read iops: %w", err)
		}

		i.config.Log.Infow("read iops throttling injected", tags.DeviceKey, i.config.Informer.Source(), tags.IopsKey, *i.spec.Throttling.ReadIOPSPerSec)
	}

	// add write iops throttle
	if i.spec.Throttling.WriteIOPSPerSec != nil {
		if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWriteIops), i.formatThrottle(*i.spec.Throttling.WriteIOPSPerSec, diskPressureThrottleModeWriteIops)); err != nil {
			return fmt.Errorf("error throttling disk write iops: %w", err)
		}

		i.config.Log.Infow("write iops throttling injected", tags.DeviceKey, i.config.Informer.Source(), tags.IopsKey, *i.spec.Throttling.WriteIOPSPerSec)
	}

	return nil
}

func (i *diskPressureInjector) UpdateConfig(config Config) {
	i.config.Config = config
}

func (i *diskPressureInjector) Clean() error {
	// clean read bytes-per-second throttle
	i.config.Log.Infow("cleaning disk read throttle", tags.DeviceKey, i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeReadBps), i.formatThrottle(0, diskPressureThrottleModeReadBps)); err != nil {
		return fmt.Errorf("error cleaning read disk throttle: %w", err)
	}

	// clean write bytes-per-second throttle
	i.config.Log.Infow("cleaning disk write throttle", tags.DeviceKey, i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWriteBps), i.formatThrottle(0, diskPressureThrottleModeWriteBps)); err != nil {
		return fmt.Errorf("error cleaning write disk throttle: %w", err)
	}

	// clean read iops throttle
	i.config.Log.Infow("cleaning disk read iops throttle", tags.DeviceKey, i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeReadIops), i.formatThrottle(0, diskPressureThrottleModeReadIops)); err != nil {
		return fmt.Errorf("error cleaning read disk iops throttle: %w", err)
	}

	// clean write iops throttle
	i.config.Log.Infow("cleaning disk write iops throttle", tags.DeviceKey, i.config.Informer.Source())

	if err := i.config.Cgroup.Write(diskPressureBlkioControllerName, i.getThrottleFilename(diskPressureThrottleModeWriteIops), i.formatThrottle(0, diskPressureThrottleModeWriteIops)); err != nil {
		return fmt.Errorf("error cleaning write disk iops throttle: %w", err)
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
		// to set that value, it is now a key/value pair (rbps/wbps for bandwidth, riops/wiops for iops)
		switch mode {
		case diskPressureThrottleModeReadBps:
			return fmt.Sprintf("%d:0 rbps=%s", i.config.Informer.Major(), sThrottle)
		case diskPressureThrottleModeWriteBps:
			return fmt.Sprintf("%d:0 wbps=%s", i.config.Informer.Major(), sThrottle)
		case diskPressureThrottleModeReadIops:
			return fmt.Sprintf("%d:0 riops=%s", i.config.Informer.Major(), sThrottle)
		case diskPressureThrottleModeWriteIops:
			return fmt.Sprintf("%d:0 wiops=%s", i.config.Informer.Major(), sThrottle)
		default:
			return "" // should never be used
		}
	}

	// cgroups v1 throttling format is much simple and only takes the value
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
	// - read and iops
	// - write and iops
	switch mode {
	case diskPressureThrottleModeReadBps:
		return "blkio.throttle.read_bps_device"
	case diskPressureThrottleModeWriteBps:
		return "blkio.throttle.write_bps_device"
	case diskPressureThrottleModeReadIops:
		return "blkio.throttle.read_iops_device"
	case diskPressureThrottleModeWriteIops:
		return "blkio.throttle.write_iops_device"
	}

	return "" // should never be used
}

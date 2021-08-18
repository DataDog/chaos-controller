// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"fmt"
	"os"
	"path/filepath"

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
	if config.Level == types.DisruptionLevelPod {
		// get host path from mount path
		path, err = config.TargetContainer.Runtime().HostPath(config.TargetContainer.ID(), spec.Path)
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}
	}

	if config.Informer == nil {
		informer, err := disk.FromPath(filepath.Clean(mountHost + path))
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		config.Informer = informer
	}

	return diskPressureInjector{
		spec:   spec,
		config: config,
	}, nil
}

func (i diskPressureInjector) Inject() error {
	// add read throttle
	if i.spec.Throttling.ReadBytesPerSec != nil {
		if err := i.config.Cgroup.DiskThrottleRead(i.config.Informer.Major(), *i.spec.Throttling.ReadBytesPerSec); err != nil {
			return fmt.Errorf("error throttling disk read: %w", err)
		}

		i.config.Log.Infow("read throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.ReadBytesPerSec)
	}

	// add write throttle
	if i.spec.Throttling.WriteBytesPerSec != nil {
		if err := i.config.Cgroup.DiskThrottleWrite(i.config.Informer.Major(), *i.spec.Throttling.WriteBytesPerSec); err != nil {
			return fmt.Errorf("error throttling disk write: %w", err)
		}

		i.config.Log.Infow("write throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.WriteBytesPerSec)
	}

	return nil
}

func (i diskPressureInjector) Clean() error {
	// clean read throttle
	i.config.Log.Infow("cleaning disk read throttle", "device", i.config.Informer.Source())

	if err := i.config.Cgroup.DiskThrottleRead(i.config.Informer.Major(), 0); err != nil {
		return fmt.Errorf("error cleaning read disk throttle: %w", err)
	}

	// clean write throttle
	i.config.Log.Infow("cleaning disk write throttle", "device", i.config.Informer.Source())

	if err := i.config.Cgroup.DiskThrottleWrite(i.config.Informer.Major(), 0); err != nil {
		return fmt.Errorf("error cleaning write disk throttle: %w", err)
	}

	return nil
}

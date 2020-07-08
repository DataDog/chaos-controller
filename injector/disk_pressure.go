// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/disk"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

type diskPressureInjector struct {
	containerInjector
	spec   v1beta1.DiskPressureSpec
	config DiskPressureInjectorConfig
}

// DiskPressureInjectorConfig is the disk pressure injector config
type DiskPressureInjectorConfig struct {
	Informer disk.Informer
}

// NewDiskPressureInjector creates a disk pressure injector with the default config
func NewDiskPressureInjector(uid string, spec v1beta1.DiskPressureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) (Injector, error) {
	return NewDiskPressureInjectorWithConfig(uid, spec, ctn, log, ms, DiskPressureInjectorConfig{})
}

// NewDiskPressureInjectorWithConfig creates a disk pressure injector with the given config
func NewDiskPressureInjectorWithConfig(uid string, spec v1beta1.DiskPressureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config DiskPressureInjectorConfig) (Injector, error) {
	if config.Informer == nil {
		// get host path from mount path
		hostPath, err := ctn.Runtime().HostPath(ctn.ID(), spec.Path)
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		// get disk informer from host path
		informer, err := disk.FromPath("/mnt/host/" + hostPath)
		if err != nil {
			return nil, fmt.Errorf("error initializing disk informer: %w", err)
		}

		config.Informer = informer
	}

	return diskPressureInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindDiskPressure,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}, nil
}

func (i diskPressureInjector) Inject() {
	// add read throttle
	if i.spec.Throttling.ReadBytesPerSec != nil {
		if err := i.container.Cgroup().DiskThrottleRead(i.config.Informer.Major(), *i.spec.Throttling.ReadBytesPerSec); err != nil {
			i.log.Fatalf("error throttling disk read: %v", err)
		}

		i.log.Infow("read throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.ReadBytesPerSec)
	}

	// add write throttle
	if i.spec.Throttling.WriteBytesPerSec != nil {
		if err := i.container.Cgroup().DiskThrottleWrite(i.config.Informer.Major(), *i.spec.Throttling.WriteBytesPerSec); err != nil {
			i.log.Fatalf("error throttling disk write: %v", err)
		}

		i.log.Infow("write throttling injected", "device", i.config.Informer.Source(), "bps", *i.spec.Throttling.WriteBytesPerSec)
	}
}
func (i diskPressureInjector) Clean() {
	// clean read throttle
	i.log.Infow("cleaning disk read throttle", "device", i.config.Informer.Source())

	if err := i.container.Cgroup().DiskThrottleRead(i.config.Informer.Major(), 0); err != nil {
		i.log.Fatalf("error cleaning read disk throttle: %v", err)
	}

	// clean write throttle
	i.log.Infow("cleaning disk write throttle", "device", i.config.Informer.Source())

	if err := i.container.Cgroup().DiskThrottleWrite(i.config.Informer.Major(), 0); err != nil {
		i.log.Fatalf("error cleaning write disk throttle: %v", err)
	}
}

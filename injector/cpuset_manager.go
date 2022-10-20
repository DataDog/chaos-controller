// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package injector

import (
	"fmt"
	"github.com/DataDog/chaos-controller/cpuset"
	"go.uber.org/zap"
	"runtime"
	"sync"
)

type StresserManager interface {
	TrackInjectorCores(config CPUPressureInjectorConfig) (cpuset.CPUSet, error)
	CoresToBeStressed() cpuset.CPUSet
	IsCoreAlreadyStressed(core int) bool
	TrackCoreAlreadyStressed(core int, stresserPID int)
	StresserPIDs() map[int]int
}

type cpuStressserManager struct {
	mutex              sync.RWMutex
	coresToBeStressed  cpuset.CPUSet
	stresserPIDPerCore map[int]int
	log                *zap.SugaredLogger
}

func NewCPUStresserManager(log *zap.SugaredLogger) StresserManager {
	return &cpuStressserManager{
		coresToBeStressed:  cpuset.NewCPUSet(),
		stresserPIDPerCore: map[int]int{},
		log:                log,
	}
}

func (manager *cpuStressserManager) setCoresToBeStressed(cores cpuset.CPUSet) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.coresToBeStressed = cores
	oldMaxProcs := runtime.GOMAXPROCS(cores.Size())

	manager.log.Infof("changed GOMAXPROCS value from %d to %d", oldMaxProcs, cores.Size())
}

func (manager *cpuStressserManager) CoresToBeStressed() cpuset.CPUSet {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.coresToBeStressed
}

func (manager *cpuStressserManager) IsCoreAlreadyStressed(core int) bool {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	_, isCoreStressed := manager.stresserPIDPerCore[core]
	return isCoreStressed
}

func (manager *cpuStressserManager) TrackCoreAlreadyStressed(core int, stresserPID int) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.stresserPIDPerCore[core] = stresserPID
}

func (manager *cpuStressserManager) StresserPIDs() map[int]int {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.stresserPIDPerCore
}

func (manager *cpuStressserManager) TrackInjectorCores(config CPUPressureInjectorConfig) (cpuset.CPUSet, error) {
	// read cpuset allocated cores
	config.Log.Infow("retrieving target cpuset allocated cores")

	cpusetCores, err := config.Cgroup.Read("cpuset", "cpuset.cpus")
	if err != nil {
		return cpuset.NewCPUSet(), fmt.Errorf("failed to read the target allocated cpus from the cpuset cgroup: %w", err)
	}

	// parse allocated cores
	cores, err := cpuset.Parse(cpusetCores)
	if err != nil {
		return cpuset.NewCPUSet(), fmt.Errorf("error parsing cpuset allocated cores: %w", err)
	}

	manager.setCoresToBeStressed(manager.coresToBeStressed.Union(cores))

	return cores, nil
}

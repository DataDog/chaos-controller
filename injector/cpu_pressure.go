// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/process"
	"github.com/DataDog/chaos-controller/stress"
	"github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
)

type cpuPressureInjector struct {
	containerInjector
	spec   v1beta1.CPUPressureSpec
	config CPUPressureInjectorConfig
}

// CPUPressureInjectorConfig is the CPU pressure injector config
type CPUPressureInjectorConfig struct {
	Stresser       stress.Stresser
	StresserExit   chan struct{}
	ProcessManager process.Manager
	SignalHandler  chan os.Signal
}

// NewCPUPressureInjector creates a CPU pressure injector with the default config
func NewCPUPressureInjector(uid string, spec v1beta1.CPUPressureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink) Injector {
	return NewCPUPressureInjectorWithConfig(uid, spec, ctn, log, ms, CPUPressureInjectorConfig{})
}

// NewCPUPressureInjectorWithConfig creates a CPU pressure injector with the given config
func NewCPUPressureInjectorWithConfig(uid string, spec v1beta1.CPUPressureSpec, ctn container.Container, log *zap.SugaredLogger, ms metrics.Sink, config CPUPressureInjectorConfig) Injector {
	// create stresser
	if config.Stresser == nil {
		config.Stresser = stress.NewCPU(runtime.NumCPU())
	}

	if config.StresserExit == nil {
		config.StresserExit = make(chan struct{})
	}

	// signal handling
	if config.SignalHandler == nil {
		config.SignalHandler = make(chan os.Signal)
	}

	signal.Notify(config.SignalHandler, syscall.SIGINT, syscall.SIGTERM)

	// create process manager
	if config.ProcessManager == nil {
		config.ProcessManager = process.NewManager()
	}

	return cpuPressureInjector{
		containerInjector: containerInjector{
			injector: injector{
				uid:  uid,
				log:  log,
				ms:   ms,
				kind: types.DisruptionKindCPUPressure,
			},
			container: ctn,
		},
		spec:   spec,
		config: config,
	}
}

func (i cpuPressureInjector) Inject() {
	// join container CPU cgroup
	i.log.Infow("joining container CPU cgroup", "container", i.container.ID())

	if err := i.container.Cgroup().JoinCPU(); err != nil {
		i.log.Fatalw("failed to inject CPU pressure", "error", err)
	}

	// prioritize the current process
	i.log.Info("highering current process priority")

	if err := i.config.ProcessManager.Prioritize(); err != nil {
		i.log.Fatalw("error highering the current process priority", "error", err)
	}

	// start eating CPU in separate goroutines
	// we start one goroutine per available CPU
	i.log.Infow("initializing load generator routines", "routines", runtime.NumCPU())

	go i.config.Stresser.Stress(i.config.StresserExit)

	// wait until the process is killed
	sig := <-i.config.SignalHandler

	i.log.Infow("received exit signal, killing the cpu stresser routines...", "signal", sig.String())

	// exit the stresser
	i.config.StresserExit <- struct{}{}

	i.log.Info("all routines has been killed, exiting")
}

func (i cpuPressureInjector) Clean() {}

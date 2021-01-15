// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"github.com/DataDog/chaos-controller/netns"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	readinessProbeFile = "/tmp/readiness_probe"
)

var rootCmd = &cobra.Command{
	Use:               "chaos-injector",
	Short:             "Datadog chaos failures injection application",
	Run:               nil,
	PersistentPostRun: cleanAndExit,
}

var (
	log         *zap.SugaredLogger
	dryRun      bool
	ms          metrics.Sink
	sink        string
	level       string
	containerID string
	config      injector.Config
	signals     chan os.Signal
	inj         injector.Injector
)

func init() {
	rootCmd.AddCommand(networkDisruptionCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(cpuPressureCmd)
	rootCmd.AddCommand(diskPressureCmd)
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	rootCmd.PersistentFlags().StringVar(&level, "level", "", "Level of injection (either pod or node)")
	rootCmd.PersistentFlags().StringVar(&containerID, "container-id", "", "Targeted container ID")
	_ = cobra.MarkFlagRequired(rootCmd.PersistentFlags(), "level")

	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initMetricsSink)
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initExitSignalsHandler)
}

func main() {
	// handle metrics sink client close on exit
	defer func() {
		log.Infow("closing metrics sink client before exiting", "sink", ms.GetSinkName())

		if err := ms.Close(); err != nil {
			log.Errorw("error closing metrics sink client", "error", err, "sink", ms.GetSinkName())
		}
	}()

	// execute command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// initLogger initializes a zap logger
func initLogger() {
	// prepare logger
	zapInstance, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("error while creating logger: %v", err)
		os.Exit(2)
	}

	log = zapInstance.Sugar()
}

// initMetricsSink initializes a metrics sink depending on the given flag
func initMetricsSink() {
	var err error

	ms, err = metrics.GetSink(types.SinkDriver(sink), types.SinkAppInjector)
	if err != nil {
		log.Errorw("error while creating metric sink, switching to noop sink", "error", err)

		ms, _ = metrics.GetSink(types.SinkDriverNoop, types.SinkAppInjector)
	}
}

// initConfig initializes the injector config and main components from the given flags
func initConfig() {
	var (
		err        error
		cgroupPath string
		pid        uint32
		ctn        container.Container
	)

	// log when dry-run mode is enabled
	if dryRun {
		log.Warn("dry run mode enabled, no disruption will be injected but most of the commands will still be executed to simulate it as much as possible")
	}

	switch level {
	case chaostypes.DisruptionLevelPod:
		// check for container ID flag
		if containerID == "" {
			log.Fatal("--container-id flag must be passed when --level=pod")
		}

		// retrieve container info
		ctn, err = container.New(containerID)
		if err != nil {
			log.Fatalw("can't create container object", "error", err)
		}

		cgroupPath = ctn.CgroupPath()
		pid = ctn.PID()
	case chaostypes.DisruptionLevelNode:
		cgroupPath = ""
		pid = 1
	default:
		log.Fatalf("unknown level: %s", level)
	}

	// create cgroup manager
	cgroupMgr, err := cgroup.NewManager(dryRun, cgroupPath)
	if err != nil {
		log.Fatalw("error creating cgroup manager", "error", err)
	}

	// create network namespace manager
	netnsMgr, err := netns.NewManager(pid)
	if err != nil {
		log.Fatalw("error creating network namespace manager", "error", err)
	}

	config = injector.Config{
		DryRun:      dryRun,
		Log:         log,
		MetricsSink: ms,
		Level:       chaostypes.DisruptionLevel(level),
		Container:   ctn,
		Cgroup:      cgroupMgr,
		Netns:       netnsMgr,
	}
}

// initExitSignalsHandler initializes the exit signal handler
func initExitSignalsHandler() {
	signals = make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
}

// injectAndWait injects the disruption with the configured injector and waits
// for an exit signal to be sent
func injectAndWait(cmd *cobra.Command, args []string) {
	log.Info("injecting the disruption")

	// start injection, do not fatal on error so we keep the pod
	// running, allowing the cleanup to happen
	if err := inj.Inject(); err != nil {
		handleMetricError(ms.MetricInjected(false, cmd.Name(), nil))
		log.Errorw("disruption injection failed", "error", err)
	} else {
		// create and write readiness probe file if injection succeeded so the pod is marked as ready
		if err := ioutil.WriteFile(readinessProbeFile, []byte("1"), 0400); err != nil {
			log.Errorw("error writing readiness probe file", "error", err)
		}

		handleMetricError(ms.MetricInjected(true, cmd.Name(), nil))
		log.Info("disruption injected, now waiting for an exit signal")
	}

	// wait for an exit signal, this is a blocking call
	sig := <-signals

	log.Infow("an exit signal has been received", "signal", sig.String())
}

// cleanAndExit cleans the disruption with the configured injector and exits nicely
func cleanAndExit(cmd *cobra.Command, args []string) {
	log.Info("cleaning the disruption")

	// start cleanup which is retried up to 3 times using an exponential backoff algorithm
	if err := backoff.RetryNotify(inj.Clean, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), retryNotifyHandler); err != nil {
		handleMetricError(ms.MetricCleaned(false, cmd.Name(), nil))
		log.Fatalw("disruption cleanup failed", "error", err)
	}

	handleMetricError(ms.MetricCleaned(true, cmd.Name(), nil))
	log.Info("disruption cleaned, now exiting")
}

// handleMetricError logs the given error if not nil
func handleMetricError(err error) {
	if err != nil {
		log.Errorw("error sending metric", "sink", ms.GetSinkName(), "error", err)
	}
}

// retryNotifyHandler is called when the cleanup fails
// it logs the error and the time to wait before the next retry
func retryNotifyHandler(err error, delay time.Duration) {
	log.Errorw("disruption cleanup failed", "error", err)
	log.Infof("retrying cleanup in %s", delay.String())
}

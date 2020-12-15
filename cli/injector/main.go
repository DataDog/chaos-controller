// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package main

import (
	"fmt"
	"os"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"github.com/DataDog/chaos-controller/netns"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "chaos-injector",
	Short: "Datadog chaos failures injection application",
	Run:   nil,
}

var log *zap.SugaredLogger
var ms metrics.Sink
var sink string
var level string
var containerID string
var config injector.Config

func init() {
	rootCmd.AddCommand(networkDisruptionCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(cpuPressureCmd)
	rootCmd.AddCommand(diskPressureCmd)
	rootCmd.PersistentFlags().StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	rootCmd.PersistentFlags().StringVar(&level, "level", "", "Level of injection (either pod or node)")
	rootCmd.PersistentFlags().StringVar(&containerID, "container-id", "", "Targeted container ID")
	_ = cobra.MarkFlagRequired(rootCmd.PersistentFlags(), "level")

	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initMetricsSink)
	cobra.OnInitialize(initConfig)
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

func initLogger() {
	// prepare logger
	zapInstance, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("error while creating logger: %v", err)
		os.Exit(2)
	}

	log = zapInstance.Sugar()
}

func initMetricsSink() {
	var err error

	ms, err = metrics.GetSink(types.SinkDriver(sink), types.SinkAppInjector)
	if err != nil {
		log.Errorw("error while creating metric sink, switching to noop sink", "error", err)

		ms, _ = metrics.GetSink(types.SinkDriverNoop, types.SinkAppInjector)
	}
}

func initConfig() {
	var (
		err        error
		cgroupPath string
		pid        uint32
		ctn        container.Container
	)

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
	cgroupMgr, err := cgroup.NewManager(cgroupPath)
	if err != nil {
		log.Fatalw("error creating cgroup manager", "error", err)
	}

	// create network namespace manager
	netnsMgr, err := netns.NewManager(pid)
	if err != nil {
		log.Fatalw("error creating network namespace manager", "error", err)
	}

	config = injector.Config{
		Log:         log,
		MetricsSink: ms,
		Level:       chaostypes.DisruptionLevel(level),
		Container:   ctn,
		Cgroup:      cgroupMgr,
		Netns:       netnsMgr,
	}
}

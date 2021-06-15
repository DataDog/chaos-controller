// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	logger  *zap.SugaredLogger
	timeout time.Duration

	rootCmd = &cobra.Command{
		Use:   "chaos-handler",
		Short: "A simple process doing nothing but waiting for a SIGUSR1 signal to exit properly",
		Run: func(cmd *cobra.Command, args []string) {
			// wait for SIGUSR1 signal
			logger.Infow("waiting for SIGUSR1", "timeout", timeout.String())
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGUSR1)

			// create timeout timer
			timer := time.NewTimer(timeout)

			// wait for a signal or a timeout
			select {
			case <-sigs:
				logger.Info("SIGUSR1 received, exiting")
				os.Exit(0)
			case <-timer.C:
				logger.Info("timed out, SIGUSR1 was never received, exiting")
				os.Exit(1)
			}
		},
	}
)

func init() {
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", time.Minute, "Time to wait for the signal before the handler exits by itself")
}

func main() {
	var err error

	logger, err = log.NewZapLogger()
	if err != nil {
		fmt.Printf("error initializing logger: %v", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Errorw("error executing command", "error", err)
	}
}

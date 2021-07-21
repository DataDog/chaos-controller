// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	logger "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	log                 *zap.SugaredLogger
	dryRun              bool
	ms                  metrics.Sink
	sink                string
	level               string
	containerIDs        []string
	disruptionName      string
	disruptionNamespace string
	targetName          string
	onInit              bool
	handlerPID          uint32
	configs             []injector.Config
	signals             chan os.Signal
	injectors           []injector.Injector
	readyToInject       bool
	dnsServer           string
	kubeDNS             string
)

func init() {
	rootCmd.AddCommand(networkDisruptionCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(containerFailureCmd)
	rootCmd.AddCommand(cpuPressureCmd)
	rootCmd.AddCommand(diskPressureCmd)
	rootCmd.AddCommand(dnsDisruptionCmd)
	rootCmd.AddCommand(grpcDisruptionCmd)

	// basic args
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	rootCmd.PersistentFlags().StringVar(&level, "level", "", "Level of injection (either pod or node)")
	rootCmd.PersistentFlags().StringSliceVar(&containerIDs, "containers-id", []string{}, "Targeted containers ID")
	rootCmd.PersistentFlags().BoolVar(&onInit, "on-init", false, "Apply the disruption on initialization, requiring a synchronization with the chaos-handler container")
	rootCmd.PersistentFlags().StringVar(&dnsServer, "dns-server", "8.8.8.8", "IP address of the upstream DNS server")
	rootCmd.PersistentFlags().StringVar(&kubeDNS, "kube-dns", "off", "Whether to use kube-dns for DNS resolution (off, internal, all)")

	// log context args
	rootCmd.PersistentFlags().StringVar(&disruptionName, "log-context-disruption-name", "", "Log value: current disruption name")
	rootCmd.PersistentFlags().StringVar(&disruptionNamespace, "log-context-disruption-namespace", "", "Log value: current disruption namespace")
	rootCmd.PersistentFlags().StringVar(&targetName, "log-context-target-name", "", "Log value: current target name")

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
	var err error

	// prepare logger
	log, err = logger.NewZapLogger()
	if err != nil {
		fmt.Printf("error while creating logger: %v", err)
		os.Exit(2)
	}

	log = log.With(
		"disruptionName", disruptionName,
		"disruptionNamespace", disruptionNamespace,
		"targetName", targetName,
	)
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
	cgroupPaths := []string{}
	pids := []uint32{}
	ctns := []container.Container{}
	cgroupMgrs := []cgroup.Manager{}
	netnsMgrs := []netns.Manager{}
	dnsConfig := network.DNSConfig{DNSServer: dnsServer, KubeDNS: kubeDNS}

	// log when dry-run mode is enabled
	if dryRun {
		log.Warn("dry run mode enabled, no disruption will be injected but most of the commands will still be executed to simulate it as much as possible")
	}

	switch level {
	case chaostypes.DisruptionLevelPod:
		// check for container ID flag
		if len(containerIDs) == 0 {
			log.Error("--containers-id flag must be passed when --level=pod")

			return
		}

		for _, containerID := range containerIDs {
			// retrieve container info
			ctn, err := container.New(containerID)
			if err != nil {
				log.Errorw("can't create container object", "error", err)

				return
			}

			log.Infow("injector targeting container", "containerID", containerID, "container name", ctn.Name())

			cgroupPath := ctn.CgroupPath()
			pid := ctn.PID()

			// keep pid for later if this is a chaos handler container
			if onInit && ctn.Name() == "chaos-handler" {
				handlerPID = pid
			}

			ctns = append(ctns, ctn)
			cgroupPaths = append(cgroupPaths, cgroupPath)
			pids = append(pids, pid)
		}
	case chaostypes.DisruptionLevelNode:
		cgroupPaths = []string{""}
		pids = []uint32{1}

		ctns = append(ctns, nil)
	default:
		log.Errorf("unknown level: %s", level)

		return
	}

	cgroups, err := cgroup.ScrapeAllCgroups(log)

	if err != nil {
		log.Errorw("can't find cgroups paths", "error", err)

		return
	}

	// create cgroup managers
	for i, cgroupPath := range cgroupPaths {
		var foundCgroup *cgroup.ContainerCgroup // We want to pass nil for a node level disruption, as we arent targeting a container

		if level == chaostypes.DisruptionLevelPod {
			var ok bool

			rawContainerID := strings.Split(containerIDs[i], "://") // This is "guaranteed" to work because it was already validated when we called container.New(containerID) earlier
			foundCgroup, ok = cgroups[rawContainerID[1]]

			if !ok {
				log.Errorf("could not find cgroup paths for containerID %s", containerIDs[i])

				return
			}
		}

		log.Infow("creating a new cgroup manager", "cgroupPath", cgroupPath)
		cgroupMgr, err := cgroup.NewManager(dryRun, foundCgroup, cgroupPath, log)

		if err != nil {
			log.Errorw("error creating cgroup manager", "error", err)

			return
		}

		cgroupMgrs = append(cgroupMgrs, cgroupMgr)
	}

	// create network namespace manager
	for _, pid := range pids {
		netnsMgr, err := netns.NewManager(pid)
		if err != nil {
			log.Errorw("error creating network namespace manager", "error", err)

			return
		}

		netnsMgrs = append(netnsMgrs, netnsMgr)
	}

	// create kubernetes clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorw("error getting kubernetes client config", "error", err)

		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorw("error creating kubernetes client", "error", err)

		return
	}

	for i, ctn := range ctns {
		config := injector.Config{
			DryRun:      dryRun,
			OnInit:      onInit,
			Log:         log,
			MetricsSink: ms,
			Level:       chaostypes.DisruptionLevel(level),
			Container:   ctn,
			Cgroup:      cgroupMgrs[i],
			Netns:       netnsMgrs[i],
			K8sClient:   clientset,
			DNS:         dnsConfig,
		}

		configs = append(configs, config)
	}

	// mark the disruption as ready to be injected only when all injector configurations are successfully created
	readyToInject = true
}

// initExitSignalsHandler initializes the exit signal handler
func initExitSignalsHandler() {
	// signals needs to be a buffered channel, so that we can receive a signal anytime during the injection process
	// as an unbuffered channel will ignore signals sent before it begins listening. We can keep the buffer size to `1` here
	// as no matter how many SIGINT or SIGTERMs we receive, we want to carry out the same action: clean and die
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
}

// injectAndWait injects the disruption with the configured injector and waits
// for an exit signal to be sent
func injectAndWait(cmd *cobra.Command, args []string) {
	// early exit if an injector configuration failed to be generated during initialization
	if !readyToInject {
		log.Error("an injector could not be configured successfully during initialization, aborting the injection now")

		return
	}

	log.Info("injecting the disruption")

	errOnInject := false

	for _, inj := range injectors {
		// start injection, do not fatal on error so we keep the pod
		// running, allowing the cleanup to happen
		if err := inj.Inject(); err != nil {
			errOnInject = true

			handleMetricError(ms.MetricInjected(false, cmd.Name(), nil))
			log.Errorw("disruption injection failed", "error", err)
		} else {
			handleMetricError(ms.MetricInjected(true, cmd.Name(), nil))
			log.Info("disruption injected, now waiting for an exit signal")
		}
	}

	// create and write readiness probe file if injection succeeded so the pod is marked as ready
	if !errOnInject {
		if err := ioutil.WriteFile(readinessProbeFile, []byte("1"), 0400); err != nil {
			log.Errorw("error writing readiness probe file", "error", err)
		}
	} else {
		log.Error("an injector could not inject the disruption successfully, please look at the logs above for more details")
	}

	// once injected, send a signal to the handler container so it can exit and let other containers go on
	if onInit {
		log.Info("notifying the handler container that injection is now done")

		// ensure a handler container was found
		if handlerPID != 0 {
			// retrieve handler container process to send a signal
			handlerProcess, err := os.FindProcess(int(handlerPID))
			if err != nil {
				log.Errorw("error retrieving handler container process", "error", err)
			}

			// send the SIGUSR1 signal
			if err := handlerProcess.Signal(syscall.SIGUSR1); err != nil {
				log.Errorw("error sending a SIGUSR1 signal to the handler container process", "error", err, "pid", handlerPID)
			}
		} else {
			log.Error("the --on-init flag was provided but no handler container could be found")
		}
	}

	// wait for an exit signal, this is a blocking call
	sig := <-signals

	log.Infow("an exit signal has been received", "signal", sig.String())
}

// cleanAndExit cleans the disruption with the configured injector and exits nicely
func cleanAndExit(cmd *cobra.Command, args []string) {
	log.Info("cleaning the disruption")

	errs := []error{}

	for _, inj := range injectors {
		// start cleanup which is retried up to 3 times using an exponential backoff algorithm
		if err := backoff.RetryNotify(inj.Clean, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), retryNotifyHandler); err != nil {
			handleMetricError(ms.MetricCleaned(false, cmd.Name(), nil))

			errs = append(errs, err)
		}

		handleMetricError(ms.MetricCleaned(true, cmd.Name(), nil))
	}

	// 1 or more injectors failed to clean, log and fatal
	if len(errs) != 0 {
		var combined strings.Builder

		for _, err := range errs {
			combined.WriteString(err.Error())
			combined.WriteString(",")
		}

		log.Fatalw(fmt.Sprintf("disruption cleanup failed on %d injectors (comma separated errors)", len(errs)), "errors", combined.String())
	}

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

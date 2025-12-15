// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/injector"
	logger "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	metricstypes "github.com/DataDog/chaos-controller/o11y/metrics/types"
	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/pflag"
	"github.com/DataDog/chaos-controller/process"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

const (
	readinessProbeFile             = "/tmp/readiness_probe"
	chaosInitContName              = "chaos-handler"
	maxDurationWithoutParentSignal = 5 * time.Second
)

var rootCmd = &cobra.Command{
	Use:               "chaos-injector",
	Short:             "Datadog chaos failures injection application",
	Run:               nil,
	PersistentPostRun: cleanAndExit,
}

var (
	disruptionArgs      chaosapi.DisruptionArgs
	disruptionLevelRaw  string
	parentPID           uint32
	log                 *zap.SugaredLogger
	ms                  metrics.Sink
	rawTargetContainers []string // contains name:id containers
	handlerPID          uint32
	configs             []injector.Config
	signals             chan os.Signal
	injectorCtx         context.Context
	injectors           []injector.Injector
	readyToInject       bool
	clientset           *kubernetes.Clientset
	deadline            time.Time
)

func init() {
	notInjectedBeforeFlag, err := pflag.NewTimeWithFormat(time.RFC3339, &disruptionArgs.NotInjectedBefore)
	if err != nil {
		panic(err)
	}

	deadlineFlag, err := pflag.NewTimeWithFormat(time.RFC3339, &deadline)
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(networkDisruptionCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(podReplacementCmd)
	rootCmd.AddCommand(containerFailureCmd)
	rootCmd.AddCommand(cpuPressureCmd)
	rootCmd.AddCommand(cpuPressureStressCmd)
	rootCmd.AddCommand(diskFailureCmd)
	rootCmd.AddCommand(diskPressureCmd)
	rootCmd.AddCommand(grpcDisruptionCmd)

	// basic args
	rootCmd.PersistentFlags().BoolVar(&disruptionArgs.DryRun, "dry-run", false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.MetricsSink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	rootCmd.PersistentFlags().StringVar(&disruptionLevelRaw, "level", "", "Level of injection (either pod or node)")
	rootCmd.PersistentFlags().StringSliceVar(&rawTargetContainers, "target-containers", []string{}, "Targeted containers")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.TargetPodIP, "target-pod-ip", "", "Pod IP of targeted pod")
	rootCmd.PersistentFlags().BoolVar(&disruptionArgs.OnInit, "on-init", false, "Apply the disruption on initialization, requiring a synchronization with the chaos-handler container")
	rootCmd.PersistentFlags().DurationVar(&disruptionArgs.PulseInitialDelay, "pulse-initial-delay", time.Duration(0), "Duration to wait after injector starts before beginning the activeDuration")
	rootCmd.PersistentFlags().DurationVar(&disruptionArgs.PulseActiveDuration, "pulse-active-duration", time.Duration(0), "Duration of the disruption being active in a pulsing disruption (empty if the disruption is not pulsing)")
	rootCmd.PersistentFlags().DurationVar(&disruptionArgs.PulseDormantDuration, "pulse-dormant-duration", time.Duration(0), "Duration of the disruption being dormant in a pulsing disruption (empty if the disruption is not pulsing)")
	rootCmd.PersistentFlags().Var(notInjectedBeforeFlag, "not-injected-before", "")
	rootCmd.PersistentFlags().Var(deadlineFlag, string(injector.DeadlineFlag), "RFC3339 time at which the disruption must be over by")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.ChaosNamespace, "chaos-namespace", "chaos-engineering", "Namespace that contains this chaos pod")
	rootCmd.PersistentFlags().Uint32Var(&parentPID, string(injector.ParentPIDFlag), 0, "Parent process PID")

	// log context args
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.DisruptionName, "log-context-disruption-name", "", "Log value: current disruption name")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.DisruptionNamespace, "log-context-disruption-namespace", "", "Log value: current disruption namespace")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.TargetName, "log-context-target-name", "", "Log value: current target name")
	rootCmd.PersistentFlags().StringVar(&disruptionArgs.TargetNodeName, "log-context-target-node-name", "", "Log value: node hosting the current target pod")

	_ = cobra.MarkFlagRequired(rootCmd.PersistentFlags(), "level")
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(initMetricsSink)
	cobra.OnInitialize(initExitSignalsHandler)
	cobra.OnInitialize(initConfig)
}

func main() {
	// handle metrics sink client close on exit
	defer func() {
		log.Infow("closing metrics sink client before exiting", tags.SinkKey, ms.GetSinkName())

		if err := ms.Close(); err != nil {
			log.Errorw("error closing metrics sink client", tags.ErrorKey, err, tags.SinkKey, ms.GetSinkName())
		}
	}()

	// execute command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1) //nolint:gocritic
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
		tags.DisruptionNameKey, disruptionArgs.DisruptionName,
		tags.DisruptionNamespaceKey, disruptionArgs.DisruptionNamespace,
		tags.TargetNameKey, disruptionArgs.TargetName,
		tags.TargetNodeNameKey, disruptionArgs.TargetNodeName,
	)

	if parentPID != 0 {
		log = log.With(tags.ParentPidKey, parentPID)
	}
}

// initMetricsSink initializes a metrics sink depending on the given flag
func initMetricsSink() {
	var err error

	ms, err = metrics.GetSink(log, metricstypes.SinkDriver(disruptionArgs.MetricsSink), metricstypes.SinkAppInjector)
	if err != nil {
		log.Errorw("error while creating metric sink, switching to noop sink", tags.ErrorKey, err, tags.DriverKey, disruptionArgs.MetricsSink)

		if ms, err = metrics.GetSink(log, metricstypes.SinkDriverNoop, metricstypes.SinkAppInjector); err != nil {
			log.Fatalw("error while creating noop metric sink", tags.ErrorKey, err)
		}
	}
}

func initManagers(pid uint32) (netns.Manager, cgroup.Manager, error) {
	netnsMgr, err := netns.NewManager(log, pid)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating network namespace manager: %w", err)
	}

	// retrieve cgroup mount path set in env or fallback to the default value
	cgroupMount := ""
	if mount, exists := os.LookupEnv(env.InjectorMountCgroup); exists {
		cgroupMount = mount
	} else {
		// process ID 1 is usually the init process primarily responsible for starting and shutting down the system
		// originally, process ID 1 was not specifically reserved for init by any technical measures
		// it simply had this ID as a natural consequence of being the first process invoked by the kernel
		cgroupMount = "/proc/1/root/sys/fs/cgroup"
	}

	// create cgroups manager
	cgroupMgr, err := cgroup.NewManager(disruptionArgs.DryRun, pid, cgroupMount, log)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating cgroup manager: %w", err)
	}

	return netnsMgr, cgroupMgr, nil
}

// initConfig initializes the injector config and main components from the given flags
func initConfig() {
	pids := []uint32{}
	ctns := []container.Container{}

	// log when dry-run mode is enabled
	if disruptionArgs.DryRun {
		log.Warn("dry run mode enabled, no disruption will be injected but most of the commands will still be executed to simulate it as much as possible")
	}

	disruptionArgs.TargetContainers = map[string]string{}

	// parse target-containers
	for _, cnt := range rawTargetContainers {
		splittedCntInfo := strings.SplitN(cnt, ";", 2)

		if len(splittedCntInfo) == 2 {
			disruptionArgs.TargetContainers[splittedCntInfo[0]] = splittedCntInfo[1]
		}
	}

	// assign to the pointer to level the new value to persist it after this method
	disruptionArgs.Level = chaostypes.DisruptionLevel(disruptionLevelRaw)

	// check if we're running pod-replacement command which doesn't need containers
	isPodReplacement := false

	for _, arg := range os.Args {
		if arg == chaostypes.DisruptionKindPodReplacement {
			isPodReplacement = true
			break
		}
	}

	switch disruptionArgs.Level {
	case chaostypes.DisruptionLevelPod:
		// check for container ID flag
		if len(disruptionArgs.TargetContainers) == 0 {
			log.Fatal("--target-containers flag must be passed when --level=pod")

			return
		}

		if !isPodReplacement {
			// Pod replacement operates at the pod level and doesn't need container information
			for containerName, containerID := range disruptionArgs.TargetContainers {
				// retrieve container info
				ctn, err := container.New(containerID, containerName)
				if err != nil {
					log.Fatalw("can't create container object", tags.ErrorKey, err)

					return
				}

				log.Infow("injector targeting container", tags.ContainerIDKey, containerID, tags.ContainerNameKey, containerName)

				pid := ctn.PID()

				// keep pid for later if this is a chaos handler container
				if disruptionArgs.OnInit && ctn.Name() == chaosInitContName {
					handlerPID = pid
				}

				ctns = append(ctns, ctn)
				pids = append(pids, pid)
			}
		} else {
			// check for pod IP flag
			if disruptionArgs.TargetPodIP == "" {
				log.Fatal("--target-pod-ip flag must be passed when --level=pod")

				return
			}

			pids = []uint32{1}
			ctns = []container.Container{nil}
		}
	case chaostypes.DisruptionLevelNode:
		pids = []uint32{1}
		ctns = []container.Container{nil}
	default:
		log.Fatal("unknown level: %s", disruptionArgs.Level)

		return
	}

	// create kubernetes clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalw("error getting kubernetes client config", tags.ErrorKey, err)

		return
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalw("error creating kubernetes client", tags.ErrorKey, err)

		return
	}

	// create injector configs
	for i, pid := range pids {
		// create network namespace manager
		netnsMgr, cgroupMgr, err := initManagers(pid)
		if err != nil {
			log.Fatalw("unable to create ns and cgroup managers for pid", tags.ErrorKey, err, tags.PidKey, pid)

			return
		}

		// generate injector config
		injectorConfig := injector.Config{
			MetricsSink:        ms,
			TargetContainer:    ctns[i],
			DisruptionDeadline: deadline,
			Cgroup:             cgroupMgr,
			Netns:              netnsMgr,
			K8sClient:          clientset,
			Disruption:         disruptionArgs,
			InjectorCtx:        injectorCtx,
		}
		injectorConfig.Log = log.With(tags.TargetLevelKey, disruptionArgs.Level, tags.TargetNameKey, injectorConfig.TargetName()) // targetName is already taken in the initLogger

		configs = append(configs, injectorConfig)
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

	var injectorCancelFunc context.CancelFunc
	injectorCtx, injectorCancelFunc = context.WithCancel(context.Background())

	// In case the inject phase manages to take more than the disruption duration + activeDeadlineSeconds
	// we can end up receiving a sigkill during inject, leaving us stuck on removal. To prevent this, we pass a context
	// to the injector config that can be cancelled if we receive any early exit signals. Injectors with possible length injects should
	// take care to check if this context has been cancelled. After cancelling that context, we return the exit signal to the signals channel
	// so that the appropriate handlers that trigger cleanup can begin
	go func() {
		sig := <-signals

		injectorCancelFunc()

		go func() { signals <- sig }()
	}()
}

// initPodWatch initializes the target pod watcher
func initPodWatch(resourceVersion string) (<-chan watch.Event, error) {
	podWatcher, err := clientset.CoreV1().Pods(disruptionArgs.DisruptionNamespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector:       "metadata.name=" + disruptionArgs.TargetName,
		ResourceVersion:     resourceVersion,
		AllowWatchBookmarks: true,
	})
	if err != nil {
		return nil, err
	}

	if podWatcher == nil {
		return nil, fmt.Errorf("unable to watch pod %s in namespace %s", disruptionArgs.TargetName, disruptionArgs.DisruptionNamespace)
	}

	return podWatcher.ResultChan(), err
}

// inject all the disruptions using the list of injectors
// returns true if injection succeeded, false otherwise
func inject(kind string, sendToMetrics bool, reinjection bool) bool {
	errOnInject := false

	for _, inj := range injectors {
		// start injection, do not fatal on error so we keep the pod
		// running, allowing the cleanup to happen
		if err := inj.Inject(); err != nil {
			errOnInject = true

			if sendToMetrics {
				if reinjection {
					handleMetricError(ms.MetricReinjected(false, kind, nil))
				} else {
					handleMetricError(ms.MetricInjected(false, kind, nil))
				}
			}

			log.Errorw("disruption injection failed", tags.ErrorKey, err, tags.TargetNameKey, inj.TargetName())
		} else {
			if sendToMetrics {
				if reinjection {
					handleMetricError(ms.MetricReinjected(true, kind, nil))
				} else {
					handleMetricError(ms.MetricInjected(true, kind, nil))
				}
			}

			if reinjection {
				log.Infof("disruption %s is reinjected", inj.GetDisruptionKind())
			} else {
				log.Infof("disruption %s injected", inj.GetDisruptionKind())
			}
		}
	}

	if errOnInject {
		log.Error("an injector could not inject the disruption successfully, please look at the logs above for more details")
	}

	return !errOnInject
}

// reinject reinitialize conf, clean and inject all the disruptions
func reinject(cmdName string) error {
	var err error

	// Clean all injections to reinject on an empty slate
	if ok := clean(cmdName, true, true); !ok {
		log.Errorw("couldn't clean targets before reinjection. Reinjecting anyway")
	}

	// We rebuild and update the configuration
	for i, conf := range configs {
		if conf.TargetContainer == nil {
			continue
		}

		newContainerID, found := disruptionArgs.TargetContainers[conf.TargetContainer.Name()]
		if !found {
			return fmt.Errorf("container %s is not found (old containerID is %s)", conf.TargetContainer.Name(), conf.TargetContainer.ID())
		}

		if conf.TargetContainer, err = container.New(newContainerID, conf.TargetContainer.Name()); err != nil {
			return fmt.Errorf("unable to create a container from containerID %s: %w", newContainerID, err)
		}

		if conf.Netns, conf.Cgroup, err = initManagers(conf.TargetContainer.PID()); err != nil {
			return fmt.Errorf("unable to reinitialize netns manager and cgroup manager for containerID %s (PID: %d): %w", conf.TargetContainer.ID(), conf.TargetContainer.PID(), err)
		}

		// Update disruption args with latest updates from container for every config
		conf.Disruption = disruptionArgs

		injectors[i].UpdateConfig(conf)
	}

	// Reinject target
	if ok := inject(cmdName, true, true); !ok {
		return fmt.Errorf("couldn't reinject target")
	}

	return nil
}

// clean will remove or undo all the disruptions using the list of injectors
// returns true if cleanup succeeded, false otherwise
func clean(kind string, sendToMetrics bool, reinjectionClean bool) bool {
	errOnClean := false

	for _, inj := range injectors {
		// start cleanup which is retried up to 3 times using an exponential backoff algorithm
		if err := backoff.RetryNotify(inj.Clean, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), retryNotifyHandler); err != nil {
			errOnClean = true

			if sendToMetrics {
				if reinjectionClean {
					handleMetricError(ms.MetricCleanedForReinjection(false, kind, nil))
				} else {
					handleMetricError(ms.MetricCleaned(false, kind, nil))
				}
			}

			log.Errorw("disruption cleaning failed", tags.ErrorKey, err)
		} else {
			log.Infof("disruption %s cleaned", inj.GetDisruptionKind())
		}

		if sendToMetrics {
			if reinjectionClean {
				handleMetricError(ms.MetricCleanedForReinjection(true, kind, nil))
			} else {
				handleMetricError(ms.MetricCleaned(true, kind, nil))
			}
		}
	}

	if errOnClean {
		log.Errorw("an injector could not clean the disruption successfully, please look at the logs above for more details")
	}

	return !errOnClean
}

// pulse pulse disruptions (injection and cleaning)
// nolint: unparam,staticcheck
func pulse(isInjected bool, cmdName string) (bool, time.Time, error) {
	// if the disruption is injected, we clean it
	if isInjected {
		log.Debugw("pulse: attempt at cleaning the disruption for the dormant duration", tags.DurationKey, disruptionArgs.PulseDormantDuration)

		if ok := clean(cmdName, true, true); !ok {
			return true, time.Now().Add(disruptionArgs.PulseDormantDuration), fmt.Errorf("error on pulsing disruption mechanism when attempting to clean the disruption for the dormant pulse")
		}

		return false, time.Now().Add(disruptionArgs.PulseDormantDuration), nil
	}

	// if the disruption is not injected, we inject it
	log.Debugw("pulse: attempt at injecting the disruption for the active duration", tags.DurationKey, disruptionArgs.PulseActiveDuration)

	if ok := inject(cmdName, true, true); !ok {
		return false, time.Now().Add(disruptionArgs.PulseActiveDuration), fmt.Errorf("error on pulsing disruption mechanism when attempting to inject the disruption for the active pulse")
	}

	return true, time.Now().Add(disruptionArgs.PulseActiveDuration), nil
}

// injectAndWait injects the disruption with the configured injector and waits
// for an exit signal to be sent
func injectAndWait(cmd *cobra.Command, args []string) {
	// early exit if an injector configuration failed to be generated during initialization
	if !readyToInject || len(injectors) == 0 {
		log.Error("an injector could not be configured successfully during initialization, aborting the injection now")

		return
	}

	if !disruptionArgs.NotInjectedBefore.IsZero() {
		log.Infow("waiting for synchronized start to begin", tags.TimeUntilNotInjectedBeforeKey, time.Until(disruptionArgs.NotInjectedBefore).String())
		select {
		case sig := <-signals:
			log.Infow("an exit signal has been received", tags.SignalKey, sig.String())

			return
		case <-time.After(time.Until(disruptionArgs.NotInjectedBefore)):
			break
		}
	}

	if disruptionArgs.PulseInitialDelay > 0 {
		log.Infow("waiting for initialDelay to pass", tags.InitialDelayKey, disruptionArgs.PulseInitialDelay)
		select {
		case <-time.After(disruptionArgs.PulseInitialDelay):
			break
		case sig := <-signals:
			log.Infow("an exit signal has been received", tags.SignalKey, sig.String())

			return
		}
	}

	processManager := process.NewManager(disruptionArgs.DryRun)

	log.Infow("injecting the disruption", tags.KindKey, cmd.Name())

	injectSuccess := inject(cmd.Name(), true, false)

	// create and write readiness probe file if injection succeeded so the pod is marked as ready
	if injectSuccess {
		log.Infof("disruption(s) injected, now waiting for an exit signal")

		if err := os.WriteFile(readinessProbeFile, []byte("1"), 0o400); err != nil {
			log.Errorw("error writing readiness probe file", tags.ErrorKey, err)
		}
	}

	// once injected, send a signal to the handler container so it can exit and let other containers go on
	if disruptionArgs.OnInit {
		log.Info("notifying the handler container that injection is now done")

		// ensure a handler container was found
		if handlerPID != 0 {
			// retrieve handler container process to send a signal
			handlerProcess, err := processManager.Find(int(handlerPID))
			if err != nil {
				log.Errorw("error retrieving handler container process", tags.ErrorKey, err)
			} else if err := handlerProcess.Signal(syscall.SIGUSR1); err != nil { // send the SIGUSR1 signal
				log.Errorw("error sending a SIGUSR1 signal to the handler container process", tags.ErrorKey, err, tags.PidKey, handlerPID)
			}
		} else {
			log.Error("the --on-init flag was provided but no handler container could be found")
		}
	}

	switch {
	case !injectSuccess:
		break
	// those disruptions should not watch target to re-inject on container restart
	case v1beta1.DisruptionIsNotReinjectable((chaostypes.DisruptionKindName)(cmd.Name())):
	case disruptionArgs.Level == chaostypes.DisruptionLevelNode:
		if disruptionArgs.PulseActiveDuration > 0 && disruptionArgs.PulseDormantDuration > 0 {
			var (
				err error
			)

			// at that point, the disruption is injected. We start the pulsing loop with the active duration
			isInjected := true
			pulseDeadline := time.Now().Add(disruptionArgs.PulseActiveDuration)

			// using a label for the loop to be able to break out of it
		pulsingLoop:
			for { // This loop will wait, clean, wait, inject until a signal is received
				select { // Quit on signal reception or sleep and injects / cleans the disruptions
				case sig := <-signals:
					log.Infow("an exit signal has been received", tags.SignalKey, sig.String())

					return
				case <-time.After(getDuration(deadline)):
					log.Infow("duration has expired")

					return
				case <-time.After(getDuration(pulseDeadline)):
					isInjected, pulseDeadline, err = pulse(isInjected, cmd.Name())
					if err != nil {
						log.Errorw("an error occurred when calling pulse", tags.ErrorKey, err)

						// break of PulsingLoop only
						break pulsingLoop
					}

					log.Infow("pulse action has been performed", tags.PulseNextActionTimestampKey, pulseDeadline)
				}
			}
		}
	default:
		if parentPID != 0 {
			log.Info("Process is a child process, SKIPPING watch target and reinject (parent process is responsible for that)...")

			go func() {
				log.Infow("starting to watch parent ID process existence every 1 second")

				for range time.Tick(1 * time.Second) {
					if _, err := processManager.Exists(int(parentPID)); err != nil {
						log.Errorw("an error occurred when looking at parent process, it may no longer exists, exiting...", tags.ErrorKey, err)
						signals <- os.Interrupt
					}
				}
			}()

			break
		}

		if disruptionArgs.OnInit {
			log.Debugw("the init container will not get restarted on container restart")
		}

		// we watch for targeted pod containers restart to reinject
		if err := backoff.RetryNotify(
			func() error {
				return watchTargetAndReinject(deadline, cmd.Name(), disruptionArgs.PulseActiveDuration, disruptionArgs.PulseDormantDuration)
			}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5), func(err error, delay time.Duration) {
				log.Warnln("couldn't watch targeted pod, retrying", tags.ErrorKey, err, tags.RetryingKey, delay)
			}); err != nil {
			log.Errorln("unable to watch targeted pod after several retry, ending...", tags.ErrorKey, err)
		} else {
			return
		}
	}

	// When it's a child process, we want to quit early in case our parent dies
	// Below mechanism aims to receive signals sent from parent to have a sliding expiration mechanism to kill ourselves
	// When no signal are received in allocated duration, child process dies
	var (
		parentSignal        chan os.Signal
		noParentSignalTimer *time.Timer
	)

	if parentPID != 0 {
		parentSignal = make(chan os.Signal, 1)
		signal.Notify(parentSignal, syscall.SIGCONT)

		noParentSignalTimer = time.NewTimer(maxDurationWithoutParentSignal)
	} else {
		noParentSignalTimer = time.NewTimer(getDuration(deadline) + 1*time.Minute)
	}

	log.Info("waiting for system signals...")

	for {
		select {
		case <-parentSignal: // when a channel is nil it waits forever (case when NOT a child process)
			if !noParentSignalTimer.Stop() {
				<-noParentSignalTimer.C
			}

			noParentSignalTimer.Reset(maxDurationWithoutParentSignal)
		case <-noParentSignalTimer.C:
			log.Warnf("parent did not sent any signal in the last %v seconds, exiting", maxDurationWithoutParentSignal)
			return
		case sig := <-signals:
			log.Infow("an exit signal has been received, exiting", tags.SignalKey, sig.String())
			return
		case <-time.After(getDuration(deadline)):
			log.Info("disruption duration has expired, exiting")
			return
		}
	}
}

// watchTargetAndReinject handle reinjection of the disruption on container restart
func watchTargetAndReinject(deadline time.Time, commandName string, pulseActiveDuration time.Duration, pulseDormantDuration time.Duration) error {
	// we keep track of resource version in case of errors during watch to pick up where we were before the error
	resourceVersion, err := getPodResourceVersion()
	if err != nil {
		return err
	}

	// at that point, the disruption is injected. We start the pulsing loop with the active duration
	pulseIndexIsInjected := true

	var pulseDeadline time.Time

	// set sleepDuration to after deadline duration to never go into the pulsing condition
	if pulseActiveDuration > 0 && pulseDormantDuration > 0 {
		pulseDeadline = time.Now().Add(pulseActiveDuration)
	} else {
		pulseDeadline = time.Now().Add(getDuration(deadline) + time.Hour)
	}

	var channel <-chan watch.Event

	for {
		if channel == nil {
			if channel, err = initPodWatch(resourceVersion); err != nil {
				return err
			}
		}

		select {
		case sig := <-signals:
			log.Infow("an exit signal has been received", tags.SignalKey, sig.String())
			// We've already consumed the signal from the channel, so our caller won't find it when checking after we return.
			// Thus we need to put the signal back into the channel. We do it in a gothread in case we are blocked when writing to the channel
			go func() { signals <- sig }()

			return nil
		case <-time.After(getDuration(deadline)):
			log.Infow("duration has expired")

			return nil
		// shouldn't go there if it's not a pulsing disruption
		case <-time.After(getDuration(pulseDeadline)):
			if pulseActiveDuration == 0 || pulseDormantDuration == 0 {
				break
			}

			pulseIndexIsInjected, pulseDeadline, err = pulse(pulseIndexIsInjected, commandName)
			if err != nil {
				return err
			}

			log.Infow("pulse action has been performed", tags.PulseNextActionTimestampKey, pulseDeadline)
		case event, ok := <-channel: // We have changes in the pod watched
			log.Debugw("received event during target watch", tags.TypeKey, event.Type)

			if !ok {
				channel = nil

				continue
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				log.Debugw("received event was not a pod", tags.EventKey, event, tags.EventObjectKey, event.Object)

				status, sok := event.Object.(*metav1.Status)
				if sok {
					if status.Code == 410 {
						// Status Code 410 indicates our resource version has expired
						// Get the newest resource version and re-create the channel
						updResourceVersion, err := getPodResourceVersion()
						if err != nil {
							return fmt.Errorf("could not get latest resource version: %w", err)
						}

						if updResourceVersion == resourceVersion {
							// An unset resource version is an implicit "latest" request.
							resourceVersion = ""
						} else {
							resourceVersion = updResourceVersion
						}

						channel = nil

						log.Debugw("restarting pod watching channel with newest resource version", tags.ResourceVersionKey, resourceVersion)

						continue
					}
				}

				return fmt.Errorf("watched object received from event is not a pod")
			}

			if event.Type == watch.Bookmark {
				resourceVersion = pod.ResourceVersion
				channel = nil

				log.Debugw("received bookmark event, new resource version found", tags.ResourceVersionKey, pod.ResourceVersion)
			}

			if event.Type != watch.Modified {
				continue
			}

			notReady := false

			// We wait for the pod to have all containers ready.
			for _, status := range pod.Status.ContainerStatuses {
				// we don't control the state of the init container
				if disruptionArgs.TargetContainers[status.Name] == "" || (disruptionArgs.OnInit && status.Name == chaosInitContName) {
					continue
				}

				if status.Started == nil || (status.Started != nil && !*status.Started) {
					notReady = true

					break
				}
			}

			if notReady {
				continue
			}

			hasChanged, err := updateTargetContainersAndDetectChange(*pod)
			if err != nil {
				return fmt.Errorf("an error occurred to detect change: %w", err)
			} else if !hasChanged {
				continue
			}

			// if a container is in crashloop, we might fail due to not found stuff
			// we might want to retry to retrieve pod AND changes later on instead of stopping abruptly
			if err := reinject(commandName); err != nil {
				return fmt.Errorf("an error occurred during reinjection: %w", err)
			}
		}
	}
}

// cleanAndExit cleans the disruption with the configured injector and exits nicely
func cleanAndExit(cmd *cobra.Command, args []string) {
	// 1 or more injectors failed to clean, we exit
	if ok := clean(cmd.Name(), true, false); !ok {
		os.Exit(1)
	}

	// When the command is considered a child command, we don't expect to cleanup pod level finalizer
	// it's supposedly done by the main process that spins us up
	if parentPID != 0 {
		log.Info("child process cleaned, skipping finalizer removal and exiting")
		return
	}

	if err := backoff.RetryNotify(cleanFinalizer, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), retryNotifyHandler); err != nil {
		log.Errorw("couldn't safely remove this pod's finalizer", tags.ErrorKey, err)
	}

	log.Info("disruption(s) cleaned, now exiting")
}

func cleanFinalizer() error {
	if len(configs) == 0 {
		err := fmt.Errorf("no configuration available for this disruption")

		log.Warnw("couldn't GET this pod in order to remove its finalizer", tags.PodNameKey, os.Getenv(env.InjectorPodName), tags.ErrorKey, err)

		return err
	}

	pod, err := configs[0].K8sClient.CoreV1().Pods(disruptionArgs.ChaosNamespace).Get(context.Background(), os.Getenv(env.InjectorPodName), metav1.GetOptions{})
	if err != nil {
		log.Warnw("couldn't GET this pod in order to remove its finalizer", tags.PodNameKey, os.Getenv(env.InjectorPodName), tags.ErrorKey, err)
		return err
	}

	controllerutil.RemoveFinalizer(pod, chaostypes.ChaosPodFinalizer)

	_, err = configs[0].K8sClient.CoreV1().Pods(disruptionArgs.ChaosNamespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	if err != nil {
		log.Warnw("couldn't remove this pod's finalizer", tags.PodNameKey, os.Getenv(env.InjectorPodName), tags.ErrorKey, err)
		return err
	}

	return nil
}

// handleMetricError logs the given error if not nil
func handleMetricError(err error) {
	if err != nil {
		log.Errorw("error sending metric", tags.SinkKey, ms.GetSinkName(), tags.ErrorKey, err)
	}
}

// retryNotifyHandler is called when the cleanup fails
// it logs the error and the time to wait before the next retry
func retryNotifyHandler(err error, delay time.Duration) {
	if v1beta1.IsUpdateConflictError(err) {
		log.Infow("a retryable error occurred during disruption cleanup", tags.ErrorKey, err)
	} else {
		log.Errorw("disruption cleanup failed", tags.ErrorKey, err)
	}

	log.Infof("retrying cleanup in %s", delay.String())
}

// getDuration returns the time between time.Now() and when the disruption is due to expire
// This gives the chaos pod plenty of time to clean up before it hits activeDeadlineSeconds and becomes Failed
func getDuration(deadline time.Time) time.Duration {
	return time.Until(deadline)
}

// getPodResourceVersion get the resource version of the targeted pod
func getPodResourceVersion() (string, error) {
	target, err := clientset.CoreV1().Pods(disruptionArgs.DisruptionNamespace).Get(context.Background(), disruptionArgs.TargetName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return target.ResourceVersion, nil
}

// updateTargetContainersAndDetectChange get all target container infos to determine if one container has changed ID
// if it has changed ID, the container just restarted and need to be reinjected
func updateTargetContainersAndDetectChange(pod v1.Pod) (bool, error) {
	var err error
	// transform map of targetContainer info (name, id) to only an array of names
	targetContainerNames := []string{}
	for name := range disruptionArgs.TargetContainers {
		targetContainerNames = append(targetContainerNames, name)
	}

	log.Debugw("disruption args containers BEFORE updating them", tags.TargetContainersKey, disruptionArgs.TargetContainers)

	if disruptionArgs.TargetContainers, err = v1beta1.TargetedContainers(pod, targetContainerNames); err != nil {
		return false, fmt.Errorf("unable to get targeted containers info. Waiting for next change to reinject: %w", err)
	}

	log.Debugw("disruption args containers AFTER updating them", tags.TargetContainersKey, disruptionArgs.TargetContainers)

	for _, conf := range configs {
		newContainerID, newContainerExists := disruptionArgs.TargetContainers[conf.TargetContainer.Name()]

		parsedNewContainerID, _, err := container.ParseContainerID(newContainerID)
		if err != nil {
			return false, fmt.Errorf("unable to parse provided container ID %s: %w", newContainerID, err)
		}

		if !newContainerExists || parsedNewContainerID != conf.TargetContainer.ID() {
			log.Infow("change detected for container",
				tags.ContainerNameKey, conf.TargetContainer.Name(),
				tags.NewContainerExistsKey, newContainerExists,
				tags.NewContainerIDKey, newContainerID,
				tags.OldContainerIDKey, conf.TargetContainer.ID(),
			)

			return true, nil
		}
	}

	return false, nil
}

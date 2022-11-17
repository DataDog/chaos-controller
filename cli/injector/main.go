// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/injector"
	logger "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"
	"github.com/cenkalti/backoff"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	readinessProbeFile = "/tmp/readiness_probe"
	chaosInitContName  = "chaos-handler"
)

var rootCmd = &cobra.Command{
	Use:               "chaos-injector",
	Short:             "Datadog chaos failures injection application",
	Run:               nil,
	PersistentPostRun: cleanAndExit,
}

var (
	log                  *zap.SugaredLogger
	dryRun               bool
	ms                   metrics.Sink
	sink                 string
	level                string
	rawTargetContainers  []string // contains name:id containers
	targetContainers     map[string]string
	targetPodIP          string
	disruptionName       string
	disruptionNamespace  string
	chaosNamespace       string
	targetName           string
	targetNodeName       string
	onInit               bool
	pulseActiveDuration  time.Duration
	pulseDormantDuration time.Duration
	deadlineRaw          string
	handlerPID           uint32
	configs              []injector.Config
	signals              chan os.Signal
	injectors            []injector.Injector
	readyToInject        bool
	dnsServer            string
	kubeDNS              string
	clientset            *kubernetes.Clientset
)

func init() {
	rootCmd.AddCommand(networkDisruptionCmd)
	rootCmd.AddCommand(nodeFailureCmd)
	rootCmd.AddCommand(containerFailureCmd)
	rootCmd.AddCommand(cpuPressureCmd)
	rootCmd.AddCommand(diskFailureCmd)
	rootCmd.AddCommand(diskPressureCmd)
	rootCmd.AddCommand(dnsDisruptionCmd)
	rootCmd.AddCommand(grpcDisruptionCmd)

	// basic args
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Enable dry-run mode")
	rootCmd.PersistentFlags().StringVar(&sink, "metrics-sink", "noop", "Metrics sink (datadog, or noop)")
	rootCmd.PersistentFlags().StringVar(&level, "level", "", "Level of injection (either pod or node)")
	rootCmd.PersistentFlags().StringSliceVar(&rawTargetContainers, "target-containers", []string{}, "Targeted containers")
	rootCmd.PersistentFlags().StringVar(&targetPodIP, "target-pod-ip", "", "Pod IP of targeted pod")
	rootCmd.PersistentFlags().BoolVar(&onInit, "on-init", false, "Apply the disruption on initialization, requiring a synchronization with the chaos-handler container")
	rootCmd.PersistentFlags().DurationVar(&pulseActiveDuration, "pulse-active-duration", time.Duration(0), "Duration of the disruption being active in a pulsing disruption (empty if the disruption is not pulsing)")
	rootCmd.PersistentFlags().DurationVar(&pulseDormantDuration, "pulse-dormant-duration", time.Duration(0), "Duration of the disruption being dormant in a pulsing disruption (empty if the disruption is not pulsing)")
	rootCmd.PersistentFlags().StringVar(&deadlineRaw, "deadline", "", "Timestamp at which the disruption must be over by")
	rootCmd.PersistentFlags().StringVar(&dnsServer, "dns-server", "8.8.8.8", "IP address of the upstream DNS server")
	rootCmd.PersistentFlags().StringVar(&kubeDNS, "kube-dns", "off", "Whether to use kube-dns for DNS resolution (off, internal, all)")
	rootCmd.PersistentFlags().StringVar(&chaosNamespace, "chaos-namespace", "chaos-engineering", "Namespace that contains this chaos pod")

	// log context args
	rootCmd.PersistentFlags().StringVar(&disruptionName, "log-context-disruption-name", "", "Log value: current disruption name")
	rootCmd.PersistentFlags().StringVar(&disruptionNamespace, "log-context-disruption-namespace", "", "Log value: current disruption namespace")
	rootCmd.PersistentFlags().StringVar(&targetName, "log-context-target-name", "", "Log value: current target name")
	rootCmd.PersistentFlags().StringVar(&targetNodeName, "log-context-target-node-name", "", "Log value: node hosting the current target pod")

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
		"disruptionName", disruptionName,
		"disruptionNamespace", disruptionNamespace,
		"targetName", targetName,
		"targetNodeName", targetNodeName,
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

func initManagers(pid uint32) (netns.Manager, cgroup.Manager, error) {
	netnsMgr, err := netns.NewManager(log, pid)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating network namespace manager: %s", err.Error())
	}

	// create cgroups manager
	cgroupMgr, err := cgroup.NewManager(dryRun, pid, log)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating cgroup manager: %s", err.Error())
	}

	return netnsMgr, cgroupMgr, nil
}

// initConfig initializes the injector config and main components from the given flags
func initConfig() {
	pids := []uint32{}
	ctns := []container.Container{}
	dnsConfig := network.DNSConfig{DNSServer: dnsServer, KubeDNS: kubeDNS}

	// log when dry-run mode is enabled
	if dryRun {
		log.Warn("dry run mode enabled, no disruption will be injected but most of the commands will still be executed to simulate it as much as possible")
	}

	targetContainers = map[string]string{}

	// parse target-containers
	for _, cnt := range rawTargetContainers {
		splittedCntInfo := strings.SplitN(cnt, ";", 2)

		if len(splittedCntInfo) == 2 {
			targetContainers[splittedCntInfo[0]] = splittedCntInfo[1]
		}
	}

	switch level {
	case chaostypes.DisruptionLevelPod:
		// check for container ID flag
		if len(targetContainers) == 0 {
			log.Error("--target-containers flag must be passed when --level=pod")

			return
		}

		for _, containerID := range targetContainers {
			// retrieve container info
			ctn, err := container.New(containerID)
			if err != nil {
				log.Errorw("can't create container object", "error", err)

				return
			}

			log.Infow("injector targeting container", "containerID", containerID, "container name", ctn.Name())

			pid := ctn.PID()

			// keep pid for later if this is a chaos handler container
			if onInit && ctn.Name() == chaosInitContName {
				handlerPID = pid
			}

			ctns = append(ctns, ctn)
			pids = append(pids, pid)
		}

		// check for pod IP flag
		if targetPodIP == "" {
			log.Error("--target-pod-ip flag must be passed when --level=pod")

			return
		}
	case chaostypes.DisruptionLevelNode:
		pids = []uint32{1}
		ctns = []container.Container{nil}
	default:
		log.Errorf("unknown level: %s", level)

		return
	}

	// create kubernetes clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Errorw("error getting kubernetes client config", "error", err)

		return
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorw("error creating kubernetes client", "error", err)

		return
	}

	// create injector configs
	for i, pid := range pids {
		// create network namespace manager
		netnsMgr, cgroupMgr, err := initManagers(pid)
		if err != nil {
			log.Errorw(err.Error(), "pid", pid)

			return
		}

		// generate injector config
		config := injector.Config{
			DryRun:          dryRun,
			OnInit:          onInit,
			Log:             log,
			MetricsSink:     ms,
			Level:           chaostypes.DisruptionLevel(level),
			TargetContainer: ctns[i],
			TargetPodIP:     targetPodIP,
			TargetNodeName:  targetNodeName,
			Cgroup:          cgroupMgr,
			Netns:           netnsMgr,
			K8sClient:       clientset,
			DNS:             dnsConfig,
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

// initPodWatch initializes the target pod watcher
func initPodWatch(resourceVersion string) (<-chan watch.Event, error) {
	podWatcher, err := clientset.CoreV1().Pods(disruptionNamespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector:       "metadata.name=" + targetName,
		ResourceVersion:     resourceVersion,
		AllowWatchBookmarks: true,
	})

	return podWatcher.ResultChan(), err
}

// inject inject all the disruptions using the list of injectors
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

			log.Errorw("disruption injection failed", "error", err)
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
		log.Errorf("an injector could not inject the disruption successfully, please look at the logs above for more details")
	}

	return !errOnInject
}

// reinject reinitialize conf, clean and inject all the disruptions
func reinject(pod *v1.Pod, cmdName string) error {
	var err error

	// Clean all injections to reinject on an empty slate
	if ok := clean(cmdName, true, true); !ok {
		log.Errorw("couldn't clean targets before reinjection. Reinjecting anyway")
	}

	// We rebuild and update the configuration
	for ctnName, ctnID := range targetContainers {
		if ctnName == chaosInitContName && onInit {
			continue
		}

		for i, conf := range configs {
			if conf.TargetContainer == nil || conf.TargetContainer.Name() != ctnName {
				continue
			}

			conf.TargetContainer, err = container.New(ctnID)
			if err != nil {
				log.Warnw("can't create container object", "error", err)

				continue
			}

			// create network namespace and cgroup  manager
			conf.Netns, conf.Cgroup, err = initManagers(conf.TargetContainer.PID())
			if err != nil {
				log.Warnw("can't reinitialize netns manager and cgroup manager", "error", err)

				continue
			}

			injectors[i].UpdateConfig(conf)

			break
		}
	}

	// Reinject target
	if ok := inject(cmdName, true, true); !ok {
		return fmt.Errorf("couldn't reinject target")
	}

	return nil
}

// clean clean all the disruptions using the list of injectors
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

			log.Errorw("disruption cleaning failed", "error", err)
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
func pulse(isInjected *bool, sleepDuration *time.Duration, action func(string, bool, bool) bool, cmdName string) (func(string, bool, bool) bool, error) {
	actionName := ""

	if !*isInjected {
		action = inject
		actionName = "inject"
		*sleepDuration = pulseDormantDuration
	} else {
		action = clean
		actionName = "clean"
		*sleepDuration = pulseActiveDuration
	}

	if ok := action(cmdName, true, true); !ok {
		return nil, fmt.Errorf("error on pulsing disruption mechanism when attempting to %s", actionName)
	}

	newInjected := !*isInjected
	*isInjected = newInjected

	return action, nil
}

// injectAndWait injects the disruption with the configured injector and waits
// for an exit signal to be sent
func injectAndWait(cmd *cobra.Command, args []string) {
	// early exit if an injector configuration failed to be generated during initialization
	if !readyToInject || len(injectors) == 0 {
		log.Error("an injector could not be configured successfully during initialization, aborting the injection now")

		return
	}

	log.Infow("injecting the disruption", "kind", cmd.Name())

	injectSuccess := inject(cmd.Name(), true, false)

	// create and write readiness probe file if injection succeeded so the pod is marked as ready
	if injectSuccess {
		log.Infof("disruption(s) injected, now waiting for an exit signal")

		if err := ioutil.WriteFile(readinessProbeFile, []byte("1"), 0400); err != nil {
			log.Errorw("error writing readiness probe file", "error", err)
		}
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

	deadline, err := time.Parse(time.RFC3339, deadlineRaw)
	if err != nil {
		deadline = time.Now().Add(time.Hour)

		log.Errorw("unable to determine disruption deadline, will self-terminate in one hour instead", "err", err)
	}

	switch {
	case !injectSuccess:
		break
	// those disruptions should not watch target to re-inject on container restart
	case v1beta1.DisruptionIsReinjectable((chaostypes.DisruptionKindName)(cmd.Name())):
	case level == chaostypes.DisruptionLevelNode:
		if pulseActiveDuration > 0 && pulseDormantDuration > 0 {
			var action func(string, bool, bool) bool

			isInjected := true
			sleepDuration := pulseActiveDuration

			// using a label for the loop to be able to break out of it
		pulsingLoop:
			for { // This loop will wait, clean, wait, inject until a signal is received
				select { // Quit on signal reception or sleep and injects / cleans the disruptions
				case sig := <-signals:
					log.Infow("an exit signal has been received", "signal", sig.String())

					return
				case <-time.After(getDuration(deadline)):
					log.Infow("duration has expired")

					return
				case <-time.After(sleepDuration):
					action, err = pulse(&isInjected, &sleepDuration, action, cmd.Name())
					if err != nil {
						log.Errorf(err.Error())

						// break of PulsingLoop only
						break pulsingLoop
					}
				}
			}
		}
	default:
		if onInit {
			log.Debugw("the init container will not get restarted on container restart")
		}

		// we watch for targeted pod containers restart to reinject
		err := watchTargetAndReinject(deadline, cmd.Name(), pulseActiveDuration, pulseDormantDuration)

		if err != nil {
			log.Errorw("couldn't continue watching targeted pod", "err", err)
		} else {
			return
		}
	}

	// wait for an exit signal, this is a blocking call
	select {
	case sig := <-signals:
		log.Infow("an exit signal has been received", "signal", sig.String())
	case <-time.After(getDuration(deadline)):
		log.Infow("duration has expired")
	}
}

// watchTargetAndReinject handle reinjection of the disruption on container restart
func watchTargetAndReinject(deadline time.Time, commandName string, pulseActiveDuration time.Duration, pulseDormantDuration time.Duration) error {
	// we keep track of resource version in case of errors during watch to pick up where we were before the error
	resourceVersion, err := getPodResourceVersion()
	if err != nil {
		return err
	}

	pulseIndexIsInjected := true

	var pulseSleepDuration time.Duration

	// set sleepDuration to after deadline duration to never go into the pulsing condition
	if pulseActiveDuration > 0 && pulseDormantDuration > 0 {
		pulseSleepDuration = pulseActiveDuration
	} else {
		pulseSleepDuration = getDuration(deadline) + time.Hour
	}

	var channel <-chan watch.Event

	var actionOnPulse func(string, bool, bool) bool

	for {
		if channel == nil {
			if channel, err = initPodWatch(resourceVersion); err != nil {
				return err
			}
		}

		select {
		case sig := <-signals:
			log.Infow("an exit signal has been received", "signal", sig.String())
			// We've already consumed the signal from the channel, so our caller won't find it when checking after we return.
			// Thus we need to put the signal back into the channel. We do it in a gothread in case we are blocked when writing to the channel
			go func() { signals <- sig }()

			return nil
		case <-time.After(getDuration(deadline)):
			log.Infow("duration has expired")

			return nil
		// shouldn't go there if it's not a pulsing disruption
		case <-time.After(pulseSleepDuration):
			if pulseActiveDuration == 0 || pulseDormantDuration == 0 {
				break
			}

			actionOnPulse, err = pulse(&pulseIndexIsInjected, &pulseSleepDuration, actionOnPulse, commandName)
			if err != nil {
				return err
			}
		case event, ok := <-channel: // We have changes in the pod watched
			log.Debugw("received event during target watch", "type", event.Type)

			if !ok {
				channel = nil

				continue
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				log.Debugw("received event was not a pod", "event", event, "event.Object", event.Object)

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
							// We don't have the latest resource version, but we can't seem to find a new one.
							// Wait 10 seconds and retry.
							time.Sleep(time.Second * 10)
							// An unset resource version is an implicit "latest" request.
							resourceVersion = ""
						} else {
							resourceVersion = updResourceVersion
						}

						channel = nil

						log.Debugw("restarting pod watching channel with newest resource version", "resourceVersion", resourceVersion)

						continue
					}
				}

				return fmt.Errorf("watched object received from event is not a pod")
			}

			if event.Type == watch.Bookmark {
				resourceVersion = pod.ResourceVersion
				channel = nil

				log.Debugw("received bookmark event, new resource version found", "resourceVersion", pod.ResourceVersion)
			}

			if event.Type != watch.Modified {
				continue
			}

			notReady := false

			// We wait for the pod to have all containers ready.
			for _, status := range pod.Status.ContainerStatuses {
				// we don't control the state of the init container
				if targetContainers[status.Name] == "" || (onInit && status.Name == chaosInitContName) {
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

			hasChanged, err := updateTargetContainersAndDetectChange(pod)
			if err != nil {
				return err
			}

			if hasChanged {
				if err := reinject(pod, commandName); err != nil {
					return err
				}
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

	if err := backoff.RetryNotify(cleanFinalizer, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3), retryNotifyHandler); err != nil {
		log.Errorw("couldn't safely remove this pod's finalizer", "err", err)
	}

	log.Info("disruption(s) cleaned, now exiting")
}

func cleanFinalizer() error {
	if len(configs) == 0 {
		err := fmt.Errorf("no configuration available for this disruption")

		log.Warnw("couldn't GET this pod in order to remove its finalizer", "pod", os.Getenv(env.InjectorPodName), "err", err)

		return err
	}

	pod, err := configs[0].K8sClient.CoreV1().Pods(chaosNamespace).Get(context.Background(), os.Getenv(env.InjectorPodName), metav1.GetOptions{})
	if err != nil {
		log.Warnw("couldn't GET this pod in order to remove its finalizer", "pod", os.Getenv(env.InjectorPodName), "err", err)
		return err
	}

	controllerutil.RemoveFinalizer(pod, chaostypes.ChaosPodFinalizer)

	_, err = configs[0].K8sClient.CoreV1().Pods(chaosNamespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	if err != nil {
		log.Warnw("couldn't remove this pod's finalizer", "pod", os.Getenv(env.InjectorPodName), "err", err)
		return err
	}

	return nil
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

// getDuration returns the time between time.Now() and when the disruption is due to expire
// This gives the chaos pod plenty of time to clean up before it hits activeDeadlineSeconds and becomes Failed
func getDuration(deadline time.Time) time.Duration {
	return time.Until(deadline)
}

// getPodResourceVersion get the resource version of the targeted pod
func getPodResourceVersion() (string, error) {
	target, err := clientset.CoreV1().Pods(disruptionNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return target.ResourceVersion, nil
}

// updateTargetContainersAndDetectChange get all target container infos to determine if one container has changed ID
// if it has changed ID, the container just restarted and need to be reinjected
func updateTargetContainersAndDetectChange(pod *v1.Pod) (bool, error) {
	var err error

	// transform map of targetContainer info (name, id) to only an array of names
	targetContainerNames := []string{}
	for name := range targetContainers {
		targetContainerNames = append(targetContainerNames, name)
	}

	// update map of targetContainer info (name, id)
	targetContainers, err = utils.GetTargetedContainersInfo(pod, targetContainerNames)
	if err != nil {
		log.Warnw("couldn't get containers info. Waiting for next change to reinject", "err", err)

		return false, err
	}

	// Determine if reinjection is needed
	for ctnName, ctnID := range targetContainers {
		// we don't check for init containers
		if ctnName == chaosInitContName && onInit {
			continue
		}

		for _, conf := range configs {
			// we check if a container has changed IDs, meaning it was restarted
			if conf.TargetContainer.Name() != ctnName || conf.TargetContainer.ID() == ctnID {
				continue
			}

			return true, nil
		}
	}

	return false, nil
}

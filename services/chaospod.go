// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ChaosPodService is an interface that defines methods for managing chaos pods of a disruption on Kubernetes pods.
type ChaosPodService interface {
	// GetChaosPodsOfDisruption retrieves a list chaos pods of a disruption for the given labels.
	GetChaosPodsOfDisruption(ctx context.Context, instance *chaosv1beta1.Disruption, ls labels.Set) ([]corev1.Pod, error)

	// HandleChaosPodTermination handles the termination of a chaos pod during a disruption event.
	HandleChaosPodTermination(ctx context.Context, disruption *chaosv1beta1.Disruption, pod *corev1.Pod) (stuckOnRemoval bool, err error)

	// DeletePod deletes a pod from the Kubernetes cluster.
	DeletePod(ctx context.Context, pod corev1.Pod) bool

	// GenerateChaosPodOfDisruption generates a pod for the disruption.
	GenerateChaosPodOfDisruption(disruption *chaosv1beta1.Disruption, targetName, targetNodeName string, args []string, kind chaostypes.DisruptionKindName) corev1.Pod

	// GenerateChaosPodsOfDisruption generates a list of chaos pods for the disruption.
	GenerateChaosPodsOfDisruption(instance *chaosv1beta1.Disruption, targetName, targetNodeName string, targetContainers map[string]string, targetPodIP string, injectionHasCloudHosts *bool) ([]corev1.Pod, error)

	// GetPodInjectorArgs retrieves arguments to inject into a pod.
	GetPodInjectorArgs(pod corev1.Pod) []string

	// CreatePod creates a pod in the Kubernetes cluster.
	CreatePod(ctx context.Context, pod *corev1.Pod) error

	// WaitForPodCreation waits for a pod to be created in the Kubernetes cluster.
	WaitForPodCreation(ctx context.Context, pod corev1.Pod) error

	// HandleOrphanedChaosPods handles orphaned chaos pods based on a controller request.
	HandleOrphanedChaosPods(ctx context.Context, req ctrl.Request) error
}

// ChaosPodServiceInjectorConfig contains configuration options for the injector.
type ChaosPodServiceInjectorConfig struct {
	ServiceAccount                string            // Service account to be used by the injector.
	Image                         string            // Image to be used for the injector.
	Annotations, Labels           map[string]string // Annotations and labels to be applied to injected pods.
	NetworkDisruptionAllowedHosts []string          // List of hosts allowed during network disruption.
	DNSDisruptionDNSServer        string            // DNS server to be used for DNS disruption.
	DNSDisruptionKubeDNS          string            // KubeDNS server to be used for DNS disruption.
	ImagePullSecrets              string            // Image pull secrets for the injector.
}

// ChaosPodServiceConfig contains configuration options for the chaosPodService.
type ChaosPodServiceConfig struct {
	Client                        client.Client                              // Kubernetes client for interacting with the API server.
	Log                           *zap.SugaredLogger                         // Logger for logging.
	ChaosNamespace                string                                     // Namespace where chaos-related resources are located.
	TargetSelector                targetselector.TargetSelector              // Target selector for selecting target pods.
	Injector                      ChaosPodServiceInjectorConfig              // Configuration options for the injector.
	ImagePullSecrets              string                                     // Image pull secrets for the chaosPodService.
	MetricsSink                   metrics.Sink                               // Sink for exporting metrics.
	CloudServicesProvidersManager cloudservice.CloudServicesProvidersManager // Manager for cloud service providers.
}

type chaosPodService struct {
	config ChaosPodServiceConfig
}

type ChaosPodAllowedErrors map[string]struct{}

func (c ChaosPodAllowedErrors) isNotAllowed(errorMsg string) bool {
	_, allowed := chaosPodAllowedErrors[errorMsg]

	return !allowed
}

var chaosPodAllowedErrors = ChaosPodAllowedErrors{
	"pod is not running": {},
	"node is not ready":  {},
}

// NewChaosPodService create a new chaos pod service instance with the provided configuration.
func NewChaosPodService(config ChaosPodServiceConfig) (ChaosPodService, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("you must provide a non nil Kubernetes client")
	}

	return &chaosPodService{
		config: config,
	}, nil
}

// CreatePod creates a pod in the Kubernetes cluster.
func (m *chaosPodService) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	return m.config.Client.Create(ctx, pod)
}

// GetChaosPodsOfDisruption retrieves a list of chaos-related pods affected by a disruption event,
// filtered by the provided labels.
func (m *chaosPodService) GetChaosPodsOfDisruption(ctx context.Context, instance *chaosv1beta1.Disruption, ls labels.Set) ([]corev1.Pod, error) {
	return chaosv1beta1.GetChaosPods(ctx, m.config.Log, m.config.ChaosNamespace, m.config.Client, instance, ls)
}

// HandleChaosPodTermination handles the termination of a chaos-related pod during a disruption event.
func (m *chaosPodService) HandleChaosPodTermination(ctx context.Context, disruption *chaosv1beta1.Disruption, chaosPod *corev1.Pod) (stuckOnRemoval bool, err error) {
	// Ignore chaos pods not having the finalizer anymore
	if !controllerutil.ContainsFinalizer(chaosPod, chaostypes.ChaosPodFinalizer) {
		return false, nil
	}

	// Ignore chaos pods that are not being deleted
	if chaosPod.DeletionTimestamp.IsZero() {
		return false, nil
	}

	target := chaosPod.Labels[chaostypes.TargetLabel]

	// Check if the target of the disruption is healthy (running and ready).
	if err := m.config.TargetSelector.TargetIsHealthy(target, m.config.Client, disruption); err != nil {
		// return the error unless we have a specific reason to ignore it.
		if !apierrors.IsNotFound(err) && chaosPodAllowedErrors.isNotAllowed(strings.ToLower(err.Error())) {
			return false, err
		}

		// If the target is not in a good shape, proceed with cleanup phase.
		m.config.Log.Infow("Target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "target", target)

		// Remove the finalizer for the chaos pod since cleanup won't be fully reliable.
		err = m.removeFinalizerForChaosPod(ctx, chaosPod)

		return false, err
	}

	// It is always safe to remove some chaos pods. It is usually hard to tell if these chaos pods have
	// succeeded or not, but they have no possibility of leaving side effects, so we choose to always remove the finalizer.
	if chaosv1beta1.DisruptionHasNoSideEffects(chaosPod.Labels[chaostypes.DisruptionKindLabel]) {
		err = m.removeFinalizerForChaosPod(ctx, chaosPod)
		return false, err
	}

	shouldRemoveFinalizer := false

	switch chaosPod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodPending:
		// we can remove the pod and the finalizer, so that it'll be garbage collected
		shouldRemoveFinalizer = true
	case corev1.PodFailed:
		// we need to determine if we can remove it safely or if we need to block disruption deletion
		// check if a container has been created (if not, the disruption was not injected)
		if len(chaosPod.Status.ContainerStatuses) == 0 {
			shouldRemoveFinalizer = true
		}

		// if the pod died only because it exceeded its activeDeadlineSeconds, we can remove the finalizer
		if chaosPod.Status.Reason == "DeadlineExceeded" {
			shouldRemoveFinalizer = true
		}

		// check if the container was able to start or not
		// if not, we can safely delete the pod since the disruption was not injected
		for _, cs := range chaosPod.Status.ContainerStatuses {
			if cs.Name != "injector" {
				continue
			}

			if cs.State.Terminated != nil && cs.State.Terminated.Reason == "StartError" {
				shouldRemoveFinalizer = true
			}

			break
		}
	default:
		// If we're in this default case, then the chaos pod is not yet in a "terminated" state
		// It's likely still cleaning up the relevant disruption, in which case we aren't ready
		// to try to remove the finalizer, or to mark it as StuckOnRemoval, so we should just return here.
		// And check again next time we reconcile.
		return false, nil
	}

	if shouldRemoveFinalizer {
		// Remove the finalizer for the chaos pod since cleanup was successful.
		err = m.removeFinalizerForChaosPod(ctx, chaosPod)
		return false, err
	}

	return true, nil
}

// DeletePod attempts to delete the specified pod from the Kubernetes cluster.
// Returns true if deletion was successful, otherwise returns false.
func (m *chaosPodService) DeletePod(ctx context.Context, pod corev1.Pod) bool {
	m.config.Log.Infow("terminating chaos pod to trigger cleanup", "chaosPod", pod.Name)

	if err := m.deletePod(ctx, pod); err != nil {
		m.config.Log.Errorw("Error terminating chaos pod", "error", err, "chaosPod", pod.Name)

		return false
	}

	return true
}

// GenerateChaosPodsOfDisruption generates a list of chaos pods for the given disruption instance,
// target information, and other configuration parameters.
func (m *chaosPodService) GenerateChaosPodsOfDisruption(instance *chaosv1beta1.Disruption, targetName string, targetNodeName string, targetContainers map[string]string, targetPodIP string, injectionHasCloudHosts *bool) ([]corev1.Pod, error) {
	pods := []corev1.Pod{}

	// generate chaos pods for each possible disruptions
	for _, kind := range chaostypes.DisruptionKindNames {
		subspec := instance.Spec.DisruptionKindPicker(kind)
		if reflect.ValueOf(subspec).IsNil() {
			continue
		}

		pulseActiveDuration, pulseDormantDuration, pulseInitialDelay := time.Duration(0), time.Duration(0), time.Duration(0)
		if instance.Spec.Pulse != nil {
			pulseInitialDelay = instance.Spec.Pulse.InitialDelay.Duration()
			pulseActiveDuration = instance.Spec.Pulse.ActiveDuration.Duration()
			pulseDormantDuration = instance.Spec.Pulse.DormantDuration.Duration()
		}

		notInjectedBefore := instance.TimeToInject()
		allowedHosts := m.config.Injector.NetworkDisruptionAllowedHosts

		// get the ip ranges of cloud provider services
		if instance.Spec.Network != nil && !*injectionHasCloudHosts {
			err := instance.Spec.Network.UpdateHostsOnCloudDisruption(m.config.CloudServicesProvidersManager)
			if err != nil {
				return nil, err
			}

			hasCloudHosts := true
			injectionHasCloudHosts = &hasCloudHosts

			// remove default allowed hosts if disabled
			if instance.Spec.Network.DisableDefaultAllowedHosts {
				allowedHosts = make([]string, 0)
			}
		}

		xargs := chaosapi.DisruptionArgs{
			Level:                instance.Spec.Level,
			Kind:                 kind,
			TargetContainers:     targetContainers,
			TargetName:           targetName,
			TargetNodeName:       targetNodeName,
			TargetPodIP:          targetPodIP,
			DryRun:               instance.Spec.DryRun,
			DisruptionName:       instance.Name,
			DisruptionNamespace:  instance.Namespace,
			OnInit:               instance.Spec.OnInit,
			PulseInitialDelay:    pulseInitialDelay,
			PulseActiveDuration:  pulseActiveDuration,
			PulseDormantDuration: pulseDormantDuration,
			NotInjectedBefore:    notInjectedBefore,
			MetricsSink:          m.config.MetricsSink.GetSinkName(),
			AllowedHosts:         allowedHosts,
			DNSServer:            m.config.Injector.DNSDisruptionDNSServer,
			KubeDNS:              m.config.Injector.DNSDisruptionKubeDNS,
			ChaosNamespace:       m.config.ChaosNamespace,
		}

		args := xargs.CreateCmdArgs(subspec.GenerateArgs())

		pod := m.GenerateChaosPodOfDisruption(instance, targetName, targetNodeName, args, kind)

		pods = append(pods, pod)
	}

	return pods, nil
}

// GenerateChaosPodOfDisruption generates a chaos pod for a specific disruption.
func (m *chaosPodService) GenerateChaosPodOfDisruption(disruption *chaosv1beta1.Disruption, targetName, targetNodeName string, args []string, kind chaostypes.DisruptionKindName) (chaosPod corev1.Pod) {
	// volume host path type definitions
	hostPathDirectory := corev1.HostPathDirectory
	hostPathFile := corev1.HostPathFile

	// The default TerminationGracePeriodSeconds is 30s. This can be too low for a chaos pod to finish cleaning. After TGPS passes,
	// the signal sent to a pod becomes SIGKILL, which will interrupt any in-progress cleaning. By double this to 1 minute in the pod podSpec itself,
	// ensures that whether a chaos pod is deleted directly or by deleting a disruption, it will have time to finish cleaning up after itself.
	terminationGracePeriod := int64(60) // 60 seconds

	// Chaos pods will clean themselves automatically when duration expires, so we set activeDeadlineSeconds to ten seconds after that
	// to give time for cleaning
	activeDeadlineSeconds := int64(disruption.RemainingDuration().Seconds()) + 10

	// It can cause abnormalities in the status of a Disruption if injector pods consider themselves complete in the ~second before the chaos-controller believes the disruption is complete.
	// To avoid making our termination state machine logic more complicated, we will pad the injector pod duration by two seconds.
	// See https://github.com/DataDog/chaos-controller/issues/748
	args = append(args,
		"--deadline", time.Now().Add(chaostypes.InjectorPadDuration).Add(disruption.RemainingDuration()).Format(time.RFC3339))

	chaosPod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("chaos-%s-", disruption.Name),      // generate the pod name automatically with a prefix
			Namespace:    m.config.ChaosNamespace,                        // chaos pods need to be in the same namespace as their service account to run
			Annotations:  m.config.Injector.Annotations,                  // add extra annotations passed to the controller
			Labels:       m.generateLabels(disruption, targetName, kind), // add default and extra podLabels passed to the controller
		},
		Spec: m.generateChaosPodSpec(targetNodeName, terminationGracePeriod, activeDeadlineSeconds, args, hostPathDirectory, hostPathFile),
	}

	// add finalizer to the pod, so it is not deleted before we can control its exit status
	controllerutil.AddFinalizer(&chaosPod, chaostypes.ChaosPodFinalizer)

	return chaosPod
}

// GetPodInjectorArgs retrieves the arguments used by the "injector" container in a chaos pod.
func (m *chaosPodService) GetPodInjectorArgs(chaosPod corev1.Pod) []string {
	chaosPodArgs := []string{}

	if len(chaosPod.Spec.Containers) == 0 {
		m.config.Log.Errorw("no containers found in chaos pod spec", "chaosPodSpec", chaosPod.Spec)

		return chaosPodArgs
	}

	for _, container := range chaosPod.Spec.Containers {
		if container.Name == "injector" {
			chaosPodArgs = container.Args
		}
	}

	if len(chaosPodArgs) == 0 {
		m.config.Log.Warnw("unable to find the args for this chaos pod", "chaosPodName", chaosPod.Name, "chaosPodSpec", chaosPod.Spec, "chaosPodContainerCount", len(chaosPod.Spec.Containers))
	}

	return chaosPodArgs
}

// WaitForPodCreation waits for the given pod to be created
// it tries to get the pod using an exponential backoff with a max retry interval of 1 second and a max duration of 30 seconds
// if an unexpected error occurs (an error other than a "not found" error), the retry loop is stopped
func (m *chaosPodService) WaitForPodCreation(ctx context.Context, pod corev1.Pod) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxInterval = time.Second
	expBackoff.MaxElapsedTime = 30 * time.Second

	return backoff.Retry(func() error {
		err := m.config.Client.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, &pod)
		if client.IgnoreNotFound(err) != nil {
			return backoff.Permanent(err)
		}

		return err
	}, expBackoff)
}

// HandleOrphanedChaosPods handles orphaned chaos pods related to a specific disruption.
func (m *chaosPodService) HandleOrphanedChaosPods(ctx context.Context, req ctrl.Request) error {
	ls := make(map[string]string)

	// Set labels for filtering chaos pods related to the specified disruption.
	ls[chaostypes.DisruptionNameLabel] = req.Name
	ls[chaostypes.DisruptionNamespaceLabel] = req.Namespace

	// Retrieve chaos pods matching the specified labels.
	pods, err := m.GetChaosPodsOfDisruption(ctx, nil, ls)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		m.handleMetricSinkError(m.config.MetricsSink.MetricOrphanFound([]string{"disruption:" + req.Name, "chaosPod:" + pod.Name, "namespace:" + req.Namespace}))

		target := pod.Labels[chaostypes.TargetLabel]

		var p corev1.Pod

		m.config.Log.Infow("checking if we can clean up orphaned chaos pod", "chaosPod", pod.Name, "target", target)

		// if target doesn't exist, we can try to clean up the chaos pod
		if err = m.config.Client.Get(ctx, types.NamespacedName{Name: target, Namespace: req.Namespace}, &p); apierrors.IsNotFound(err) {
			m.config.Log.Warnw("orphaned chaos pod detected, will attempt to delete", "chaosPod", pod.Name)

			if err = m.removeFinalizerForChaosPod(ctx, &pod); err != nil {
				continue
			}

			// if the chaos pod still exists after having its finalizer removed, delete it
			if err = m.deletePod(ctx, pod); err != nil {
				if chaosv1beta1.IsUpdateConflictError(err) {
					m.config.Log.Infow("retryable error deleting orphaned chaos pod", "error", err, "chaosPod", pod.Name)
				} else {
					m.config.Log.Errorw("error deleting orphaned chaos pod", "error", err, "chaosPod", pod.Name)
				}
			}
		}
	}

	return nil
}

func (m *chaosPodService) generateLabels(disruption *chaosv1beta1.Disruption, targetName string, kind chaostypes.DisruptionKindName) map[string]string {
	podLabels := make(map[string]string)

	for k, v := range m.config.Injector.Labels {
		podLabels[k] = v
	}

	podLabels[chaostypes.TargetLabel] = targetName                        // target name label
	podLabels[chaostypes.DisruptionKindLabel] = string(kind)              // disruption kind label
	podLabels[chaostypes.DisruptionNameLabel] = disruption.Name           // disruption name label, used to determine ownership
	podLabels[chaostypes.DisruptionNamespaceLabel] = disruption.Namespace // disruption namespace label, used to determine ownership

	return podLabels
}

func (m *chaosPodService) generateChaosPodSpec(targetNodeName string, terminationGracePeriod int64, activeDeadlineSeconds int64, args []string, hostPathDirectory corev1.HostPathType, hostPathFile corev1.HostPathType) corev1.PodSpec {
	podSpec := corev1.PodSpec{
		HostPID:                       true,                             // enable host pid
		RestartPolicy:                 corev1.RestartPolicyNever,        // do not restart the pod on fail or completion
		NodeName:                      targetNodeName,                   // specify node name to schedule the pod
		ServiceAccountName:            m.config.Injector.ServiceAccount, // service account to use
		TerminationGracePeriodSeconds: &terminationGracePeriod,
		ActiveDeadlineSeconds:         &activeDeadlineSeconds,
		Containers: []corev1.Container{
			{
				Name:            "injector",              // container name
				Image:           m.config.Injector.Image, // container image gathered from controller flags
				ImagePullPolicy: corev1.PullIfNotPresent, // pull the image only when it is not present
				Args:            args,                    // pass disruption arguments
				SecurityContext: &corev1.SecurityContext{
					Privileged: func() *bool { b := true; return &b }(), // enable privileged mode
				},
				ReadinessProbe: &corev1.Probe{ // define readiness probe (file created by the injector when the injection is successful)
					PeriodSeconds:    1,
					FailureThreshold: 5,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"test", "-f", "/tmp/readiness_probe"},
						},
					},
				},
				Resources: corev1.ResourceRequirements{ // set resources requests and limits to zero
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
						corev1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
					},
				},
				Env: []corev1.EnvVar{ // define environment variables
					{
						Name: env.InjectorTargetPodHostIP,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "status.hostIP",
							},
						},
					},
					{
						Name: env.InjectorChaosPodIP,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "status.podIP",
							},
						},
					},
					{
						Name: env.InjectorPodName,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
					{
						Name:  env.InjectorMountHost,
						Value: "/mnt/host/",
					},
					{
						Name:  env.InjectorMountProc,
						Value: "/mnt/host/proc/",
					},
					{
						Name:  env.InjectorMountSysrq,
						Value: "/mnt/sysrq",
					},
					{
						Name:  env.InjectorMountSysrqTrigger,
						Value: "/mnt/sysrq-trigger",
					},
					{
						Name:  env.InjectorMountCgroup,
						Value: "/mnt/cgroup/",
					},
				},
				VolumeMounts: []corev1.VolumeMount{ // define volume mounts required for disruptions to work
					{
						Name:      "run",
						MountPath: "/run",
					},
					{
						Name:      "sysrq",
						MountPath: "/mnt/sysrq",
					},
					{
						Name:      "sysrq-trigger",
						MountPath: "/mnt/sysrq-trigger",
					},
					{
						Name:      "cgroup",
						MountPath: "/mnt/cgroup",
					},
					{
						Name:      "host",
						MountPath: "/mnt/host",
						ReadOnly:  true,
					},
					{
						Name:      "boot",
						MountPath: "/boot",
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []corev1.Volume{ // declare volumes required for disruptions to work
			{
				Name: "run",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/run",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "proc",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "sysrq",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc/sys/kernel/sysrq",
						Type: &hostPathFile,
					},
				},
			},
			{
				Name: "sysrq-trigger",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/proc/sysrq-trigger",
						Type: &hostPathFile,
					},
				},
			},
			{
				Name: "cgroup",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys/fs/cgroup",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "host",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/",
						Type: &hostPathDirectory,
					},
				},
			},
			{
				Name: "boot",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/boot",
						Type: &hostPathDirectory,
					},
				},
			},
		},
	}

	if m.config.ImagePullSecrets != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: m.config.ImagePullSecrets,
			},
		}
	}

	return podSpec
}

func (m *chaosPodService) removeFinalizerForChaosPod(ctx context.Context, chaosPod *corev1.Pod) error {
	controllerutil.RemoveFinalizer(chaosPod, chaostypes.ChaosPodFinalizer)

	if err := m.config.Client.Update(ctx, chaosPod); err != nil {
		if chaosv1beta1.IsUpdateConflictError(err) {
			m.config.Log.Debugw("cannot remove chaos pod finalizer, need to re-reconcile", "error", err)
		} else {
			m.config.Log.Errorw("error removing chaos pod finalizer", "error", err, "chaosPod", chaosPod.Name)
		}

		return err
	}

	return nil
}

func (m *chaosPodService) handleMetricSinkError(err error) {
	if err != nil {
		m.config.Log.Errorw("error sending a metric", "error", err)
	}
}

func (m *chaosPodService) deletePod(ctx context.Context, pod corev1.Pod) error {
	// Attempt to delete the pod using the Kubernetes client.
	// Ignore "not found" errors using client.IgnoreNotFound to avoid returning an error if the pod is already deleted.
	if err := m.config.Client.Delete(ctx, &pod); client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

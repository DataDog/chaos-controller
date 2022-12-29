// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package controllers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/safemode"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/utils"
	"github.com/cenkalti/backoff"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/env"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	client.Client
	BaseLog                               *zap.SugaredLogger
	Scheme                                *runtime.Scheme
	Recorder                              record.EventRecorder
	MetricsSink                           metrics.Sink
	TargetSelector                        targetselector.TargetSelector
	InjectorAnnotations                   map[string]string
	InjectorLabels                        map[string]string
	InjectorServiceAccount                string
	InjectorImage                         string
	ImagePullSecrets                      string
	log                                   *zap.SugaredLogger
	ChaosNamespace                        string
	InjectorDNSDisruptionDNSServer        string
	InjectorDNSDisruptionKubeDNS          string
	InjectorNetworkDisruptionAllowedHosts []string
	SafetyNets                            []safemode.Safemode
	ExpiredDisruptionGCDelay              *time.Duration
	CacheContextStore                     map[string]CtxTuple
	Controller                            controller.Controller
	Reader                                client.Reader // Use the k8s API without the cache
	EnableObserver                        bool          // Enable Observer on targets update with dynamic targeting
	CloudServicesProvidersManager         *cloudservice.CloudServicesProvidersManager
}

type CtxTuple struct {
	Ctx                      context.Context
	CancelFunc               context.CancelFunc
	DisruptionNamespacedName types.NamespacedName
}

//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch;list;watch;get
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=list;watch
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch

func (r *DisruptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	instance := &chaosv1beta1.Disruption{}
	tsStart := time.Now()

	rand.Seed(time.Now().UnixNano())

	// prepare logger instance context
	// NOTE: it is valid while we don't do concurrent reconciling
	// because the logger instance is pointer, concurrent reconciling would create a race condition
	// where the logger context would change for all ongoing reconcile loops
	// in the case we enable concurrent reconciling, we should create one logger instance per reconciling call
	r.log = r.BaseLog.With("disruptionName", req.Name, "disruptionNamespace", req.Namespace)

	// reconcile metrics
	r.handleMetricSinkError(r.MetricsSink.MetricReconcile())

	defer func() func() {
		return func() {
			tags := []string{}
			if instance.Name != "" {
				tags = append(tags, "disruptionName:"+instance.Name, "namespace:"+instance.Namespace)
			}

			r.handleMetricSinkError(r.MetricsSink.MetricReconcileDuration(time.Since(tsStart), tags))
		}
	}()()

	if err := r.Get(context.Background(), req.NamespacedName, instance); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// If we're reconciling but without an instance, then we must have been triggered by the pod informer
			// We should check for and delete any orphaned chaos pods
			err = r.handleOrphanedChaosPods(req)
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.manageInstanceSelectorCache(instance); err != nil {
		r.log.Errorw("error managing selector cache", "error", err)
	}

	// handle any chaos pods being deleted (either by the disruption deletion or by an external event)
	if err := r.handleChaosPodsTermination(instance); err != nil {
		if isModifiedError(err) {
			r.log.Warnw("error handling chaos pods termination", "error", err)

			return ctrl.Result{}, nil
		}

		r.log.Errorw("error handling chaos pods termination", "error", err)

		return ctrl.Result{}, err
	}

	// check whether the object is being deleted or not
	if !instance.DeletionTimestamp.IsZero() {
		// the instance is being deleted, clean it if the finalizer is still present
		if controllerutil.ContainsFinalizer(instance, chaostypes.DisruptionFinalizer) {
			isCleaned, err := r.cleanDisruption(instance)
			if err != nil {
				r.log.Errorw("error cleaning disruption", "error", err)

				return ctrl.Result{}, err
			}

			// if not cleaned yet, requeue and reconcile again in 15s-20s
			// the reason why we don't rely on the exponential backoff here is that it retries too fast at the beginning
			if !isCleaned {
				requeueAfter := time.Duration(rand.Intn(5)+15) * time.Second //nolint:gosec

				r.log.Infow(fmt.Sprintf("disruption has not been fully cleaned yet, re-queuing in %v", requeueAfter))

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueAfter,
				}, r.Update(context.Background(), instance)
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.log.Infow("all chaos pods are cleaned up; removing disruption finalizer")
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionFinished, "")
			r.clearInstanceSelectorCache(instance)
			controllerutil.RemoveFinalizer(instance, chaostypes.DisruptionFinalizer)

			if err := r.Update(context.Background(), instance); err != nil {
				if isModifiedError(err) {
					r.log.Warnw("error removing disruption finalizer", "error", err)
				} else {
					r.log.Errorw("error removing disruption finalizer", "error", err)
				}

				return ctrl.Result{}, err
			}

			// send reconciling duration metric
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionCompletedDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
			r.emitKindCountMetrics(instance)

			return ctrl.Result{}, nil
		}
	} else {
		if err := r.validateDisruptionSpec(instance); err != nil {
			return ctrl.Result{Requeue: false}, err
		}

		// initialize all safety nets for future use
		if instance.Spec.Unsafemode == nil || !instance.Spec.Unsafemode.DisableAll {
			// initialize all relevant safety nets for the first time
			if len(r.SafetyNets) == 0 {
				r.SafetyNets = []safemode.Safemode{}
				r.SafetyNets = safemode.AddAllSafemodeObjects(*instance, r.Client)
			} else {
				// it is possible for a disruption to be restarted with new parameters, therefore safety nets need to be reinitialized to catch that case
				// so that we are not using values from older versions of a disruption for safety nets
				safemode.Reinit(r.SafetyNets, *instance, r.Client)
			}
		}

		// the injection is being created or modified, apply needed actions
		controllerutil.AddFinalizer(instance, chaostypes.DisruptionFinalizer)
		if err := r.Update(context.Background(), instance); err != nil {
			r.log.Errorw("error adding disruption finalizer", "error", err)

			return ctrl.Result{Requeue: true}, err
		}

		// If the disruption is at least r.ExpiredDisruptionGCDelay older than when its duration ended, then we should delete it.
		// calculateRemainingDurationSeconds returns the seconds until (or since, if negative) the duration's deadline. We compare it to negative ExpiredDisruptionGCDelay,
		// and if less than that, it means we have exceeded the deadline by at least ExpiredDisruptionGCDelay, so we can delete
		if r.ExpiredDisruptionGCDelay != nil && (calculateRemainingDuration(*instance) <= (-1 * *r.ExpiredDisruptionGCDelay)) {
			r.log.Infow("disruption has lived for more than its duration, it will now be deleted.", "duration", instance.Spec.Duration)
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionGCOver, r.ExpiredDisruptionGCDelay.String())

			var err error

			if err = r.Client.Delete(context.Background(), instance); err != nil {
				r.log.Errorw("error deleting disruption after its duration expired", "error", err)
			}

			return ctrl.Result{Requeue: true}, err
		} else if calculateRemainingDuration(*instance) <= 0 {
			if _, err := r.updateInjectionStatus(instance); err != nil {
				if isModifiedError(err) {
					r.log.Warnw("error updating disruption injection status", "error", err)
				} else {
					r.log.Errorw("error updating disruption injection status", "error", err)
				}

				return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
			}

			if r.ExpiredDisruptionGCDelay != nil {
				requeueDelay := *r.ExpiredDisruptionGCDelay

				r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionDurationOver, requeueDelay.String())
				r.log.Debugw("requeuing disruption to check for its expiration", "requeueDelay", requeueDelay.String())

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueDelay,
				}, nil
			}

			return ctrl.Result{Requeue: false}, nil
		}

		// retrieve targets from label selector
		if err := r.selectTargets(instance); err != nil {
			r.log.Errorw("error selecting targets", "error", err)

			return ctrl.Result{}, fmt.Errorf("error selecting targets: %w", err)
		}

		// start injections
		if err := r.startInjection(instance); err != nil {
			r.log.Errorw("error injecting the disruption", "error", err)

			return ctrl.Result{}, fmt.Errorf("error injecting the disruption: %w", err)
		}

		// send injection duration metric representing the time it took to fully inject the disruption until its creation
		r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))

		// update resource status injection
		// requeue the request if the disruption is not fully injected yet
		injected, err := r.updateInjectionStatus(instance)
		if err != nil {
			if isModifiedError(err) {
				r.log.Warnw("error updating injection status", "error", err)
			} else {
				r.log.Errorw("error updating injection status", "error", err)
			}

			return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
		} else if !injected {
			// requeue after 15-20 seconds, as default 1ms is too quick here
			requeueAfter := time.Duration(rand.Intn(5)+15) * time.Second //nolint:gosec
			r.log.Infow("disruption is not fully injected yet, requeuing", "injectionStatus", instance.Status.InjectionStatus)

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueAfter,
			}, nil
		}
		requeueDelay := calculateRemainingDuration(*instance)

		r.log.Infow("requeuing disruption to check for its expiration", "requeueDelay", requeueDelay.String())

		return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueDelay,
			},
			r.Update(context.Background(), instance)
	}
	// stop the reconcile loop, there's nothing else to do
	return ctrl.Result{}, nil
}

// updateInjectionStatus updates the given instance injection status depending on its chaos pods statuses
// - an instance with all chaos pods "ready" is considered as "injected"
// - an instance with at least one chaos pod as "ready" is considered as "partially injected"
// - an instance with no ready chaos pods is considered as "not injected"
func (r *DisruptionReconciler) updateInjectionStatus(instance *chaosv1beta1.Disruption) (bool, error) {
	r.log.Debugw("checking if injection status needs to be updated", "injectionStatus", instance.Status.InjectionStatus)

	status := chaostypes.DisruptionInjectionStatusNotInjected
	readyPodsCount := 0

	// get chaos pods
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return false, fmt.Errorf("error getting instance chaos pods: %w", err)
	}

	if calculateRemainingDuration(*instance) < 0 {
		status = chaostypes.DisruptionInjectionStatusPreviouslyInjected
	}

	// consider a disruption not injected if no chaos pods are existing
	if status == chaostypes.DisruptionInjectionStatusNotInjected && len(chaosPods) > 0 {
		// consider the disruption "partially injected" if we found at least one ready pod
		status = chaostypes.DisruptionInjectionStatusPartiallyInjected
		// check the chaos pods conditions looking for the ready condition
		for _, chaosPod := range chaosPods {
			podReady := false

			// search for the "Ready" condition in the pod conditions
			for _, cond := range chaosPod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					if cond.Status == corev1.ConditionTrue {
						podReady = true
						readyPodsCount++

						r.updateTargetInjectionStatus(instance, chaosPod, chaostypes.DisruptionInjectionStatusInjected, cond.LastTransitionTime)

						break
					}
				}
			}

			// consider the disruption as not fully injected if at least one not ready pod is found
			if !podReady {
				r.log.Debugw("chaos pod is not ready yet", "chaosPod", chaosPod.Name)
			}
		}

		// consider the disruption as fully injected when all pods are ready
		if len(chaosPods) == readyPodsCount {
			status = chaostypes.DisruptionInjectionStatusInjected
		}
	}

	// update instance status
	instance.Status.InjectionStatus = status

	// we divide by the number of active disruption types because we create one pod per target per disruption
	// ex: we would have 10 pods if we target 50% of all targets with 2 disruption types like network and dns
	// we also consider a target is not fully injected if not all disruptions are injected in it
	if instance.Spec.GetDisruptionCount() == 0 {
		instance.Status.InjectedTargetsCount = 0
	} else {
		instance.Status.InjectedTargetsCount = int(math.Floor(float64(readyPodsCount) / float64(instance.Spec.GetDisruptionCount())))
	}

	if err := r.Client.Status().Update(context.Background(), instance); err != nil {
		return false, err
	}

	// requeue the request if the disruption is not fully injected so we can
	// eventually catch pods that are not ready yet but will be in the future
	if status != chaostypes.DisruptionInjectionStatusInjected {
		return false, nil
	}

	return status == chaostypes.DisruptionInjectionStatusInjected || status == chaostypes.DisruptionInjectionStatusPreviouslyInjected, nil
}

// startInjection creates non-existing chaos pod for the given disruption
func (r *DisruptionReconciler) startInjection(instance *chaosv1beta1.Disruption) error {
	// chaosPodsMap is used to check if a target's chaos pods already exist or not
	chaosPodsMap := make(map[string]map[string]bool, len(instance.Status.TargetInjections))

	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return fmt.Errorf("error getting chaos pods: %w", err)
	}

	// init all the required maps
	for targetName := range instance.Status.TargetInjections {
		chaosPodsMap[targetName] = make(map[string]bool)
	}

	for _, chaosPod := range chaosPods {
		if !instance.Status.HasTarget(chaosPod.Labels[chaostypes.TargetLabel]) {
			r.deleteChaosPod(instance, chaosPod)
		} else {
			chaosPodsMap[chaosPod.Labels[chaostypes.TargetLabel]][chaosPod.Labels[chaostypes.DisruptionKindLabel]] = true
		}
	}

	if len(instance.Status.TargetInjections) > 0 && (len(instance.Status.TargetInjections) != len(chaosPodsMap)) {
		r.log.Infow("starting targets injection", "targets", instance.Status.TargetInjections)
	}

	// iterate through target + existing disruption kind -- to ensure all chaos pods exist
	for targetName := range instance.Status.TargetInjections {
		for _, disKind := range chaostypes.DisruptionKindNames {
			if subspec := instance.Spec.DisruptionKindPicker(disKind); reflect.ValueOf(subspec).IsNil() {
				continue
			}

			if _, ok := chaosPodsMap[targetName][disKind.String()]; ok {
				continue
			}

			if err = r.createChaosPods(instance, targetName); err != nil {
				if !errors.IsNotFound(err) {
					return fmt.Errorf("error creating chaos pods: %w", err)
				}

				r.log.Warnw("could not create chaos pod", "err", err)
			}

			break
		}
	}

	return nil
}

// createChaosPods attempts to create all the chaos pods for a given target. If a given chaos pod already exists, it is not recreated.
func (r *DisruptionReconciler) createChaosPods(instance *chaosv1beta1.Disruption, target string) error {
	var err error

	targetNodeName := ""
	targetContainers := map[string]string{}
	targetPodIP := ""
	targetChaosPods := []*corev1.Pod{}

	// retrieve target
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		pod := corev1.Pod{}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
			return fmt.Errorf("error getting target to inject: %w", err)
		}

		targetNodeName = pod.Spec.NodeName

		// get IDs of targeted containers or all containers
		targetContainers, err = utils.GetTargetedContainersInfo(&pod, instance.Spec.Containers)
		if err != nil {
			return fmt.Errorf("error getting target pod container ID: %w", err)
		}

		// get IP of targeted pod
		targetPodIP = pod.Status.PodIP
	case chaostypes.DisruptionLevelNode:
		targetNodeName = target
	}

	// generate injection pods specs
	if err := r.generateChaosPods(instance, &targetChaosPods, target, targetNodeName, targetContainers, targetPodIP); err != nil {
		return fmt.Errorf("error generating chaos pods: %w", err)
	}

	if len(targetChaosPods) == 0 {
		r.recordEventOnDisruption(instance, chaosv1beta1.EventEmptyDisruption, instance.Name)

		return nil
	}

	// create injection pods
	for _, chaosPod := range targetChaosPods {
		// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
		found, err := r.getChaosPods(instance, chaosPod.Labels)
		if err != nil {
			return fmt.Errorf("error getting existing chaos pods: %w", err)
		}

		// create injection pods if none have been found
		switch len(found) {
		case 0:
			r.log.Infow("creating chaos pod", "target", target)

			// create the pod
			if err = r.Create(context.Background(), chaosPod); err != nil {
				r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionCreationFailed, instance.Name)
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, false))

				return fmt.Errorf("error creating chaos pod: %w", err)
			}

			// wait for the pod to be existing
			if err := r.waitForPodCreation(chaosPod); err != nil {
				r.log.Errorw("error waiting for chaos pod to be created", "error", err, "chaosPod", chaosPod.Name, "target", target)

				continue
			}

			// send metrics and events
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionChaosPodCreated, instance.Name)
			r.recordEventOnTarget(instance, target, chaosv1beta1.EventDisrupted, chaosPod.Name, instance.Name)
			r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, true))
		case 1:
			r.log.Debugw("an injection pod is already existing for the selected target", "target", target, "chaosPod", found[0].Name)
		default:
			var chaosPodNames []string
			for _, pod := range found {
				chaosPodNames = append(chaosPodNames, pod.Name)
			}

			r.log.Errorw("multiple injection pods for one target found", "target", target, "chaosPods", strings.Join(chaosPodNames, ","), "chaosPodLabels", chaosPod.Labels)
		}
	}

	return nil
}

// waitForPodCreation waits for the given pod to be created
// it tries to get the pod using an exponential backoff with a max retry interval of 1 second and a max duration of 30 seconds
// if an unexpected error occurs (an error other than a "not found" error), the retry loop is stopped
func (r *DisruptionReconciler) waitForPodCreation(pod *corev1.Pod) error {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxInterval = time.Second
	expBackoff.MaxElapsedTime = 30 * time.Second

	return backoff.Retry(func() error {
		err := r.Get(context.Background(), types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, pod)
		if client.IgnoreNotFound(err) != nil {
			return backoff.Permanent(err)
		}

		return err
	}, expBackoff)
}

// cleanDisruption triggers the cleanup of the given instance
// for each existing chaos pod for the given instance, the function will delete the chaos pod to trigger its cleanup phase
// the function returns true when no more chaos pods are existing (meaning that it keeps returning false if some pods
// are deleted but still present)
func (r *DisruptionReconciler) cleanDisruption(instance *chaosv1beta1.Disruption) (bool, error) {
	cleaned := true

	// get already existing chaos pods for the given disruption
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return false, err
	}

	// if chaos pods still exist, even if they are completed
	// we consider the disruption as not cleaned
	if len(chaosPods) > 0 {
		cleaned = false
	}

	// terminate running chaos pods to trigger cleanup
	for _, chaosPod := range chaosPods {
		r.deleteChaosPod(instance, chaosPod)
	}

	return cleaned, nil
}

func (r *DisruptionReconciler) handleOrphanedChaosPods(req ctrl.Request) error {
	ls := make(map[string]string)

	ls[chaostypes.DisruptionNameLabel] = req.Name
	ls[chaostypes.DisruptionNamespaceLabel] = req.Namespace

	chaosPods, err := r.getChaosPods(nil, ls)
	if err != nil {
		return err
	}

	for _, chaosPod := range chaosPods {
		r.handleMetricSinkError(r.MetricsSink.MetricOrphanFound([]string{"disruption:" + req.Name, "chaosPod:" + chaosPod.Name, "namespace:" + req.Namespace}))
		target := chaosPod.Labels[chaostypes.TargetLabel]

		var p corev1.Pod

		r.log.Infow("checking if we can clean up orphaned chaos pod", "chaosPod", chaosPod.Name, "target", target)

		// if target doesn't exist, we can try to clean up the chaos pod
		if err := r.Client.Get(context.Background(), types.NamespacedName{Name: target, Namespace: req.Namespace}, &p); errors.IsNotFound(err) {
			r.log.Warnw("orphaned chaos pod detected, will attempt to delete", "chaosPod", chaosPod.Name)
			controllerutil.RemoveFinalizer(&chaosPod, chaostypes.ChaosPodFinalizer)

			if err := r.Client.Update(context.Background(), &chaosPod); err != nil {
				if isModifiedError(err) {
					r.log.Warnw("error removing chaos pod finalizer", "error", err, "chaosPod", chaosPod.Name)
				} else {
					r.log.Errorw("error removing chaos pod finalizer", "error", err, "chaosPod", chaosPod.Name)
				}

				continue
			}

			// if the chaos pod still exists after having its finalizer removed, delete it
			if err := r.Client.Delete(context.Background(), &chaosPod); client.IgnoreNotFound(err) != nil {
				if isModifiedError(err) {
					r.log.Warnw("error deleting orphaned chaos pod", "error", err, "chaosPod", chaosPod.Name)
				} else {
					r.log.Errorw("error deleting orphaned chaos pod", "error", err, "chaosPod", chaosPod.Name)
				}

				continue
			}
		}
	}

	return nil
}

// handleChaosPodsTermination looks at the given instance chaos pods status to handle any terminated pods
// such pods will have their finalizer removed, so they can be garbage collected by Kubernetes
// the finalizer is removed if:
//   - the pod is pending
//   - the pod is succeeded (exit code == 0)
//   - the pod target is not healthy (not existing anymore for instance)
//
// if a finalizer can't be removed because none of the conditions above are fulfilled, the instance is flagged
// as stuck on removal and the pod finalizer won't be removed unless someone does it manually
// the pod target will be moved to ignored targets, so it is not picked up by the next reconcile loop
func (r *DisruptionReconciler) handleChaosPodsTermination(instance *chaosv1beta1.Disruption) error {
	// get already existing chaos pods for the given disruption
	chaosPods, err := r.getChaosPods(instance, nil)
	if err != nil {
		return err
	}

	if len(chaosPods) == 0 {
		return nil
	}

	for _, chaosPod := range chaosPods {
		r.handleChaosPodTermination(instance, chaosPod)
	}

	return r.Status().Update(context.Background(), instance)
}

func (r *DisruptionReconciler) handleChaosPodTermination(instance *chaosv1beta1.Disruption, chaosPod corev1.Pod) {
	removeFinalizer := false
	ignoreStatus := false
	target := chaosPod.Labels[chaostypes.TargetLabel]

	// ignore chaos pods not being deleted or not having the finalizer anymore
	if chaosPod.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(&chaosPod, chaostypes.ChaosPodFinalizer) {
		return
	}

	// check target readiness for cleanup
	// ignore it if it is not ready anymore
	err := r.TargetSelector.TargetIsHealthy(target, r.Client, instance)
	if err != nil {
		if errors.IsNotFound(err) || strings.ToLower(err.Error()) == "pod is not running" || strings.ToLower(err.Error()) == "node is not ready" {
			// if the target is not in a good shape, we still run the cleanup phase but we don't check for any issues happening during
			// the cleanup to avoid blocking the disruption deletion for nothing
			r.log.Infow("target is not likely to be cleaned (either it does not exist anymore or it is not ready), the injector will TRY to clean it but will not take care about any failures", "target", target)

			// by enabling this, we will remove the target associated chaos pods finalizers and delete them to trigger the cleanup phase
			// but the chaos pods status will not be checked
			ignoreStatus = true
		} else {
			r.log.Error(err.Error())

			return
		}
	}

	// It is always safe to remove some chaos pods. It is usually hard to tell if these chaos pods have
	// succeeded or not, but they have no possibility of leaving side effects, so we choose to always remove the finalizer.
	if chaosv1beta1.DisruptionHasNoSideEffects(chaosPod.Labels[chaostypes.DisruptionKindLabel]) {
		removeFinalizer = true
		ignoreStatus = true
	}

	// check the chaos pod status to determine if we can safely delete it or not
	switch chaosPod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodPending:
		// pod has terminated or is pending
		// we can remove the pod and the finalizer, so that it'll be garbage collected
		removeFinalizer = true
	case corev1.PodFailed:
		// pod has failed
		// we need to determine if we can remove it safely or if we need to block disruption deletion
		// check if a container has been created (if not, the disruption was not injected)
		if len(chaosPod.Status.ContainerStatuses) == 0 {
			removeFinalizer = true
		}

		// if the pod died only because it exceeded its activeDeadlineSeconds, we can remove the finalizer
		if chaosPod.Status.Reason == "DeadlineExceeded" {
			removeFinalizer = true
		}

		// check if the container was able to start or not
		// if not, we can safely delete the pod since the disruption was not injected
		for _, cs := range chaosPod.Status.ContainerStatuses {
			if cs.Name == "injector" {
				if cs.State.Terminated != nil && cs.State.Terminated.Reason == "StartError" {
					removeFinalizer = true
				}

				break
			}
		}
	default:
		if !ignoreStatus {
			// ignoring any pods not being in a "terminated" state
			// if the target is not healthy, we clean up this pod regardless of its state
			return
		}
	}

	// remove the finalizer if possible or if we can ignore the cleanup status
	if removeFinalizer || ignoreStatus {
		r.log.Infow("chaos pod completed, removing finalizer", "target", target, "chaosPod", chaosPod.Name)

		controllerutil.RemoveFinalizer(&chaosPod, chaostypes.ChaosPodFinalizer)

		if err := r.Client.Update(context.Background(), &chaosPod); err != nil {
			if strings.Contains(err.Error(), "latest version and try again") {
				r.log.Debugw("cannot remove chaos pod finalizer, need to re-reconcile", "error", err)
			} else {
				r.log.Errorw("error removing chaos pod finalizer", "error", err, "chaosPod", chaosPod.Name)
			}

			return
		}
	} else {
		// if the chaos pod finalizer must not be removed and the chaos pod must not be deleted
		// and the cleanup status must not be ignored, we are stuck and won't be able to remove the disruption
		r.log.Infow("instance seems stuck on removal for this target, please check manually", "target", target, "chaosPod", chaosPod.Name)
		r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionStuckOnRemoval, "")

		instance.Status.IsStuckOnRemoval = true

		r.updateTargetInjectionStatus(instance, chaosPod, chaostypes.DisruptionInjectionStatusIsStuckOnRemoval, *chaosPod.DeletionTimestamp)
	}
}

func (r *DisruptionReconciler) updateTargetInjectionStatus(instance *chaosv1beta1.Disruption, chaosPod corev1.Pod, status chaostypes.DisruptionInjectionStatus, since metav1.Time) {
	targetInjection := instance.Status.TargetInjections[chaosPod.Labels[chaostypes.TargetLabel]]
	targetInjection.InjectionStatus = status
	targetInjection.Since = since
	targetInjection.InjectorPodName = chaosPod.Name
	instance.Status.TargetInjections[chaosPod.Labels[chaostypes.TargetLabel]] = targetInjection
}

// selectTargets will select min(count, all matching targets) random targets (pods or nodes depending on the disruption level)
// from the targets matching the instance label selector
// targets will only be selected once per instance
// the chosen targets names will be reflected in the instance status
// subsequent calls to this function will always return the same targets as the first call
func (r *DisruptionReconciler) selectTargets(instance *chaosv1beta1.Disruption) error {
	if len(instance.Status.TargetInjections) != 0 && instance.Spec.StaticTargeting {
		return nil
	}

	r.log.Infow("selecting targets to inject disruption to", "selector", instance.Spec.Selector.String())

	// validate the given label selector to avoid any formatting issues due to special chars
	if instance.Spec.Selector != nil {
		if err := validateLabelSelector(instance.Spec.Selector.AsSelector()); err != nil {
			r.recordEventOnDisruption(instance, chaosv1beta1.EventInvalidDisruptionLabelSelector, err.Error())

			return err
		}
	}

	matchingTargets, totalAvailableTargetsCount, err := r.getSelectorMatchingTargets(instance)
	if err != nil {
		r.log.Errorw("error getting matching targets", "error", err)
	}

	instance.Status.RemoveDeadTargets(matchingTargets)

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	targetsCount, err := getScaledValueFromIntOrPercent(instance.Spec.Count, len(matchingTargets), true)
	if err != nil {
		targetsCount = instance.Spec.Count.IntValue()
	}

	// filter matching targets to only get eligible ones
	eligibleTargets, err := r.getEligibleTargets(instance, matchingTargets)
	if err != nil {
		r.log.Errorw("error getting eligible targets", "error", err)

		return fmt.Errorf("error getting eligible targets: %w", err)
	}

	instance.Status.DesiredTargetsCount = targetsCount
	// if the asked targets count is greater than the amount of found targets, we take all of them
	targetsCount = int(math.Min(float64(targetsCount), float64(len(instance.Status.TargetInjections)+len(eligibleTargets))))
	if targetsCount < 1 {
		r.log.Info("ignored targets has reached target count, skipping")

		// If no target was previously found from the selector we don't notify the user of any ignored target, as there is no target anyway
		if len(matchingTargets) > 0 {
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionNoMoreValidTargets, "")
		}

		return nil
	}

	// Current and Desired targets count
	cTargetsCount := len(instance.Status.TargetInjections)
	dTargetsCount := targetsCount

	if cTargetsCount < dTargetsCount {
		// not enough targets: pick more targets from eligibleTargets
		instance.Status.AddTargets(dTargetsCount-cTargetsCount, eligibleTargets)
	} else if cTargetsCount > dTargetsCount {
		// too many targets: remove random extra targets
		instance.Status.RemoveTargets(cTargetsCount - dTargetsCount)
	}

	r.log.Debugw("updating instance status with targets selected for injection")

	instance.Status.SelectedTargetsCount = len(instance.Status.TargetInjections)
	instance.Status.IgnoredTargetsCount = totalAvailableTargetsCount - targetsCount

	return r.Status().Update(context.Background(), instance)
}

// getMatchingTargets fetches all existing target fitting the disruption's selector
func (r *DisruptionReconciler) getSelectorMatchingTargets(instance *chaosv1beta1.Disruption) ([]string, int, error) {
	healthyMatchingTargets := []string{}
	totalAvailableTargetsCount := 0

	// select either pods or nodes depending on the disruption level
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		pods, totalCount, err := r.TargetSelector.GetMatchingPodsOverTotalPods(r.Client, instance)
		if err != nil {
			return nil, 0, fmt.Errorf("can't get pods matching the given label selector: %w", err)
		}

		for _, pod := range pods.Items {
			healthyMatchingTargets = append(healthyMatchingTargets, pod.Name)
		}

		totalAvailableTargetsCount = totalCount
	case chaostypes.DisruptionLevelNode:
		nodes, totalCount, err := r.TargetSelector.GetMatchingNodesOverTotalNodes(r.Client, instance)
		if err != nil {
			return nil, 0, fmt.Errorf("can't get nodes matching the given label selector: %w", err)
		}

		for _, node := range nodes.Items {
			healthyMatchingTargets = append(healthyMatchingTargets, node.Name)
		}

		totalAvailableTargetsCount = totalCount
	}

	// return an error if the selector returned no targets
	if len(healthyMatchingTargets) == 0 {
		r.log.Infow("the given label selector did not return any targets, skipping", "selector", instance.Spec.Selector)
		r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionNoTargetsFound, "")

		return nil, 0, nil
	}

	return healthyMatchingTargets, totalAvailableTargetsCount, nil
}

// deleteChaosPods deletes a chaos pod using the client
func (r *DisruptionReconciler) deleteChaosPod(instance *chaosv1beta1.Disruption, chaosPod corev1.Pod) {
	// delete the chaos pod only if it has not been deleted already
	if chaosPod.DeletionTimestamp.IsZero() {
		r.log.Infow("terminating chaos pod to trigger cleanup", "chaosPod", chaosPod.Name)

		if err := r.Client.Delete(context.Background(), &chaosPod); client.IgnoreNotFound(err) != nil {
			r.log.Errorw("error terminating chaos pod", "error", err, "chaosPod", chaosPod.Name)
		}

		r.handleChaosPodTermination(instance, chaosPod)
	}
}

// getEligibleTargets returns targets which can be targeted by the given instance from the given targets pool
// it skips ignored targets and targets being already targeted by another disruption
func (r *DisruptionReconciler) getEligibleTargets(instance *chaosv1beta1.Disruption, potentialTargets []string) (chaosv1beta1.TargetInjections, error) {
	r.log.Debug("getting eligible targets for disruption injection")

	eligibleTargets := chaosv1beta1.TargetInjections{}

	for _, target := range potentialTargets {
		// skip current targets
		if instance.Status.HasTarget(target) {
			continue
		}

		targetLabels := map[string]string{
			chaostypes.TargetLabel: target, // filter with target name
		}

		if instance.Spec.Level == chaostypes.DisruptionLevelPod { // nodes aren't namespaced and thus should only check by target name
			targetLabels[chaostypes.DisruptionNamespaceLabel] = instance.Namespace // filter with current instance namespace (to avoid getting pods having the same name but living in different namespaces)
		}

		// skip targets already targeted by a chaos pod from another disruption
		chaosPods, err := r.getChaosPods(nil, targetLabels)
		if err != nil {
			return nil, fmt.Errorf("error getting chaos pods targeting the given target (%s): %w", target, err)
		}

		if len(chaosPods) > 0 {
			r.log.Infow("target is already affected by another disruption, skipping", "target", target)

			continue
		}

		// add target if eligible
		eligibleTargets[target] = chaosv1beta1.TargetInjection{
			InjectionStatus: chaostypes.DisruptionInjectionStatusNotInjected,
		}
	}

	return eligibleTargets, nil
}

func (r *DisruptionReconciler) getChaosPods(instance *chaosv1beta1.Disruption, ls labels.Set) ([]corev1.Pod, error) {
	return chaosv1beta1.GetChaosPods(context.Background(), r.log, r.ChaosNamespace, r.Client, instance, ls)
}

// generatePod generates a pod from a generic pod template in the same namespace
// and on the same node as the given pod
func (r *DisruptionReconciler) generatePod(instance *chaosv1beta1.Disruption, targetName string, targetNodeName string, args []string, kind chaostypes.DisruptionKindName) *corev1.Pod {
	// volume host path type definitions
	hostPathDirectory := corev1.HostPathDirectory
	hostPathFile := corev1.HostPathFile

	// The default TerminationGracePeriodSeconds is 30s. This can be too low for a chaos pod to finish cleaning. After TGPS passes,
	// the signal sent to a pod becomes SIGKILL, which will interrupt any in-progress cleaning. By double this to 1 minute in the pod spec itself,
	// ensures that whether a chaos pod is deleted directly or by deleting a disruption, it will have time to finish cleaning up after itself.
	terminationGracePeriod := int64(60)
	// Chaos pods will clean themselves automatically when duration expires, so we set activeDeadlineSeconds to ten seconds after that
	// to give time for cleaning
	activeDeadlineSeconds := int64(calculateRemainingDuration(*instance).Seconds()) + 10
	args = append(args,
		"--deadline", time.Now().Add(calculateRemainingDuration(*instance)).Format(time.RFC3339))

	if activeDeadlineSeconds < 1 {
		return nil
	}

	podSpec := corev1.PodSpec{
		HostPID:                       true,                      // enable host pid
		RestartPolicy:                 corev1.RestartPolicyNever, // do not restart the pod on fail or completion
		NodeName:                      targetNodeName,            // specify node name to schedule the pod
		ServiceAccountName:            r.InjectorServiceAccount,  // service account to use
		TerminationGracePeriodSeconds: &terminationGracePeriod,
		ActiveDeadlineSeconds:         &activeDeadlineSeconds,
		Containers: []corev1.Container{
			{
				Name:            "injector",              // container name
				Image:           r.InjectorImage,         // container image gathered from controller flags
				ImagePullPolicy: corev1.PullIfNotPresent, // pull the image only when it is not present
				Args:            args,                    // pass disruption arguments
				SecurityContext: &corev1.SecurityContext{
					Privileged: func() *bool { b := true; return &b }(), // enable privileged mode
				},
				ReadinessProbe: &corev1.Probe{ // define readiness probe (file created by the injector when the injection is successful)
					PeriodSeconds:    1,
					FailureThreshold: 5,
					Handler: corev1.Handler{
						Exec: &corev1.ExecAction{
							Command: []string{"cat", "/tmp/readiness_probe"},
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
		},
	}

	if r.ImagePullSecrets != "" {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: r.ImagePullSecrets,
			},
		}
	}

	podLabels := make(map[string]string)
	for k, v := range r.InjectorLabels {
		podLabels[k] = v
	}

	podLabels[chaostypes.TargetLabel] = targetName                      // target name label
	podLabels[chaostypes.DisruptionKindLabel] = string(kind)            // disruption kind label
	podLabels[chaostypes.DisruptionNameLabel] = instance.Name           // disruption name label, used to determine ownership
	podLabels[chaostypes.DisruptionNamespaceLabel] = instance.Namespace // disruption namespace label, used to determine ownership

	// define injector pod
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("chaos-%s-", instance.Name), // generate the pod name automatically with a prefix
			Namespace:    r.ChaosNamespace,                        // chaos pods need to be in the same namespace as their service account to run
			Annotations:  r.InjectorAnnotations,                   // add extra annotations passed to the controller
			Labels:       podLabels,                               // add default and extra podLabels passed to the controller
		},
		Spec: podSpec,
	}

	// add finalizer to the pod so it is not deleted before we can control its exit status
	controllerutil.AddFinalizer(&pod, chaostypes.ChaosPodFinalizer)

	return &pod
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.log.Errorw("error sending a metric", "error", err)
	}
}

func (r *DisruptionReconciler) recordEventOnDisruption(instance *chaosv1beta1.Disruption, eventReason string, optionalMessage string) {
	disEvent := chaosv1beta1.Events[eventReason]
	message := disEvent.OnDisruptionTemplateMessage

	if optionalMessage != "" {
		message = fmt.Sprintf(disEvent.OnDisruptionTemplateMessage, optionalMessage)
	}

	r.Recorder.Event(instance, disEvent.Type, disEvent.Reason, message)
}

func (r *DisruptionReconciler) emitKindCountMetrics(instance *chaosv1beta1.Disruption) {
	for _, kind := range instance.Spec.GetKindNames() {
		r.handleMetricSinkError(r.MetricsSink.MetricDisruptionsCount(kind, []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
	}
}

func (r *DisruptionReconciler) validateDisruptionSpec(instance *chaosv1beta1.Disruption) error {
	err := instance.Spec.Validate()
	if err != nil {
		r.recordEventOnDisruption(instance, chaosv1beta1.EventInvalidSpecDisruption, err.Error())

		return err
	}

	return nil
}

// generateChaosPods generates a chaos pod for the given instance and disruption kind if set
func (r *DisruptionReconciler) generateChaosPods(instance *chaosv1beta1.Disruption, pods *[]*corev1.Pod, targetName string, targetNodeName string, targetContainers map[string]string, targetPodIP string) error {
	// generate chaos pods for each possible disruptions
	for _, kind := range chaostypes.DisruptionKindNames {
		subspec := instance.Spec.DisruptionKindPicker(kind)
		if reflect.ValueOf(subspec).IsNil() {
			continue
		}

		// default level to pod if not specified
		level := instance.Spec.Level
		if level == chaostypes.DisruptionLevelUnspecified {
			level = chaostypes.DisruptionLevelPod
		}

		pulseActiveDuration, pulseDormantDuration := time.Duration(0), time.Duration(0)
		if instance.Spec.Pulse != nil {
			pulseActiveDuration = instance.Spec.Pulse.ActiveDuration.Duration()
			pulseDormantDuration = instance.Spec.Pulse.DormantDuration.Duration()
		}

		allowedHosts := r.InjectorNetworkDisruptionAllowedHosts

		// get the ip ranges of cloud provider services
		if instance.Spec.Network != nil {
			if instance.Spec.Network.Cloud != nil {
				hosts, err := transformCloudSpecToHostsSpec(r.CloudServicesProvidersManager, instance.Spec.Network.Cloud)
				if err != nil {
					return err
				}

				instance.Spec.Network.Hosts = append(instance.Spec.Network.Hosts, hosts...)
			}

			// remove default allowed hosts if disabled
			if instance.Spec.Network.DisableDefaultAllowedHosts {
				allowedHosts = make([]string, 0)
			}
		}

		xargs := chaosapi.DisruptionArgs{
			Level:                level,
			Kind:                 kind,
			TargetContainers:     targetContainers,
			TargetName:           targetName,
			TargetNodeName:       targetNodeName,
			TargetPodIP:          targetPodIP,
			DryRun:               instance.Spec.DryRun,
			DisruptionName:       instance.Name,
			DisruptionNamespace:  instance.Namespace,
			OnInit:               instance.Spec.OnInit,
			PulseActiveDuration:  pulseActiveDuration,
			PulseDormantDuration: pulseDormantDuration,
			MetricsSink:          r.MetricsSink.GetSinkName(),
			AllowedHosts:         allowedHosts,
			DNSServer:            r.InjectorDNSDisruptionDNSServer,
			KubeDNS:              r.InjectorDNSDisruptionKubeDNS,
			ChaosNamespace:       r.ChaosNamespace,
		}

		// generate args for pod
		args := chaosapi.AppendArgs(subspec.GenerateArgs(), xargs)

		// append pod to chaos pods
		pod := r.generatePod(instance, targetName, targetNodeName, args, kind)
		if pod != nil {
			*pods = append(*pods, pod)
		}
	}

	return nil
}

// recordEventOnTarget records an event on the given target which can be either a pod or a node depending on the given disruption level
func (r *DisruptionReconciler) recordEventOnTarget(instance *chaosv1beta1.Disruption, target string, disruptionEventReason, chaosPod, optionalMessage string) {
	var o runtime.Object

	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelUnspecified, chaostypes.DisruptionLevelPod:
		p := &corev1.Pod{}

		if err := r.Get(context.Background(), types.NamespacedName{Namespace: instance.Namespace, Name: target}, p); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = p
	case chaostypes.DisruptionLevelNode:
		n := &corev1.Node{}

		if err := r.Get(context.Background(), types.NamespacedName{Name: target}, n); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = n
	}

	r.Recorder.Event(o, chaosv1beta1.Events[disruptionEventReason].Type, chaosv1beta1.Events[disruptionEventReason].Reason, fmt.Sprintf(chaosv1beta1.Events[disruptionEventReason].OnTargetTemplateMessage, chaosPod, optionalMessage))
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager, kubeInformerFactory kubeinformers.SharedInformerFactory) (controller.Controller, error) {
	podToDisruption := func(c client.Object) []reconcile.Request {
		// podtoDisruption is a function that maps pods to disruptions. it is meant to be used as an event handler on a pod informer
		// this function should safely return an empty list of requests to reconcile if the object we receive is not actually a chaos pod
		// which we determine by checking the object labels for the name and namespace labels that we add to all injector pods
		disruption := []reconcile.Request{}

		if r.log != nil {
			r.log.Debugw("watching event from pod", "podName", c.GetName(), "podNamespace", c.GetNamespace())
		}

		r.handleMetricSinkError(r.MetricsSink.MetricInformed([]string{"podName:" + c.GetName(), "podNamespace:" + c.GetNamespace()}))

		podLabels := c.GetLabels()
		name := podLabels[chaostypes.DisruptionNameLabel]
		namespace := podLabels[chaostypes.DisruptionNamespaceLabel]

		if name != "" && namespace != "" {
			disruption = append(disruption, reconcile.Request{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}})
		}

		return disruption
	}

	informer := kubeInformerFactory.Core().V1().Pods().Informer()

	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Second, time.Hour)}).
		Watches(&source.Informer{Informer: informer}, handler.EnqueueRequestsFromMapFunc(podToDisruption)).
		Build(r)
}

// ReportMetrics reports some controller metrics every minute:
// - stuck on removal disruptions count
// - ongoing disruptions count
func (r *DisruptionReconciler) ReportMetrics() {
	for {
		// wait for a minute
		<-time.After(time.Minute)

		// declare counters
		stuckOnRemoval := 0
		chaosPodsCount := 0

		l := chaosv1beta1.DisruptionList{}

		// list disruptions
		if err := r.Client.List(context.Background(), &l); err != nil {
			r.log.Errorw("error listing disruptions", "error", err)
			continue
		}

		// check for stuck durations, count chaos pods, and track ongoing disruption duration
		for _, d := range l.Items {
			if d.Status.IsStuckOnRemoval {
				stuckOnRemoval++

				if err := r.MetricsSink.MetricStuckOnRemoval([]string{"disruptionName:" + d.Name, "namespace:" + d.Namespace}); err != nil {
					r.log.Errorw("error sending stuck_on_removal metric", "error", err)
				}
			}

			chaosPods, err := r.getChaosPods(&d, nil)
			if err != nil {
				r.log.Errorw("error listing chaos pods to send pods.gauge metric", "error", err)
			}

			chaosPodsCount += len(chaosPods)

			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionOngoingDuration(time.Since(d.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + d.Name, "namespace:" + d.Namespace}))
		}

		// send metrics
		if err := r.MetricsSink.MetricStuckOnRemovalGauge(float64(stuckOnRemoval)); err != nil {
			r.log.Errorw("error sending stuck_on_removal_total metric", "error", err)
		}

		if err := r.MetricsSink.MetricDisruptionsGauge(float64(len(l.Items))); err != nil {
			r.log.Errorw("error sending disruptions.gauge metric", "error", err)
		}

		if err := r.MetricsSink.MetricPodsGauge(float64(chaosPodsCount)); err != nil {
			r.log.Errorw("error sending pods.gauge metric", "error", err)
		}

		if err := r.MetricsSink.MetricSelectorCacheGauge(float64(len(r.CacheContextStore))); err != nil {
			r.log.Errorw("error sending selector.cache.gauge metric", "error", err)
		}
	}
}

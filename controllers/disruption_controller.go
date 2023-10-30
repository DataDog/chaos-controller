// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions;disruptioncrons;disruptionrollouts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/status;disruptioncrons/status;disruptionrollouts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chaos.datadoghq.com,resources=disruptions/finalizers;disruptioncrons/finalizers;disruptionrollouts/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=list;watch

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"time"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/o11y/tracer"
	"github.com/DataDog/chaos-controller/safemode"
	"github.com/DataDog/chaos-controller/services"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/DataDog/chaos-controller/watchers"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// DisruptionReconciler reconciles a Disruption object
type DisruptionReconciler struct {
	Client                     client.Client
	BaseLog                    *zap.SugaredLogger
	Scheme                     *runtime.Scheme
	Recorder                   record.EventRecorder
	MetricsSink                metrics.Sink
	TracerSink                 tracer.Sink
	TargetSelector             targetselector.TargetSelector
	log                        *zap.SugaredLogger
	SafetyNets                 []safemode.Safemode
	ExpiredDisruptionGCDelay   *time.Duration
	CacheContextStore          map[string]CtxTuple
	DisruptionsWatchersManager watchers.DisruptionsWatchersManager
	ChaosPodService            services.ChaosPodService
	DisruptionsDeletionTimeout time.Duration
}

type CtxTuple struct {
	Ctx                      context.Context
	CancelFunc               context.CancelFunc
	DisruptionNamespacedName types.NamespacedName
}

func (r *DisruptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	instance := &chaosv1beta1.Disruption{}

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	// prepare logger instance context
	// NOTE: it is valid while we don't do concurrent reconciling
	// because the logger instance is pointer, concurrent reconciling would create a race condition
	// where the logger context would change for all ongoing reconcile loops
	// in the case we enable concurrent reconciling, we should create one logger instance per reconciling call
	r.log = r.BaseLog.With("disruptionName", req.Name, "disruptionNamespace", req.Namespace)

	// reconcile metrics
	r.handleMetricSinkError(r.MetricsSink.MetricReconcile())

	defer func(tsStart time.Time) {
		tags := []string{}
		if instance.Name != "" {
			tags = append(tags, "disruptionName:"+instance.Name, "namespace:"+instance.Namespace)
		}

		r.handleMetricSinkError(r.MetricsSink.MetricReconcileDuration(time.Since(tsStart), tags))
	}(time.Now())

	defer func() {
		panicInfo := recover()
		if panicInfo != nil {
			err = fmt.Errorf("a panic occurred during reconcile:\n\tpanic: %v\n\n\terror: %w", panicInfo, err)
		} else if err == nil {
			return
		}

		if chaosv1beta1.IsUpdateConflictError(err) {
			r.log.Infow("a retryable error occurred in reconcile loop", "error", err)
		} else {
			r.log.Errorw("an error occurred in reconcile loop", "error", err)
		}
	}()

	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// If we're reconciling but without an instance, then we must have been triggered by the pod informer
			// We should check for and delete any orphaned chaos pods
			err = r.ChaosPodService.HandleOrphanedChaosPods(ctx, req)
		}

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.DisruptionsWatchersManager.RemoveAllOrphanWatchers(); err != nil {
		r.log.Errorw("error during the deletion of orphan watchers", "error", err)
	}

	if err := r.DisruptionsWatchersManager.CreateAllWatchers(instance, nil, nil); err != nil {
		r.log.Errorw("error during the creation of watchers", "error", err)
	}

	ctx, err = instance.SpanContext(ctx)
	if err != nil {
		r.log.Errorw("did not find span context", "error", err)
	}

	userInfo, err := instance.UserInfo()
	if err != nil {
		r.log.Errorw("error getting user info", "error", err)

		userInfo.Username = "did-not-find-user-info@email.com"
	}

	ctx, reconcileSpan := otel.Tracer("").Start(ctx, "reconcile", trace.WithLinks(trace.LinkFromContext(ctx)),
		trace.WithAttributes(
			attribute.String("disruption_name", instance.Name),
			attribute.String("disruption_namespace", instance.Namespace),
			attribute.String("disruption_user", userInfo.Username),
		))
	defer reconcileSpan.End()

	// allows to sync logs with traces
	r.log = r.log.With(r.TracerSink.GetLoggableTraceContext(reconcileSpan)...)

	// handle any chaos pods being deleted (either by the disruption deletion or by an external event)
	if err := r.handleChaosPodsTermination(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("error handling chaos pods termination: %w", err)
	}

	// check whether the object is being deleted or not
	if !instance.DeletionTimestamp.IsZero() {
		// the instance is being deleted, clean it if the finalizer is still present
		if controllerutil.ContainsFinalizer(instance, chaostypes.DisruptionFinalizer) {
			// Check if the deletion time has expired for the 'instance' and it's not stuck on removal.
			if instance.IsDeletionExpired(r.DisruptionsDeletionTimeout) && !instance.Status.IsStuckOnRemoval {
				instance.Status.IsStuckOnRemoval = true

				r.log.Infow("instance seems stuck on removal, the deletion time expired, please check manually")

				// Update the status of the 'instance' to reflect that it's stuck on removal.
				if err := r.Client.Status().Update(ctx, instance); err != nil {
					return ctrl.Result{}, fmt.Errorf("error marking the disruption stuck on removal: %w", err)
				}

				r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionStuckOnRemoval, "", "")
			}

			isCleaned, err := r.cleanDisruption(ctx, instance)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error cleaning disruption: %w", err)
			}

			// if not cleaned yet, requeue and reconcile again in 15s-20s
			// the reason why we don't rely on the exponential backoff here is that it retries too fast at the beginning
			if !isCleaned {
				requeueAfter := time.Duration(randSource.Intn(5)+15) * time.Second //nolint:gosec

				r.log.Infow(fmt.Sprintf("disruption has not been fully cleaned yet, re-queuing in %v", requeueAfter))

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueAfter,
				}, r.Client.Update(ctx, instance)
			}

			// we reach this code when all the cleanup pods have succeeded
			// we can remove the finalizer and let the resource being garbage collected
			r.log.Infow("all chaos pods are cleaned up; removing disruption finalizer")
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionFinished, "", "")

			r.DisruptionsWatchersManager.RemoveAllWatchers(instance)
			controllerutil.RemoveFinalizer(instance, chaostypes.DisruptionFinalizer)

			if err := r.Client.Update(ctx, instance); err != nil {
				return ctrl.Result{}, fmt.Errorf("error removing disruption finalizer: %w", err)
			}

			// send reconciling duration metric
			r.handleMetricSinkError(r.MetricsSink.MetricCleanupDuration(time.Since(instance.ObjectMeta.DeletionTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionCompletedDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
			r.emitKindCountMetrics(instance)

			// close the ongoing disruption tracing Span
			defer func() {
				_, disruptionStopSpan := otel.Tracer("").Start(ctx, "disruption deletion", trace.WithAttributes(
					attribute.String("disruption_name", instance.Name),
					attribute.String("disruption_namespace", instance.Namespace),
					attribute.String("disruption_user", userInfo.Username),
				))

				disruptionStopSpan.End()
			}()

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
		if err := r.Client.Update(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("error adding disruption finalizer: %w", err)
		}

		// If the disruption is at least r.ExpiredDisruptionGCDelay older than when its duration ended, then we should delete it.
		// calculateRemainingDurationSeconds returns the seconds until (or since, if negative) the duration's deadline. We compare it to negative ExpiredDisruptionGCDelay,
		// and if less than that, it means we have exceeded the deadline by at least ExpiredDisruptionGCDelay, so we can delete
		if r.ExpiredDisruptionGCDelay != nil && (instance.RemainingDuration() <= (-1 * *r.ExpiredDisruptionGCDelay)) {
			r.log.Infow("disruption has lived for more than its duration, it will now be deleted.", "duration", instance.Spec.Duration)
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionGCOver, r.ExpiredDisruptionGCDelay.String(), "")

			var err error

			if err = r.Client.Delete(ctx, instance); err != nil {
				r.log.Errorw("error deleting disruption after its duration expired", "error", err)
			}

			return ctrl.Result{Requeue: true}, err
		} else if instance.RemainingDuration() <= 0 {
			if err := r.updateInjectionStatus(ctx, instance); err != nil {
				return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
			}

			if r.ExpiredDisruptionGCDelay != nil {
				requeueDelay := *r.ExpiredDisruptionGCDelay

				r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionDurationOver, requeueDelay.String(), "")
				r.log.Debugw("requeuing disruption to check for its expiration", "requeueDelay", requeueDelay.String())

				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: requeueDelay,
				}, nil
			}

			return ctrl.Result{Requeue: false}, nil
		}

		// check if we have reached trigger.createPods. If not, skip the rest of reconciliation.
		requeueAfter := time.Until(instance.TimeToCreatePods())
		if requeueAfter > (time.Second * 5) {
			requeueAfter -= (time.Second * 5)
			r.log.Debugw("requeuing disruption as we haven't yet reached trigger.createPods", "requeueAfter", requeueAfter.String())

			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		// retrieve targets from label selector
		if err := r.selectTargets(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("error selecting targets: %w", err)
		}

		// start injections
		if err := r.startInjection(ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("error creating chaos pods to start the disruption: %w", err)
		}

		// send injection duration metric representing the time it took to fully inject the disruption until its creation
		r.handleMetricSinkError(r.MetricsSink.MetricInjectDuration(time.Since(instance.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))

		// update resource status injection
		// requeue the request if the disruption is not fully notFullyInjected yet
		err := r.updateInjectionStatus(ctx, instance)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating disruption injection status: %w", err)
		} else if instance.Status.InjectionStatus.NotFullyInjected() {
			// requeue after 15-20 seconds, as default 1ms is too quick here
			requeueAfter := time.Duration(randSource.Intn(5)+15) * time.Second //nolint:gosec
			r.log.Infow("disruption is not fully injected yet, requeuing", "injectionStatus", instance.Status.InjectionStatus)

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: requeueAfter,
			}, nil
		}

		disruptionEndAt := instance.RemainingDuration() + time.Second

		r.log.Infow("requeuing disruption to check once expired", "requeueDelay", disruptionEndAt)

		return ctrl.Result{
				Requeue:      true,
				RequeueAfter: disruptionEndAt,
			},
			r.Client.Update(ctx, instance)
	}

	// stop the reconcile loop, there's nothing else to do
	return ctrl.Result{}, nil
}

// updateInjectionStatus updates the given instance injection status depending on its chaos pods statuses
// - an instance with all chaos pods "ready" is considered as "injected"
// - an instance with at least one chaos pod as "ready" is considered as "partially injected"
// - an instance with no ready chaos pods is considered as "not injected"
// - an instance expired will have previously defined status prefixed with "previously"
func (r *DisruptionReconciler) updateInjectionStatus(ctx context.Context, instance *chaosv1beta1.Disruption) (err error) {
	r.log.Debugw("checking if injection status needs to be updated", "injectionStatus", instance.Status.InjectionStatus)

	defer func() {
		r.log.Debugw("injection status updated to", "injectionStatus", instance.Status.InjectionStatus, "error", err)
	}()

	readyPodsCount := 0

	// get chaos pods
	chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		return fmt.Errorf("error getting instance chaos pods: %w", err)
	}

	status := instance.Status.InjectionStatus
	if status == chaostypes.DisruptionInjectionStatusInitial {
		status = chaostypes.DisruptionInjectionStatusNotInjected
	}

	terminationStatus := instance.TerminationStatus(chaosPods)
	if terminationStatus != chaosv1beta1.TSNotTerminated {
		switch status {
		case
			chaostypes.DisruptionInjectionStatusInjected,
			chaostypes.DisruptionInjectionStatusPausedInjected,
			chaostypes.DisruptionInjectionStatusPreviouslyInjected:
			status = chaostypes.DisruptionInjectionStatusPausedInjected
			if terminationStatus == chaosv1beta1.TSDefinitivelyTerminated {
				status = chaostypes.DisruptionInjectionStatusPreviouslyInjected
			}
		case
			chaostypes.DisruptionInjectionStatusPartiallyInjected,
			chaostypes.DisruptionInjectionStatusPausedPartiallyInjected,
			chaostypes.DisruptionInjectionStatusPreviouslyPartiallyInjected:
			status = chaostypes.DisruptionInjectionStatusPausedPartiallyInjected
			if terminationStatus == chaosv1beta1.TSDefinitivelyTerminated {
				status = chaostypes.DisruptionInjectionStatusPreviouslyPartiallyInjected
			}
		case
			chaostypes.DisruptionInjectionStatusNotInjected,
			chaostypes.DisruptionInjectionStatusPreviouslyNotInjected:
			// NB: we can't be PausedNotInjected, it's NotInjected
			status = chaostypes.DisruptionInjectionStatusNotInjected
			if terminationStatus == chaosv1beta1.TSDefinitivelyTerminated {
				status = chaostypes.DisruptionInjectionStatusPreviouslyNotInjected
			}
		default:
			return fmt.Errorf("unable to transition from disruption injection status %s, unknown injection status, termination status is %d", status, terminationStatus)
		}
	} else if len(chaosPods) > 0 {
		// consider the disruption "partially injected" if we found at least one pod
		status = chaostypes.DisruptionInjectionStatusPartiallyInjected

		injectorTargetsCount := map[string]struct{}{}

		// check the chaos pods conditions looking for the ready condition
		for _, chaosPod := range chaosPods {
			podReady := false

			// search for the "Ready" condition in the pod conditions
			for _, cond := range chaosPod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					if cond.Status == corev1.ConditionTrue {
						injectorTargetsCount[chaosPod.Labels[chaostypes.TargetLabel]] = struct{}{}
						podReady = true
						readyPodsCount++

						r.updateTargetInjectionStatus(instance, chaosPod, chaostypes.DisruptionTargetInjectionStatusInjected, cond.LastTransitionTime)

						break
					}
				}
			}

			// consider the disruption as not fully injected if at least one not ready pod is found
			if !podReady {
				r.log.Debugw("chaos pod is not ready yet", "chaosPod", chaosPod.Name)
			}
		}

		// consider the disruption as fully injected when all pods are ready and match desired targets count
		if instance.Status.DesiredTargetsCount == len(injectorTargetsCount) && !instance.Status.TargetInjections.NotFullyInjected() {
			status = chaostypes.DisruptionInjectionStatusInjected
		} else {
			r.log.Debugf("not injected yet because not all pods are ready %d/%d", len(injectorTargetsCount), instance.Status.DesiredTargetsCount)
		}
	}

	// update instance status
	r.log.Infof("from status %s to %s, terminationStatus is %d, readyPodCount is %d, desired targets count is %d", instance.Status.InjectionStatus, status, terminationStatus, readyPodsCount, instance.Status.DesiredTargetsCount)
	instance.Status.InjectionStatus = status

	// we divide by the number of active disruption types because we create one pod per target per disruption
	// ex: we would have 10 pods if we target 50% of all targets with 2 disruption types like network and dns
	// we also consider a target is not fully injected if not all disruptions are injected in it
	if instance.Spec.DisruptionCount() == 0 {
		instance.Status.InjectedTargetsCount = 0
	} else {
		instance.Status.InjectedTargetsCount = int(math.Floor(float64(readyPodsCount) / float64(instance.Spec.DisruptionCount())))
	}

	if err := r.Client.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("unable to update disruption injection status: %w", err)
	}

	return nil
}

// startInjection creates non-existing chaos pod for the given disruption
func (r *DisruptionReconciler) startInjection(ctx context.Context, instance *chaosv1beta1.Disruption) error {
	// chaosPodsMap is used to check if a target's chaos pods already exist or not
	chaosPodsMap := make(map[string]map[string]bool, len(instance.Status.TargetInjections))

	chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		return fmt.Errorf("error getting chaos pods: %w", err)
	}

	// init all the required maps
	for targetName := range instance.Status.TargetInjections {
		chaosPodsMap[targetName] = make(map[string]bool)
	}

	for _, chaosPod := range chaosPods {
		if !instance.Status.HasTarget(chaosPod.Labels[chaostypes.TargetLabel]) {
			r.deleteChaosPod(ctx, instance, chaosPod)
		} else {
			chaosPodsMap[chaosPod.Labels[chaostypes.TargetLabel]][chaosPod.Labels[chaostypes.DisruptionKindLabel]] = true
		}
	}

	if len(instance.Status.TargetInjections) > 0 && (len(instance.Status.TargetInjections) != len(chaosPodsMap)) {
		r.log.Infow("starting targets injection", "targets", instance.Status.TargetInjections)
	}

	// iterate through target + existing disruption kind -- to ensure all chaos pods exist
	for targetName, injections := range instance.Status.TargetInjections {
		for _, disKind := range chaostypes.DisruptionKindNames {
			if subspec := instance.Spec.DisruptionKindPicker(disKind); reflect.ValueOf(subspec).IsNil() {
				continue
			}

			if _, ok := chaosPodsMap[targetName][disKind.String()]; ok {
				continue
			}

			injection := injections.GetInjectionWithDisruptionKind(disKind)

			if injection == nil {
				return fmt.Errorf("the injection status from the target injections with this %s kind of disruption does not exist", disKind)
			}

			if chaosv1beta1.ShouldSkipNodeFailureInjection(disKind, instance, *injection) {
				r.log.Debugw("skipping over injection, seems to be a re-injected node failure", "targetName", targetName, "injectionStatus", injections)
				continue
			}

			if err = r.createChaosPods(ctx, instance, targetName); err != nil {
				if !apierrors.IsNotFound(err) {
					return fmt.Errorf("error creating chaos pods: %w", err)
				}

				r.log.Warnw("could not create chaos pod", "error", err)
			}

			break
		}
	}

	return nil
}

// createChaosPods attempts to create all the chaos pods for a given target. If a given chaos pod already exists, it is not recreated.
func (r *DisruptionReconciler) createChaosPods(ctx context.Context, instance *chaosv1beta1.Disruption, target string) error {
	var err error

	targetNodeName := ""
	targetContainers := map[string]string{}
	targetPodIP := ""

	// retrieve target
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelPod:
		pod := corev1.Pod{}

		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: target}, &pod); err != nil {
			return fmt.Errorf("error getting target to inject: %w", err)
		}

		targetNodeName = pod.Spec.NodeName

		// get IDs of targeted containers or all containers
		targetContainers, err = chaosv1beta1.TargetedContainers(pod, instance.Spec.Containers)
		if err != nil {
			dErr := fmt.Errorf("error getting target pod's container ID: %w", err)

			r.recordEventOnDisruption(instance, chaosv1beta1.EventInvalidSpecDisruption, dErr.Error(), pod.Name)

			return dErr
		}

		// get IP of targeted pod
		targetPodIP = pod.Status.PodIP
	case chaostypes.DisruptionLevelNode:
		targetNodeName = target
	}

	// generate injection pods specs
	targetChaosPods, err := r.ChaosPodService.GenerateChaosPodsOfDisruption(instance, target, targetNodeName, targetContainers, targetPodIP)
	if err != nil {
		return fmt.Errorf("error generating chaos pods: %w", err)
	}

	if len(targetChaosPods) == 0 {
		r.recordEventOnDisruption(instance, chaosv1beta1.EventEmptyDisruption, instance.Name, "")

		return nil
	}

	if instance.RemainingDuration().Seconds() < 1 {
		r.log.Debugw("skipping creation of chaos pods, remaining duration is too small", "remainingDuration", instance.RemainingDuration().String())

		return nil
	}

	// create injection pods
	for _, targetChaosPod := range targetChaosPods {
		// check if an injection pod already exists for the given (instance, namespace, disruption kind) tuple
		found, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, instance, targetChaosPod.Labels)
		if err != nil {
			return fmt.Errorf("error getting existing chaos pods: %w", err)
		}

		// create injection pods if none have been found
		switch len(found) {
		case 0:
			chaosPodArgs := r.ChaosPodService.GetPodInjectorArgs(targetChaosPod)
			r.log.Infow("creating chaos pod", "target", target, "chaosPodArgs", chaosPodArgs)

			// create the pod
			if err = r.ChaosPodService.CreatePod(ctx, &targetChaosPod); err != nil {
				r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionCreationFailed, instance.Name, target)
				r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, false))

				return fmt.Errorf("error creating chaos pod: %w", err)
			}

			// wait for the pod to be existing
			if err := r.ChaosPodService.WaitForPodCreation(ctx, targetChaosPod); err != nil {
				r.log.Errorw("error waiting for chaos pod to be created", "error", err, "chaosPod", targetChaosPod.Name, "target", target)

				continue
			}

			// send metrics and events
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionChaosPodCreated, instance.Name, target)
			r.recordEventOnTarget(ctx, instance, target, chaosv1beta1.EventDisrupted, targetChaosPod.Name, instance.Name)
			r.handleMetricSinkError(r.MetricsSink.MetricPodsCreated(target, instance.Name, instance.Namespace, true))
		case 1:
			r.log.Debugw("an injection pod is already existing for the selected target", "target", target, "chaosPod", found[0].Name)
		default:
			var chaosPodNames []string
			for _, pod := range found {
				chaosPodNames = append(chaosPodNames, pod.Name)
			}

			r.log.Errorw("multiple injection pods for one target found", "target", target, "chaosPods", strings.Join(chaosPodNames, ","), "chaosPodLabels", targetChaosPod.Labels)
		}
	}

	return nil
}

// cleanDisruption triggers the cleanup of the given instance
// for each existing chaos pod for the given instance, the function will delete the chaos pod to trigger its cleanup phase
// the function returns true when no more chaos pods are existing (meaning that it keeps returning false if some pods
// are deleted but still present)
func (r *DisruptionReconciler) cleanDisruption(ctx context.Context, instance *chaosv1beta1.Disruption) (bool, error) {
	cleaned := true

	// get already existing chaos pods for the given disruption
	chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, instance, nil)
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
		r.deleteChaosPod(ctx, instance, chaosPod)
	}

	return cleaned, nil
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
func (r *DisruptionReconciler) handleChaosPodsTermination(ctx context.Context, instance *chaosv1beta1.Disruption) error {
	// get already existing chaos pods for the given disruption
	chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, instance, nil)
	if err != nil {
		return err
	}

	if len(chaosPods) == 0 {
		return nil
	}

	for _, chaosPod := range chaosPods {
		r.handleChaosPodTermination(ctx, instance, chaosPod)
	}

	return r.Client.Status().Update(ctx, instance)
}

func (r *DisruptionReconciler) handleChaosPodTermination(ctx context.Context, instance *chaosv1beta1.Disruption, chaosPod corev1.Pod) {
	// ignore chaos pods not being deleted
	if chaosPod.DeletionTimestamp.IsZero() {
		return
	}

	isStuckOnRemoval, err := r.ChaosPodService.HandleChaosPodTermination(ctx, instance, &chaosPod)
	if err != nil {
		r.log.Errorw("could not handle the chaos pod termination", "error", err, "chaosPod", chaosPod.Name)

		return
	}

	if isStuckOnRemoval {
		target := chaosPod.Labels[chaostypes.TargetLabel]

		// if the chaos pod finalizer must not be removed and the chaos pod must not be deleted
		// and the cleanup status must not be ignored, we are stuck and won't be able to remove the disruption
		r.log.Infow("instance seems stuck on removal for this target, please check manually", "target", target, "chaosPod", chaosPod.Name)
		r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionStuckOnRemoval, "", target)

		instance.Status.IsStuckOnRemoval = true

		r.updateTargetInjectionStatus(instance, chaosPod, chaostypes.DisruptionTargetInjectionStatusStatusIsStuckOnRemoval, *chaosPod.DeletionTimestamp)
	}
}

func (r *DisruptionReconciler) updateTargetInjectionStatus(instance *chaosv1beta1.Disruption, chaosPod corev1.Pod, status chaostypes.DisruptionTargetInjectionStatus, since metav1.Time) {
	targetInjections := instance.Status.TargetInjections[chaosPod.Labels[chaostypes.TargetLabel]]

	disruptionKindName := chaostypes.DisruptionKindName(chaosPod.Labels[chaostypes.DisruptionKindLabel])

	targetInjection := targetInjections.GetInjectionWithDisruptionKind(disruptionKindName)
	targetInjection.InjectorPodName = chaosPod.Name
	targetInjection.InjectionStatus = status
	targetInjection.Since = since

	for i, ti := range targetInjections {
		if ti.DisruptionKindName != disruptionKindName.String() {
			continue
		}

		targetInjections[i] = *targetInjection

		return
	}

	instance.Status.TargetInjections[chaosPod.Labels[chaostypes.TargetLabel]] = targetInjections
}

// selectTargets will select min(count, all matching targets) random targets (pods or nodes depending on the disruption level)
// from the targets matching the instance label selector
// targets will only be selected once per instance
// the chosen targets names will be reflected in the instance status
// subsequent calls to this function will always return the same targets as the first call
func (r *DisruptionReconciler) selectTargets(ctx context.Context, instance *chaosv1beta1.Disruption) error {
	if len(instance.Status.TargetInjections) != 0 && instance.Spec.StaticTargeting {
		return nil
	}

	r.log.Infow("selecting targets to inject disruption to", "selector", instance.Spec.Selector.String())

	// validate the given label selector to avoid any formatting issues due to special chars
	if instance.Spec.Selector != nil {
		if err := targetselector.ValidateLabelSelector(instance.Spec.Selector.AsSelector()); err != nil {
			r.recordEventOnDisruption(instance, chaosv1beta1.EventInvalidDisruptionLabelSelector, err.Error(), "")

			return err
		}
	}

	matchingTargets, totalAvailableTargetsCount, err := r.getSelectorMatchingTargets(instance)
	if err != nil {
		r.log.Errorw("error getting matching targets", "error", err)
	}

	instance.Status.RemoveDeadTargets(matchingTargets)

	// instance.Spec.Count is a string that either represents a percentage or a value, we do the translation here
	targetsCount, err := instance.GetTargetsCountAsInt(len(matchingTargets), true)
	if err != nil {
		targetsCount = instance.Spec.Count.IntValue()
	}

	// filter matching targets to only get eligible ones
	eligibleTargets, err := r.getEligibleTargets(ctx, instance, matchingTargets)
	if err != nil {
		return fmt.Errorf("error getting eligible targets: %w", err)
	}

	instance.Status.DesiredTargetsCount = targetsCount
	// if the asked targets count is greater than the amount of found targets, we take all of them
	targetsCount = int(math.Min(float64(targetsCount), float64(len(instance.Status.TargetInjections)+len(eligibleTargets))))
	if targetsCount < 1 {
		r.log.Info("ignored targets has reached target count, skipping")

		// If no target was previously found from the selector we don't notify the user of any ignored target, as there is no target anyway
		if len(matchingTargets) > 0 {
			r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionNoMoreValidTargets, "", "")
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

	r.log.Debugw("updating instance status with targets selected for injection", "selectedTargets", instance.Status.TargetInjections.GetTargetNames())

	instance.Status.SelectedTargetsCount = len(instance.Status.TargetInjections)
	instance.Status.IgnoredTargetsCount = totalAvailableTargetsCount - targetsCount

	return r.Client.Status().Update(ctx, instance)
}

// getMatchingTargets fetches all existing target fitting the disruption's selector
func (r *DisruptionReconciler) getSelectorMatchingTargets(instance *chaosv1beta1.Disruption) ([]string, int, error) {
	healthyMatchingTargets := []string{}
	totalAvailableTargetsCount := 0

	// select either pods or nodes depending on the disruption level
	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelPod:
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
		r.recordEventOnDisruption(instance, chaosv1beta1.EventDisruptionNoTargetsFound, "", "")

		return nil, 0, nil
	}

	return healthyMatchingTargets, totalAvailableTargetsCount, nil
}

// deleteChaosPods deletes a chaos pod using the client
func (r *DisruptionReconciler) deleteChaosPod(ctx context.Context, instance *chaosv1beta1.Disruption, chaosPod corev1.Pod) {
	// delete the chaos pod only if it has not been deleted already
	if chaosPod.DeletionTimestamp.IsZero() {
		r.ChaosPodService.DeletePod(ctx, chaosPod)
		r.handleChaosPodTermination(ctx, instance, chaosPod)
	}
}

// handleMetricSinkError logs the given metric sink error if it is not nil
func (r *DisruptionReconciler) handleMetricSinkError(err error) {
	if err != nil {
		r.log.Errorw("error sending a metric", "error", err)
	}
}

func (r *DisruptionReconciler) recordEventOnDisruption(instance *chaosv1beta1.Disruption, eventReason chaosv1beta1.DisruptionEventReason, optionalMessage string, targetName string) {
	disEvent := chaosv1beta1.Events[eventReason]
	message := disEvent.OnDisruptionTemplateMessage

	if optionalMessage != "" {
		message = fmt.Sprintf(disEvent.OnDisruptionTemplateMessage, optionalMessage)
	}

	if targetName != "" {
		r.Recorder.AnnotatedEventf(instance, map[string]string{
			"target_name": targetName,
		}, disEvent.Type, string(disEvent.Reason), message)
	} else {
		r.Recorder.Event(instance, disEvent.Type, string(disEvent.Reason), message)
	}
}

func (r *DisruptionReconciler) emitKindCountMetrics(instance *chaosv1beta1.Disruption) {
	for _, kind := range instance.Spec.KindNames() {
		r.handleMetricSinkError(r.MetricsSink.MetricDisruptionsCount(kind, []string{"disruptionName:" + instance.Name, "namespace:" + instance.Namespace}))
	}
}

func (r *DisruptionReconciler) validateDisruptionSpec(instance *chaosv1beta1.Disruption) error {
	err := instance.Spec.Validate()
	if err != nil {
		r.recordEventOnDisruption(instance, chaosv1beta1.EventInvalidSpecDisruption, err.Error(), "")

		return err
	}

	return nil
}

// recordEventOnTarget records an event on the given target which can be either a pod or a node depending on the given disruption level
func (r *DisruptionReconciler) recordEventOnTarget(ctx context.Context, instance *chaosv1beta1.Disruption, target string, disruptionEventReason chaosv1beta1.DisruptionEventReason, chaosPod, optionalMessage string) {
	var o runtime.Object

	switch instance.Spec.Level {
	case chaostypes.DisruptionLevelPod:
		p := &corev1.Pod{}

		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: target}, p); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = p
	case chaostypes.DisruptionLevelNode:
		n := &corev1.Node{}

		if err := r.Client.Get(ctx, types.NamespacedName{Name: target}, n); err != nil {
			r.log.Errorw("event failed to be registered on target", "error", err, "target", target)
		}

		o = n
	}

	eventReason := chaosv1beta1.Events[disruptionEventReason]

	r.Recorder.Event(o, eventReason.Type, string(eventReason.Reason), fmt.Sprintf(eventReason.OnTargetTemplateMessage, chaosPod, optionalMessage))
}

// SetupWithManager setups the current reconciler with the given manager
func (r *DisruptionReconciler) SetupWithManager(mgr ctrl.Manager, kubeInformerFactory kubeinformers.SharedInformerFactory) (controller.Controller, error) {
	podToDisruption := func(c client.Object) []reconcile.Request {
		// podtoDisruption is a function that maps pods to disruptions. it is meant to be used as an event handler on a pod informer
		// this function should safely return an empty list of requests to reconcile if the object we receive is not actually a chaos pod
		// which we determine by checking the object labels for the name and namespace labels that we add to all injector pods
		if r.BaseLog != nil {
			r.BaseLog.Debugw("watching event from pod", "podName", c.GetName(), "podNamespace", c.GetNamespace())
		}

		r.handleMetricSinkError(r.MetricsSink.MetricInformed([]string{"podName:" + c.GetName(), "podNamespace:" + c.GetNamespace()}))

		podLabels := c.GetLabels()
		name := podLabels[chaostypes.DisruptionNameLabel]
		namespace := podLabels[chaostypes.DisruptionNamespaceLabel]

		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}}}
	}

	informer := kubeInformerFactory.Core().V1().Pods().Informer()

	return ctrl.NewControllerManagedBy(mgr).
		For(&chaosv1beta1.Disruption{}).
		WithOptions(controller.Options{RateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Second, time.Hour)}).
		Watches(&source.Informer{Informer: informer}, handler.EnqueueRequestsFromMapFunc(podToDisruption)).
		WithEventFilter(chaosEventsPredicate()).
		Build(r)
}

// chaosEventsPredicate determines if given event is a chaos related one or not
func chaosEventsPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return shouldTriggerReconcile(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return shouldTriggerReconcile(e.ObjectOld)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return shouldTriggerReconcile(e.Object)
		},
	}
}

// shouldTriggerReconcile determines whether the currently given object should trigger a reconcile or not
func shouldTriggerReconcile(o client.Object) bool {
	if _, ok := o.(*chaosv1beta1.Disruption); ok {
		return true
	}

	pod, ok := o.(*corev1.Pod)
	if !ok {
		return false
	}

	podLabels := pod.GetLabels()
	name := podLabels[chaostypes.DisruptionNameLabel]
	namespace := podLabels[chaostypes.DisruptionNamespaceLabel]

	return name != "" && namespace != ""
}

// ReportMetrics reports some controller metrics every minute:
// - stuck on removal disruptions count
// - ongoing disruptions count
func (r *DisruptionReconciler) ReportMetrics(ctx context.Context) {
	for {
		// wait for a minute
		<-time.After(time.Minute)

		// declare counters
		stuckOnRemoval := 0
		chaosPodsCount := 0

		l := chaosv1beta1.DisruptionList{}

		// list disruptions
		if err := r.Client.List(ctx, &l); err != nil {
			r.BaseLog.Errorw("error listing disruptions", "error", err)
			continue
		}

		// check for stuck durations, count chaos pods, and track ongoing disruption duration
		for _, d := range l.Items {
			if d.Status.IsStuckOnRemoval {
				stuckOnRemoval++

				if err := r.MetricsSink.MetricStuckOnRemoval([]string{"disruptionName:" + d.Name, "namespace:" + d.Namespace}); err != nil {
					r.BaseLog.Errorw("error sending stuck_on_removal metric", "error", err)
				}
			}

			chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, &d, nil)
			if err != nil {
				r.BaseLog.Errorw("error listing chaos pods to send pods.gauge metric", "error", err)
			}

			chaosPodsCount += len(chaosPods)

			r.handleMetricSinkError(r.MetricsSink.MetricDisruptionOngoingDuration(time.Since(d.ObjectMeta.CreationTimestamp.Time), []string{"disruptionName:" + d.Name, "namespace:" + d.Namespace}))
		}

		// send metrics
		if err := r.MetricsSink.MetricStuckOnRemovalGauge(float64(stuckOnRemoval)); err != nil {
			r.BaseLog.Errorw("error sending stuck_on_removal_total metric", "error", err)
		}

		if err := r.MetricsSink.MetricDisruptionsGauge(float64(len(l.Items))); err != nil {
			r.BaseLog.Errorw("error sending disruptions.gauge metric", "error", err)
		}

		if err := r.MetricsSink.MetricPodsGauge(float64(chaosPodsCount)); err != nil {
			r.BaseLog.Errorw("error sending pods.gauge metric", "error", err)
		}

		if err := r.MetricsSink.MetricSelectorCacheGauge(float64(len(r.CacheContextStore))); err != nil {
			r.BaseLog.Errorw("error sending selector.cache.gauge metric", "error", err)
		}
	}
}

// getEligibleTargets returns targets which can be targeted by the given instance from the given targets pool
// it skips ignored targets and targets being already targeted by another disruption
func (r *DisruptionReconciler) getEligibleTargets(ctx context.Context, instance *chaosv1beta1.Disruption, potentialTargets []string) (eligibleTargets chaosv1beta1.TargetInjections, err error) {
	defer func() {
		r.log.Debugw("getting eligible targets for disruption injection", "potentialTargets", potentialTargets, "eligibleTargets", eligibleTargets, "error", err)
	}()

	eligibleTargets = make(chaosv1beta1.TargetInjections)

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

		chaosPods, err := r.ChaosPodService.GetChaosPodsOfDisruption(ctx, nil, targetLabels)
		if err != nil {
			return nil, fmt.Errorf("error getting chaos pods targeting the given target (%s): %w", target, err)
		}

		// skip targets already targeted by a chaos pod from another disruption with the same kind if any
		if len(chaosPods) != 0 {
			if !instance.Spec.AllowDisruptedTargets {
				r.log.Infow(`disruption spec does not allow to use already disrupted targets with ANY kind of existing disruption, skipping...
NB: you can specify "spec.allowDisruptedTargets: true" to allow a new disruption without any disruption kind intersection to target the same pod`, "target", target, "targetLabels", targetLabels)

				continue
			}

			targetDisruptedByKinds := map[chaostypes.DisruptionKindName]string{}
			for _, chaosPod := range chaosPods {
				targetDisruptedByKinds[chaostypes.DisruptionKindName(chaosPod.Labels[chaostypes.DisruptionKindLabel])] = chaosPod.Name
			}

			intersectionOfKinds := []string{}

			for _, kind := range instance.Spec.KindNames() {
				if chaosPodName, ok := targetDisruptedByKinds[kind]; ok {
					intersectionOfKinds = append(intersectionOfKinds, fmt.Sprintf("kind:%s applied by chaos-pod:%s", kind, chaosPodName))
				}
			}

			if len(intersectionOfKinds) != 0 {
				r.log.Infow("target is already disrupted by at least one provided kind, skipping", "target", target, "targetLabels", targetLabels, "targetDisruptedByKinds", targetDisruptedByKinds, "intersectionOfKinds", intersectionOfKinds)

				continue
			}
		}

		// add target if eligible for each disruption kind of the disruption
		for _, disKind := range chaostypes.DisruptionKindNames {
			if subspec := instance.Spec.DisruptionKindPicker(disKind); reflect.ValueOf(subspec).IsNil() {
				continue
			}

			eligibleTargets[target] = append(eligibleTargets[target], chaosv1beta1.TargetInjection{
				InjectionStatus:    chaostypes.DisruptionTargetInjectionStatusNotInjected,
				DisruptionKindName: disKind.String(),
			})
		}
	}

	return eligibleTargets, nil
}

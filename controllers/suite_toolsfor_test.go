// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package controllers

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	disruptionPotentialChangesEvery = time.Second
)

type lightConfig struct {
	Controller struct {
		ExpiredDisruptionGCDelay time.Duration `yaml:"expiredDisruptionGCDelay"`
		DefaultDuration          time.Duration `yaml:"defaultDuration"`
	} `yaml:"controller"`
}

// type Concurrently []func(Gomega, SpecContext) error
type Concurrently []func(SpecContext)

// DoAndWait will spawn a goroutine for all items in current array
// it will wait for all goroutine to complete before exiting
func (c Concurrently) DoAndWait(ctx SpecContext) {
	GinkgoHelper()

	sync := make(chan struct{}, len(c))

	for _, do := range c {
		do := do

		go func(ctx SpecContext) {
			defer func() {
				sync <- struct{}{}
			}()

			defer GinkgoRecover()

			GinkgoHelper()

			do(ctx)
		}(ctx)
	}

	for i := 0; i < cap(sync); i++ {
		<-sync
	}
}

// calcDisruptionGoneTimeout returns the guaranteed duration after which the disruption should no longer exists and has been garbage collected
func calcDisruptionGoneTimeout(disruption chaosv1beta1.Disruption) time.Duration {
	GinkgoHelper()

	disruptionDuration := lightCfg.Controller.DefaultDuration
	if disruption.Spec.Duration.Duration() != 0 {
		disruptionDuration = disruption.Spec.Duration.Duration()
	}

	// if a disruption has already been created, we can reduce the timeout to wait less time
	if !disruption.CreationTimestamp.IsZero() {
		if disruption.Spec.Duration.Duration() == 0 {
			Fail("an existing disruption should have a non-zero duration")
		}

		if remainingDisruptionDuration := calculateRemainingDuration(disruption); remainingDisruptionDuration > 0 {
			disruptionDuration = remainingDisruptionDuration
		}
	}

	disruptionGoneDuration := disruptionDuration + lightCfg.Controller.ExpiredDisruptionGCDelay + 5*time.Second

	AddReportEntry(fmt.Sprintf("disruption %s will be gone at %v (in %v)", disruption.Name, time.Now().Add(disruptionGoneDuration), disruptionGoneDuration))

	return disruptionGoneDuration
}

// allContainersAreRunning ensure not only provided pods are in running phase
// but also all their containers have a non nil container status state running
func allContainersAreRunning(ctx SpecContext, pods ...corev1.Pod) bool {
	// check the pod containers statuses (pod phase can be running while all containers are not running)
	// we return false if at least one container in the pod is not running
	for _, p := range pods {
		if p.Status.Phase != corev1.PodRunning {
			return false
		}

		for _, status := range p.Status.ContainerStatuses {
			if status.State.Running == nil {
				return false
			}
		}
	}

	return true
}

// podsInPhase check all pods and returns the live ones that have provided phase
func podsInPhase(ctx SpecContext, phase corev1.PodPhase, pods ...corev1.Pod) ([]corev1.Pod, error) {
	podsInPhase := []corev1.Pod{}

	for _, pod := range pods {
		var p corev1.Pod

		// retrieve pod
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &p); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}

			return nil, fmt.Errorf("unable to get pod: %w", err)
		}

		// check the pod phase
		if p.Status.Phase == phase {
			podsInPhase = append(podsInPhase, p)
		}
	}

	return podsInPhase, nil
}

// ExpectChaosPods not only check for chaos pod existence
// but also for their "needed" fields existence
func ExpectChaosPods(ctx SpecContext, disruption chaosv1beta1.Disruption, count int) {
	GinkgoHelper()

	Eventually(expectChaosPod).
		WithContext(ctx).WithArguments(disruption, count).
		Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).
		Should(Succeed())
}

// InjectPodsAndDisruption create 2 pods and a disruption targeting them
// it will also delete those pods and disruption on test cleanup
func InjectPodsAndDisruption(ctx SpecContext, wantDisruption chaosv1beta1.Disruption, skipSecondPod bool) (disruption chaosv1beta1.Disruption, targetPod, anotherTargetPod corev1.Pod) {
	GinkgoHelper()

	var anotherTargetPodCreated <-chan corev1.Pod
	targetPod = uniquePod()
	targetPod.Spec.RestartPolicy = corev1.RestartPolicyAlways

	if !skipSecondPod {
		anotherTargetPod = *targetPod.DeepCopy()
		// we want some small differences from the previous pod, a single container, and other annotation value for foo
		anotherTargetPod.Annotations["foo"] = "qux"
		anotherTargetPod.Spec.Containers = anotherTargetPod.Spec.Containers[0:1]
		anotherTargetPod.Spec.Volumes = anotherTargetPod.Spec.Volumes[0:1] // second volume is used by second container, hence not needed as we removed it
		anotherTargetPodCreated = CreateRunningPod(ctx, anotherTargetPod)
	} else {
		// if you ask to NOT create the second pod, you will NEVER receive a message, don't wait for it
		anotherTargetPodCreated = func() <-chan corev1.Pod {
			return nil
		}()
	}

	targetPod = <-CreateRunningPod(ctx, targetPod)
	// if we can wait for anotherTargetPod we do
	// they are still created in parallel
	// hence waiting time is reduced
	if anotherTargetPodCreated != nil {
		anotherTargetPod = <-anotherTargetPodCreated
	}

	AddReportEntry("both pods created and running with labels", targetPod.Labels)

	disruption = CreateDisruption(ctx, wantDisruption, targetPod)
	return
}

func CreateDisruption(ctx SpecContext, disruption chaosv1beta1.Disruption, targetPod corev1.Pod) chaosv1beta1.Disruption {
	disruption.ResourceVersion = ""

	if disruption.Name == "" {
		disruption.Name = targetPod.Name
	}

	if disruption.Spec.Selector == nil {
		disruption.Spec.Selector = targetPod.Labels
	}

	// this is required when running the controller locally as we do not benefit from webhook defaults
	if disruption.Spec.Duration == "" {
		disruption.Spec.Duration = chaosv1beta1.DisruptionDuration(lightCfg.Controller.DefaultDuration.String())
	}

	Eventually(func(ctx SpecContext) error {
		return StopTryingNotRetryableKubernetesError(k8sClient.Create(ctx, &disruption), true, false)
	}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed())

	AddReportEntry(fmt.Sprintf("disruption %s created at %v", disruption.Name, disruption.CreationTimestamp.Time), disruption)

	DeferCleanup(DeleteDisruption, disruption)

	return disruption
}

// ExpectDisruptionStatus wait until provided disruption has provided status
// if skipTargetStatus is provided, it will NOT check that injection target has the provided status
// most useful case to call this method is to expect a disruption (and all targeted containers) is Injected
func ExpectDisruptionStatus(ctx SpecContext, disruption chaosv1beta1.Disruption, statuses ...chaostypes.DisruptionInjectionStatus) {
	GinkgoHelper()

	AddReportEntry("Waiting for disruption to have one of statuses", statuses)

	Eventually(func(ctx SpecContext) error {
		// retrieve the previously created disruption
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: disruption.Namespace,
			Name:      disruption.Name,
		}, &disruption); err != nil {
			return StopTryingNotRetryableKubernetesError(err, false, false)
		}

		AddReportEntry(fmt.Sprintf("disruption %s has injection status %v", disruption.Name, disruption.Status.InjectionStatus))

		disruptionStatus := false
		for _, status := range statuses {
			disruptionStatus = disruptionStatus || disruption.Status.InjectionStatus == status

			// We check the targets status only if it's "convertible" from a disruption status
			// if it is, it should be valid
			switch status {
			case chaostypes.DisruptionInjectionStatusInjected, chaostypes.DisruptionInjectionStatusNotInjected:
				targetInjectionStatus := chaostypes.DisruptionTargetInjectionStatus(status)

				AddReportEntry(fmt.Sprintf("checking chaos pods have status %s", targetInjectionStatus))

				// check targets injection
				for targetName, target := range disruption.Status.TargetInjections {
					// check status
					if target.InjectionStatus != targetInjectionStatus {
						return fmt.Errorf("target injection %s is not injected, current status is %s", targetName, target.InjectionStatus)
					}

					// check if the chaos pod is defined
					if target.InjectorPodName == "" {
						return fmt.Errorf("the %s target pod does not have an injector pod ", targetName)
					}
				}
			}
		}

		if !disruptionStatus {
			return fmt.Errorf("disruptions does not has expected status yet, retrying, current status is %s, allowed statuses are %+v", disruption.Status.InjectionStatus, statuses)
		}

		return nil
	}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
}

// ExpectDisruptionStatusUntilExpired consistently expect provided status until disruption is gone
func ExpectDisruptionStatusUntilExpired(ctx SpecContext, disruption chaosv1beta1.Disruption, status chaostypes.DisruptionInjectionStatus) {
	GinkgoHelper()

	AddReportEntry("Checking disruption has always status", status)

	Consistently(func(ctx SpecContext) error {
		// retrieve the previously created disruption
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: disruption.Namespace,
			Name:      disruption.Name,
		}, &disruption); err != nil {
			return StopTryingNotRetryableKubernetesError(err, false, true)
		}

		if disruption.Status.InjectionStatus != status {
			// when requested status is NOT a previsouly status, we should stop looking when transition occured and new status is a previously one
			if !status.Previously() && disruption.Status.InjectionStatus.Previously() {
				return nil
			}

			return fmt.Errorf("unexpected disruption status %s, expect %s", disruption.Status.InjectionStatus, status)
		}

		return nil
	}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
}

// DeleteDisruption safely delete a disruption and does not fail if the disruption does not exists
// it also expect disruption side effects to NOT be there (associated pods should no longer exists)
func DeleteDisruption(ctx SpecContext, disruption chaosv1beta1.Disruption) {
	GinkgoHelper()

	// We might need to delete the pod, or not
	// it's not the system under test, hence no need to expect to succeed here
	Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&disruption).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()))
	ExpectChaosPods(ctx, disruption, 0)
	Eventually(k8sClient.Get).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).WithArguments(types.NamespacedName{
		Namespace: disruption.Namespace,
		Name:      disruption.Name,
	}, &chaosv1beta1.Disruption{}).Should(WithTransform(apierrors.IsNotFound, BeTrue()))
}

// CreateRunningPod create a pod and wait until it status is in phase Running
// created pod will be automatically deleted on test cleanup
func CreateRunningPod(ctx SpecContext, pod corev1.Pod) <-chan corev1.Pod {
	GinkgoHelper()

	pod.ResourceVersion = ""
	if !pod.CreationTimestamp.IsZero() {
		// once a pod has been created, Name will be set
		// we mostly use DeepCopy to easily create similar pod and run them, however we want them to follow the initial generate name pattern
		// hence we unset the name
		// in case you still want to define you own name from a deepCopy, unset CreationTimestamp
		pod.Name = ""
	}

	internalNotifyPodCanBeDeleted := make(chan corev1.Pod, 1)
	externalNotifyPodIsRunning := make(chan corev1.Pod, 1)

	go func() {
		defer GinkgoRecover()

		GinkgoHelper()

		Eventually(k8sClient.Create).WithArguments(&pod).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed())
		internalNotifyPodCanBeDeleted <- pod

		Eventually(func(ctx SpecContext) error {
			runningPods, err := podsInPhase(ctx, corev1.PodRunning, pod)
			if err != nil {
				return fmt.Errorf("unable to check pods in phase: %w", err)
			}

			if len(runningPods) == 0 || allContainersAreRunning(ctx, runningPods...) {
				return nil
			}

			return fmt.Errorf("some containers are not running in pods")
		}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed())

		AddReportEntry(fmt.Sprintf("pod %s created", pod.Name))

		externalNotifyPodIsRunning <- pod
	}()

	DeferCleanup(
		func(ctx SpecContext) {
			// The cleanup is called
			// we try to read the buffered channel to delete the pod if already created
			// we don't wait for the message to be received and SKIP the cleanup if the pod NOT already created
			// we rely on context cancellation (and namespace deletion) to guarantee cleanup is done as expected in such case
			select {
			case pod := <-internalNotifyPodCanBeDeleted:
				DeleteRunningPod(ctx, pod)
			default:
			}
		},
		NodeTimeout(k8sAPIServerResponseTimeout))

	return externalNotifyPodIsRunning
}

// DeleteRunningPod delete a pod safely and ignore if pod does not exists
// it wait until pod is entirely gone
func DeleteRunningPod(ctx SpecContext, pod corev1.Pod) {
	GinkgoHelper()

	Eventually(k8sClient.Delete).WithContext(ctx).WithArguments(&pod).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(WithTransform(client.IgnoreNotFound, Succeed()))

	Eventually(func(ctx SpecContext) error {
		runningPods, err := podsInPhase(ctx, corev1.PodRunning, pod)
		if err != nil {
			AddReportEntry("an error occurred while checking for pod in phase", err)

			return err
		}

		if len(runningPods) != 0 {
			podNames := make([]string, 0, len(runningPods))
			for _, pod := range runningPods {
				podNames = append(podNames, pod.Name)
			}

			return fmt.Errorf("some pods are still running: %v", podNames)
		}

		return nil
	}).WithContext(ctx).Within(k8sAPIServerResponseTimeout).ProbeEvery(k8sAPIPotentialChangesEvery).Should(Succeed())
}

// PickFirstChaodPod waits until a chaos pod is available and return the first one that is disregarding it's status (running, pending, ...)
func PickFirstChaodPod(ctx SpecContext, disruption chaosv1beta1.Disruption) corev1.Pod {
	var firstPod corev1.Pod

	Eventually(func(g Gomega, ctx SpecContext) {
		chaosPods, err := listChaosPods(ctx, disruption)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(chaosPods.Items).ToNot(BeEmpty())

		firstPod = chaosPods.Items[0]
	}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())

	return firstPod
}

// ExpectChaosPodToDisappear expect provided namespacedName is associated to a pod that should disappear when disruption ended
func ExpectChaosPodToDisappear(ctx SpecContext, chaosPodKey types.NamespacedName, disruption chaosv1beta1.Disruption) {
	Eventually(k8sClient.Get).
		WithContext(ctx).WithArguments(chaosPodKey, &corev1.Pod{}).
		Within(calculateRemainingDuration(disruption)).ProbeEvery(disruptionPotentialChangesEvery).
		Should(WithTransform(apierrors.IsNotFound, BeTrue()))
}

func uniquePod() corev1.Pod {
	uuid := uuid.NewString()

	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: uuid + "-", // in case we copy the pod, we will generate a new one simply still with the same label selector
			Namespace:    namespace,
			Labels: map[string]string{
				"foo": "bar",
				"uid": uuid,
			},
			Annotations: map[string]string{
				"foo": "baz",
				"uid": uuid,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: "k8s.gcr.io/pause:3.4.1",
					Name:  "ctn1",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "foov",
							MountPath: "/mnt/foo",
						},
					},
				},
				{
					Image: "k8s.gcr.io/pause:3.4.1",
					Name:  "ctn2",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "barv",
							MountPath: "/mnt/bar",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "foov",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "barv",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

// listChaosPods returns all the chaos pods for the given instance
func listChaosPods(ctx SpecContext, disruption chaosv1beta1.Disruption) (corev1.PodList, error) {
	l := corev1.PodList{}
	ls := labels.NewSelector()

	// create requirements
	disruptionNameRequirement, _ := labels.NewRequirement(chaostypes.DisruptionNameLabel, selection.Equals, []string{disruption.Name})
	disruptionNamespaceRequirement, _ := labels.NewRequirement(chaostypes.DisruptionNamespaceLabel, selection.Equals, []string{disruption.Namespace})

	// add requirements to label selector
	ls = ls.Add(*disruptionNamespaceRequirement, *disruptionNameRequirement)

	// get matching pods
	if err := k8sClient.List(ctx, &l, &client.ListOptions{
		LabelSelector: ls,
	}); err != nil {
		return corev1.PodList{}, fmt.Errorf("can't list chaos pods: %w", err)
	}

	return l, nil
}

// expectChaosPod retrieves the list of created chaos pods related to the given and to the
// given mode (inject or clean) and returns an error if it doesn't
// equal the given count
func expectChaosPod(ctx SpecContext, disruption chaosv1beta1.Disruption, count int) error {
	GinkgoHelper()

	l, err := listChaosPods(ctx, disruption)
	if err != nil {
		return fmt.Errorf("an error occured while retrieving chaos pods: %w", err)
	}

	AddReportEntry(fmt.Sprintf("chaos pods count: %d/%d", len(l.Items), count))

	if len(l.Items) != count {
		return fmt.Errorf("chaos pods count is not equal to expected count: %d != %d", len(l.Items), count)
	}

	// ensure generated pods have the needed fields
	for _, p := range l.Items {
		if p.GenerateName == "" {
			return StopTrying("GenerateName field can't be empty")
		}
		if len(p.Spec.Containers[0].Args) == 0 {
			return StopTrying("pod container args must be set")
		}
		if p.Spec.Containers[0].Image == "" {
			return StopTrying("pod container image must be set")
		}
		if len(p.ObjectMeta.Finalizers) == 0 {
			return StopTrying("pod finalizer must be set")
		}

		// ensure pod container is running (not completed or failed)
		if !allContainersAreRunning(ctx, p) {
			return fmt.Errorf("at least one of pod containers is not running")
		}
	}

	return nil
}

// ExpectChaosInjectors retrieves the list of created chaos pods and confirms
// that the targeted containers are present
func ExpectChaosInjectors(ctx SpecContext, disruption chaosv1beta1.Disruption, count int) {
	GinkgoHelper()

	Eventually(func(ctx SpecContext) error {
		injectors := 0

		// get chaos pods
		l, err := listChaosPods(ctx, disruption)
		if err != nil {
			return err
		}

		// sum up injectors
		for _, p := range l.Items {
			args := p.Spec.Containers[0].Args
			for i, arg := range args {
				if arg == "--target-containers" {
					injectors += len(strings.Split(args[i+1], ","))
				}
			}
		}

		AddReportEntry(fmt.Sprintf("chaos pods injectors count: %d/%d", injectors, count))

		if injectors != count {
			return fmt.Errorf("incorrect number of targeted containers in spec: expected %d, found %d", count, injectors)
		}

		return nil
	}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
}

func ExpectDisruptionStatusCounts(ctx SpecContext, disruption chaosv1beta1.Disruption, desiredTargetsCount, ignoredTargetsCount, selectedTargetsCount, injectedTargetsCount int) {
	GinkgoHelper()

	Eventually(func(ctx SpecContext) error {
		updatedInstance := &chaosv1beta1.Disruption{}

		if err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: disruption.Namespace,
			Name:      disruption.Name,
		}, updatedInstance); err != nil {
			return err
		}

		if desiredTargetsCount != updatedInstance.Status.DesiredTargetsCount {
			return fmt.Errorf("incorrect number of desired targets: expected %d, found %d", desiredTargetsCount, updatedInstance.Status.DesiredTargetsCount)
		}
		if ignoredTargetsCount != updatedInstance.Status.IgnoredTargetsCount {
			return fmt.Errorf("incorrect number of ignored targets: expected %d, found %d", ignoredTargetsCount, updatedInstance.Status.IgnoredTargetsCount)
		}
		if injectedTargetsCount != updatedInstance.Status.InjectedTargetsCount {
			return fmt.Errorf("incorrect number of injected targets: expected %d, found %d", injectedTargetsCount, updatedInstance.Status.InjectedTargetsCount)
		}
		if selectedTargetsCount != updatedInstance.Status.SelectedTargetsCount {
			return fmt.Errorf("incorrect number of selected targets: expected %d, found %d", selectedTargetsCount, updatedInstance.Status.SelectedTargetsCount)
		}

		return nil
	}).WithContext(ctx).Within(calcDisruptionGoneTimeout(disruption)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
}

func StopTryingNotRetryableKubernetesError(err error, okExists, okNotFound bool) error {
	if okExists && apierrors.IsAlreadyExists(err) ||
		okNotFound && apierrors.IsNotFound(err) {
		return nil
	}

	if apierrors.IsBadRequest(err) || apierrors.IsInvalid(err) {
		return StopTrying("invalid or bad request").Wrap(err)
	}

	if apierrors.IsForbidden(err) {
		return StopTrying("forbidden").Wrap(err)
	}

	return err
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package builderstest_test

import (
	"time"

	"github.com/DataDog/chaos-controller/env"
	"github.com/DataDog/chaos-controller/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChaosPodBuilder is a struct used to build a chaos pod instance.
type ChaosPodBuilder struct {
	*v1.Pod
	// we store action we want to perform instead of performing them right away because they are time sensitive
	// this enables us to ensure time.Now is as late as it can be without faking it (that we should do at some point)
	modifiers []func()
}

// NewPodBuilder creates a new ChaosPodBuilder instance with initial pod configuration.
func NewPodBuilder(podName, namespace string) *ChaosPodBuilder {
	return (&ChaosPodBuilder{
		Pod: &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              podName,
				Namespace:         namespace,
				CreationTimestamp: metav1.NewTime(time.Now()),
				Labels: map[string]string{
					"app": podName,
				},
			},
		},
	}).WithCreation(30 * time.Second)
}

// Build generates a v1.Pod instance based on the configuration.
func (b *ChaosPodBuilder) Build() v1.Pod {
	for _, modifier := range b.modifiers {
		modifier()
	}

	return *b.Pod
}

// Reset resets the ChaosPodBuilder by clearing all modifiers.
func (b *ChaosPodBuilder) Reset() *ChaosPodBuilder {
	b.modifiers = nil

	return b
}

// WithCreation adjusts the creation timestamp.
func (b *ChaosPodBuilder) WithCreation(past time.Duration) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.CreationTimestamp = metav1.NewTime(time.Now().Add(-past))
		})

	return b
}

// WithDeletion sets the deletion timestamp to the current time.
func (b *ChaosPodBuilder) WithDeletion() *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			v1t := metav1.NewTime(time.Now())

			b.DeletionTimestamp = &v1t
		})

	return b
}

// WithChaosPodLabels sets chaos-related labels.
func (b *ChaosPodBuilder) WithChaosPodLabels(name, namespace, target, kind string) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Labels[types.DisruptionNameLabel] = name
			b.Labels[types.DisruptionNamespaceLabel] = namespace
			b.Labels[types.TargetLabel] = target
			b.Labels[types.DisruptionKindLabel] = kind
		})

	return b
}

// WithLabels sets custom labels.
func (b *ChaosPodBuilder) WithLabels(labels map[string]string) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			for key, value := range labels {
				b.Labels[key] = value
			}
		})

	return b
}

// WithStatusPhase sets the status phase.
func (b *ChaosPodBuilder) WithStatusPhase(phase v1.PodPhase) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Status.Phase = phase
		})

	return b
}

// WithChaosFinalizer sets the ChaosPodFinalizer.
func (b *ChaosPodBuilder) WithChaosFinalizer() *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.SetFinalizers([]string{types.ChaosPodFinalizer})
		})

	return b
}

// WithStatus sets the status.
func (b *ChaosPodBuilder) WithStatus(status v1.PodStatus) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Status = status
		})

	return b
}

// WithContainerStatuses sets the container statuses to the status.
func (b *ChaosPodBuilder) WithContainerStatuses(statuses []v1.ContainerStatus) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Status.ContainerStatuses = statuses
		})

	return b
}

// WithPullSecrets sets image pull secrets to the spec.
func (b *ChaosPodBuilder) WithPullSecrets(imagePullSecrets []v1.LocalObjectReference) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Spec.ImagePullSecrets = imagePullSecrets
		})

	return b
}

// WithChaosSpec sets the chaos-specific pod spec.
func (b *ChaosPodBuilder) WithChaosSpec(targetNodeName string, terminationGracePeriod, activeDeadlineSeconds int64, args []string, hostPathDirectory, pathFile v1.HostPathType, serviceAccountName string, image string) *ChaosPodBuilder {
	b.modifiers = append(
		b.modifiers,
		func() {
			b.Spec = v1.PodSpec{
				HostPID:                       true,                  // enable host pid
				RestartPolicy:                 v1.RestartPolicyNever, // do not restart the pod on fail or completion
				NodeName:                      targetNodeName,        // specify node name to schedule the pod
				ServiceAccountName:            serviceAccountName,    // service account to use
				TerminationGracePeriodSeconds: &terminationGracePeriod,
				ActiveDeadlineSeconds:         &activeDeadlineSeconds,
				Containers: []v1.Container{
					{
						Name:            "injector",          // container name
						Image:           image,               // container image gathered from controller flags
						ImagePullPolicy: v1.PullIfNotPresent, // pull the image only when it is not present
						Args:            args,                // pass disruption arguments
						SecurityContext: &v1.SecurityContext{
							Privileged: func() *bool { b := true; return &b }(), // enable privileged mode
						},
						ReadinessProbe: &v1.Probe{ // define readiness probe (file created by the injector when the injection is successful)
							PeriodSeconds:    1,
							FailureThreshold: 5,
							ProbeHandler: v1.ProbeHandler{
								Exec: &v1.ExecAction{
									Command: []string{"test", "-f", "/tmp/readiness_probe"},
								},
							},
						},
						Resources: v1.ResourceRequirements{ // set resources requests and limits to zero
							Limits: v1.ResourceList{
								v1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
								v1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
							},
							Requests: v1.ResourceList{
								v1.ResourceCPU:    *resource.NewQuantity(0, resource.DecimalSI),
								v1.ResourceMemory: *resource.NewQuantity(0, resource.DecimalSI),
							},
						},
						Env: []v1.EnvVar{ // define environment variables
							{
								Name: env.InjectorTargetPodHostIP,
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.hostIP",
									},
								},
							},
							{
								Name: env.InjectorChaosPodIP,
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
							{
								Name: env.InjectorPodName,
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
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
						VolumeMounts: []v1.VolumeMount{ // define volume mounts required for disruptions to work
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
				Volumes: []v1.Volume{ // declare volumes required for disruptions to work
					{
						Name: "run",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/run",
								Type: &hostPathDirectory,
							},
						},
					},
					{
						Name: "proc",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/proc",
								Type: &hostPathDirectory,
							},
						},
					},
					{
						Name: "sysrq",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/proc/sys/kernel/sysrq",
								Type: &pathFile,
							},
						},
					},
					{
						Name: "sysrq-trigger",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/proc/sysrq-trigger",
								Type: &pathFile,
							},
						},
					},
					{
						Name: "cgroup",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/sys/fs/cgroup",
								Type: &hostPathDirectory,
							},
						},
					},
					{
						Name: "host",
						VolumeSource: v1.VolumeSource{
							HostPath: &v1.HostPathVolumeSource{
								Path: "/",
								Type: &hostPathDirectory,
							},
						},
					},
				},
			}
		})

	return b
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package services_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/services"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("disk-full direct hostPath writable mount", func() {
	It("mounts the direct hostPath rather than the kubelet pods directory", func() {
		target := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "apps"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:         "app",
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
				}},
				Volumes: []corev1.Volume{{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{Path: "/opt/app-data"},
					},
				}},
			},
		}
		client := fake.NewClientBuilder().WithObjects(target).Build()
		service, err := services.NewChaosPodService(services.ChaosPodServiceConfig{Client: client})
		Expect(err).ToNot(HaveOccurred())

		disruption := &v1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{Name: "fill", Namespace: "apps", CreationTimestamp: metav1.Now()},
			Spec: v1beta1.DisruptionSpec{
				Level:    types.DisruptionLevelPod,
				Duration: "1m",
				DiskFull: &v1beta1.DiskFullSpec{Path: "/data", Capacity: "95%"},
			},
		}
		pod := service.GenerateChaosPodOfDisruption(disruption, "target", "node", nil, types.DisruptionKindDiskFull)

		var writable *corev1.HostPathVolumeSource
		for _, volume := range pod.Spec.Volumes {
			if volume.Name == "disk-full-writable" {
				writable = volume.HostPath
			}
		}
		Expect(writable).ToNot(BeNil())
		Expect(writable.Path).To(Equal("/opt/app-data"))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(corev1.VolumeMount{
			Name:      "disk-full-writable",
			MountPath: "/mnt/host/opt/app-data",
		}))
	})
})

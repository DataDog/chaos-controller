// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"context"
	"strconv"
	"testing"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPodReplacementDeleteStorage(t *testing.T) {
	oldConfigs, oldInjectors := configs, injectors
	flag := podReplacementCmd.Flags().Lookup("delete-storage")
	oldValue, oldChanged := flag.Value.String(), flag.Changed
	t.Cleanup(func() {
		configs, injectors = oldConfigs, oldInjectors
		_ = flag.Value.Set(oldValue)
		flag.Changed = oldChanged
	})

	for _, deleteStorage := range []bool{false, true} {
		t.Run(strconv.FormatBool(deleteStorage), func(t *testing.T) {
			node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "app"},
				Spec: corev1.PodSpec{
					NodeName: "node",
					Volumes: []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data"},
					}}},
				},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.2"},
			}
			pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "app"}}
			client := fake.NewSimpleClientset(node, pod, pvc)
			configs = []injector.Config{{
				Log:       zap.NewNop().Sugar(),
				K8sClient: client,
				Disruption: chaosapi.DisruptionArgs{
					Level: chaostypes.DisruptionLevelPod, TargetName: pod.Name,
					DisruptionNamespace: pod.Namespace, TargetNodeName: node.Name, TargetPodIP: pod.Status.PodIP,
				},
			}}
			injectors = nil
			if err := flag.Value.Set(flag.DefValue); err != nil {
				t.Fatal(err)
			}
			spec := v1beta1.PodReplacementSpec{DeleteStorage: deleteStorage}
			if err := podReplacementCmd.ParseFlags(spec.GenerateArgs()[1:]); err != nil {
				t.Fatal(err)
			}
			podReplacementCmd.PreRun(podReplacementCmd, nil)
			if err := injectors[0].Inject(); err != nil {
				t.Fatal(err)
			}
			_, err := client.CoreV1().PersistentVolumeClaims("app").Get(context.Background(), "data", metav1.GetOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}
			if deleted := apierrors.IsNotFound(err); deleted != deleteStorage {
				t.Fatalf("deleteStorage=%t: PVC deleted=%t", deleteStorage, deleted)
			}
		})
	}
}

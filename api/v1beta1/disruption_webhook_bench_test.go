// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// buildIndexer populates a client-go indexer with n realistic pods,
// matching how the controller-runtime cache stores objects internally.
func buildIndexer(b *testing.B, n int) cache.Indexer {
	b.Helper()

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})

	for i := range n {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "bench-ns",
				Labels:    map[string]string{"app": "bench", "env": "prod", "team": "platform"},
				Annotations: map[string]string{
					"description":        "realistic pod with metadata to make deep copy meaningful",
					"config.hash":        "abc123def456",
					"prometheus.io/port": "9090",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "main",
						Image: "nginx:1.25",
						Env: []corev1.EnvVar{
							{Name: "ENV", Value: "production"},
							{Name: "VERSION", Value: "1.0.0"},
							{Name: "LOG_LEVEL", Value: "info"},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
						},
					},
					{
						Name:  "sidecar",
						Image: "datadog/agent:latest",
						Env: []corev1.EnvVar{
							{Name: "DD_ENV", Value: "prod"},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
							},
						},
					},
					{
						Name: "secrets",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{SecretName: "app-secrets"},
						},
					},
				},
			},
		}
		pod.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})

		if err := indexer.Add(pod); err != nil {
			b.Fatal(err)
		}
	}

	return indexer
}

// listFromIndexer replicates the CacheReader.List hot path:
// fetch objects from the indexer, optionally deep-copy each one, collect into a slice.
func listFromIndexer(indexer cache.Indexer, namespace string, deepCopy bool) []corev1.Pod {
	objs, _ := indexer.ByIndex(cache.NamespaceIndex, namespace)

	pods := make([]corev1.Pod, 0, len(objs))
	for _, item := range objs {
		obj := item.(runtime.Object)
		if deepCopy {
			obj = obj.DeepCopyObject()
		}

		pod := obj.(*corev1.Pod)
		pods = append(pods, *pod)
	}

	return pods
}

func benchmarkCacheList(b *testing.B, podCount int, deepCopy bool) {
	b.Helper()

	indexer := buildIndexer(b, podCount)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		pods := listFromIndexer(indexer, "bench-ns", deepCopy)
		if len(pods) != podCount {
			b.Fatalf("expected %d pods, got %d", podCount, len(pods))
		}
	}
}

// --- With deep copy (default cache behavior) ---

func BenchmarkCacheList_DeepCopy_100(b *testing.B)   { benchmarkCacheList(b, 100, true) }
func BenchmarkCacheList_DeepCopy_1000(b *testing.B)  { benchmarkCacheList(b, 1000, true) }
func BenchmarkCacheList_DeepCopy_5000(b *testing.B)  { benchmarkCacheList(b, 5000, true) }
func BenchmarkCacheList_DeepCopy_10000(b *testing.B) { benchmarkCacheList(b, 10000, true) }

// --- Without deep copy (UnsafeDisableDeepCopy=true) ---

func BenchmarkCacheList_NoDeepCopy_100(b *testing.B)   { benchmarkCacheList(b, 100, false) }
func BenchmarkCacheList_NoDeepCopy_1000(b *testing.B)  { benchmarkCacheList(b, 1000, false) }
func BenchmarkCacheList_NoDeepCopy_5000(b *testing.B)  { benchmarkCacheList(b, 5000, false) }
func BenchmarkCacheList_NoDeepCopy_10000(b *testing.B) { benchmarkCacheList(b, 10000, false) }

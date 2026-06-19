// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	k8scache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Factory is an interface for creating Watchers.
type Factory interface {
	// NewChaosPodWatcher creates a new ChaosPodWatcher with the given name, disruption, and cache.
	NewChaosPodWatcher(name string, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error)

	// NewDisruptionTargetWatcher creates a new DisruptionTargetWatcher with the given name, enableObserver flag, disruption, and cache.
	NewDisruptionTargetWatcher(name string, enableObserver bool, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error)
}

type factory struct{ config FactoryConfig }

type FactoryConfig struct {
	Log            *zap.SugaredLogger
	MetricSink     metrics.Sink
	Reader         client.Reader
	Recorder       record.EventRecorder
	ChaosNamespace string
	// CachePool is the shared namespace cache pool used by DisruptionTargetWatcher.
	// When nil, each watcher creates its own cache (legacy behaviour, kept for tests).
	CachePool *NamespaceCachePool
}

// NewWatcherFactory creates a new instance of the factory for creating new watcher instances.
func NewWatcherFactory(config FactoryConfig) Factory {
	return factory{config: config}
}

// NewChaosPodWatcher creates a new watcher instance for chaos pods.
func (f factory) NewChaosPodWatcher(name string, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error) {
	// Add instance specific labels if provided
	cacheOptions, err := f.newChaosPodCacheOptions(disruption)
	if err != nil {
		return nil, err
	}

	// Create a new handler for this watcher instance
	handler := NewChaosPodHandler(disruption, f.config.Recorder, f.config.Log, NewWatcherMetricsAdapter(f.config.MetricSink, f.config.Log))

	// Create a new watcher configuration object
	watcherConfig := WatcherConfig{
		Name:           name,
		Handler:        &handler,
		ObjectType:     &corev1.Pod{},
		CacheOptions:   cacheOptions,
		Log:            f.config.Log,
		NamespacedName: types.NamespacedName{Name: disruption.GetName(), Namespace: disruption.GetNamespace()},
	}

	return NewWatcher(watcherConfig, cacheMock, nil)
}

// NewDisruptionTargetWatcher creates a new watcher instance for target pods of a disruption.
// When a CachePool is configured on the factory, the watcher uses a shared per-namespace cache
// instead of creating a dedicated cache, reducing goroutine and memory usage at scale.
func (f factory) NewDisruptionTargetWatcher(name string, enableObserver bool, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error) {
	// Compute the label selector for this disruption's targets.
	labelSelector, err := targetselector.GetLabelSelectorFromInstance(disruption)
	if err != nil {
		return nil, fmt.Errorf("could not create the %s disruption target watcher: %w", name, err)
	}

	// Create a new handler for this watcher instance
	h := DisruptionTargetHandler{
		recorder:       f.config.Recorder,
		reader:         f.config.Reader,
		enableObserver: enableObserver,
		disruption:     disruption,
		metricsAdapter: NewWatcherMetricsAdapter(f.config.MetricSink, f.config.Log),
		log:            f.config.Log,
		labelSelector:  labelSelector,
	}

	// targetObjectType can either be a pod or a node
	var targetObjectType client.Object = &corev1.Pod{}

	// Namespace for pool key: empty string for node-level (cluster-scoped).
	targetNamespace := disruption.Namespace
	if disruption.Spec.Level == chaostypes.DisruptionLevelNode {
		targetObjectType = &corev1.Node{}
		targetNamespace = ""
	}

	watcherConfig := WatcherConfig{
		Name:           name,
		Handler:        &h,
		ObjectType:     targetObjectType,
		LabelSelector:  labelSelector,
		Log:            f.config.Log,
		NamespacedName: types.NamespacedName{Name: disruption.GetName(), Namespace: disruption.GetNamespace()},
		Namespace:      targetNamespace,
	}

	if cacheMock == nil && f.config.CachePool != nil {
		// Shared cache path: pool manages cache lifecycle.
		watcherConfig.CachePool = f.config.CachePool
	} else {
		// Own cache path (tests or no pool configured).
		cacheOptions, err := newDisruptionTargetCacheOptions(disruption)
		if err != nil {
			return nil, fmt.Errorf("could not create the %s disruption target watcher: %w", name, err)
		}

		watcherConfig.CacheOptions = cacheOptions
	}

	return NewWatcher(watcherConfig, cacheMock, nil)
}

// newDisruptionTargetCacheOptions creates the cache options used to watch for targets of the given disruption.
func newDisruptionTargetCacheOptions(disruption *v1beta1.Disruption) (k8scache.Options, error) {
	// Get the label selector for the given disruption instance
	disCompleteSelector, err := targetselector.GetLabelSelectorFromInstance(disruption)
	if err != nil {
		return k8scache.Options{}, fmt.Errorf("error getting instance selector: %w", err)
	}

	// If the disruption level is "node", watch for Node objects matching the label selector
	if disruption.Spec.Level == chaostypes.DisruptionLevelNode {
		return k8scache.Options{
			ByObject: map[client.Object]k8scache.ByObject{
				&corev1.Node{}: {Label: disCompleteSelector},
			},
		}, nil
	}

	// If the disruption level is not "node", watch for Pod objects matching the label selector in the disruption's namespace
	return k8scache.Options{
		ByObject: map[client.Object]k8scache.ByObject{
			&corev1.Pod{}: {
				Namespaces: map[string]k8scache.Config{disruption.Namespace: {LabelSelector: disCompleteSelector}},
			},
		},
	}, nil
}

// newChaosPodCacheOptions creates the cache options for a ChaosPodWatcher based on the given disruption object.
// It adds specific labels to the options so that only pods associated with the disruption are watched and cached.
func (f factory) newChaosPodCacheOptions(disruption *v1beta1.Disruption) (k8scache.Options, error) {
	ls := make(map[string]string, 2)

	// Add the disruption name and namespace labels to the label set for this watcher's cache
	if disruption.Name == "" || disruption.Namespace == "" {
		return k8scache.Options{}, fmt.Errorf("the disruption fields name and namespace of the ObjectMeta field are required")
	}

	ls[chaostypes.DisruptionNameLabel] = disruption.Name
	ls[chaostypes.DisruptionNamespaceLabel] = disruption.Namespace

	// Define the cache options for this watcher with the provided labels for Pods
	return k8scache.Options{
		ByObject: map[client.Object]k8scache.ByObject{
			&corev1.Pod{}: {
				Namespaces: map[string]k8scache.Config{f.config.ChaosNamespace: {LabelSelector: labels.SelectorFromValidatedSet(ls)}},
			},
		},
	}, nil
}

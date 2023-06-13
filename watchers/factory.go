// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

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

type factory struct {
	log         *zap.SugaredLogger
	metricSinks metrics.Sink
	reader      client.Reader
	recorder    record.EventRecorder
}

// NewWatcherFactory creates a new instance of the factory for creating new watcher instances.
func NewWatcherFactory(logger *zap.SugaredLogger, metricSinks metrics.Sink, reader client.Reader, recorder record.EventRecorder) Factory {
	return factory{
		log:         logger,
		metricSinks: metricSinks,
		reader:      reader,
		recorder:    recorder,
	}
}

// NewChaosPodWatcher creates a new watcher instance for chaos pods.
func (f factory) NewChaosPodWatcher(name string, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error) {
	// Add instance specific labels if provided
	cacheOptions, err := newChaosPodCacheOptions(disruption)
	if err != nil {
		return nil, err
	}

	// Create a new handler for this watcher instance
	handler := NewChaosPodHandler(f.recorder, disruption, f.log)

	// Create a new watcher configuration object
	watcherConfig := WatcherConfig{
		Name:         name,
		Handler:      &handler,
		ObjectType:   &corev1.Pod{},
		CacheOptions: cacheOptions,
		Log:          f.log,
	}

	return NewWatcher(watcherConfig, cacheMock, nil)
}

// NewDisruptionTargetWatcher creates a new watcher instance for target pods of a disruption.
func (f factory) NewDisruptionTargetWatcher(name string, enableObserver bool, disruption *v1beta1.Disruption, cacheMock k8scache.Cache) (Watcher, error) {
	// Add instance specific labels if provided
	cacheOptions, err := newDisruptionTargetCacheOptions(disruption)
	if err != nil {
		return nil, fmt.Errorf("could not create the %s disruption target watcher. Error: %w", name, err)
	}

	// Create a new handler for this watcher instance
	handler := DisruptionTargetHandler{
		recorder:       f.recorder,
		reader:         f.reader,
		enableObserver: enableObserver,
		disruption:     disruption,
		metricsSink:    f.metricSinks,
		log:            f.log,
	}

	// Create a new watcher configuration object
	watcherConfig := WatcherConfig{
		Name:         name,
		Handler:      &handler,
		ObjectType:   &corev1.Pod{},
		CacheOptions: cacheOptions,
		Log:          f.log,
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
			SelectorsByObject: k8scache.SelectorsByObject{
				&corev1.Node{}: {Label: disCompleteSelector},
			},
		}, nil
	}

	// If the disruption level is not "node", watch for Pod objects matching the label selector in the disruption's namespace
	return k8scache.Options{
		SelectorsByObject: k8scache.SelectorsByObject{
			&corev1.Pod{}: {Label: disCompleteSelector},
		},
		Namespace: disruption.Namespace,
	}, nil
}

// newChaosPodCacheOptions creates the cache options for a ChaosPodWatcher based on the given disruption object.
// It adds specific labels to the options so that only pods associated with the disruption are watched and cached.
func newChaosPodCacheOptions(disruption *v1beta1.Disruption) (k8scache.Options, error) {
	ls := make(map[string]string, 2)

	// Add the disruption name and namespace labels to the label set for this watcher's cache
	if disruption.Name == "" || disruption.Namespace == "" {
		return k8scache.Options{}, fmt.Errorf("the disruption fields name and namespace of the ObjectMeta field are required")
	}

	ls[chaostypes.DisruptionNameLabel] = disruption.Name
	ls[chaostypes.DisruptionNamespaceLabel] = disruption.Namespace

	// Define the cache options for this watcher with the provided labels for Pods
	return k8scache.Options{
		SelectorsByObject: k8scache.SelectorsByObject{
			&corev1.Pod{}: {Label: labels.SelectorFromValidatedSet(ls)},
		},
	}, nil
}

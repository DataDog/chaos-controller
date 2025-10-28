// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"k8s.io/apimachinery/pkg/types"
	k8scontrollercache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	cLog "github.com/DataDog/chaos-controller/log"
)

// DisruptionsWatchersManager defines the interface for a manager that can handle adding, removing Watchers for a disruption
type DisruptionsWatchersManager interface {
	// CreateAllWatchers adds new Watchers instances for a disruption
	CreateAllWatchers(ctx context.Context, disruption *v1beta1.Disruption, watcherManagerMock Manager, cacheMock k8scontrollercache.Cache) error

	// RemoveAllWatchers removes all existing Watchers of a disruption
	RemoveAllWatchers(ctx context.Context, disruption *v1beta1.Disruption)

	// RemoveAllOrphanWatchers removes all watchers linked to an expired disruption
	RemoveAllOrphanWatchers(ctx context.Context) error

	// RemoveAllExpiredWatchers removes all expired Watchers
	RemoveAllExpiredWatchers(ctx context.Context)
}

// WatcherManagers represents a map of Manager instances
type WatcherManagers map[types.NamespacedName]Manager

// disruptionsWatchersManager is the struct that implement the DisruptionsWatchersManager interface.
type disruptionsWatchersManager struct {
	controller       controller.Controller
	factory          Factory
	reader           client.Reader
	watchersManagers WatcherManagers
}

type WatcherName string

const (
	ChaosPodWatcherName         WatcherName = "ChaosPod"
	DisruptionTargetWatcherName WatcherName = "DisruptionTarget"
)

var watchersNames = []WatcherName{
	ChaosPodWatcherName,
	DisruptionTargetWatcherName,
}

// CreateAllWatchers creates all the Watchers associated with the given Disruption.
func (d disruptionsWatchersManager) CreateAllWatchers(ctx context.Context, disruption *v1beta1.Disruption, watcherManagerMock Manager, cacheMock k8scontrollercache.Cache) error {
	// Check that the disruption object has a name and namespace
	if disruption.ObjectMeta.Name == "" || disruption.ObjectMeta.Namespace == "" {
		return fmt.Errorf("the disruption is not valid. It should contain a name and a namespace")
	}

	// Get the namespaced name of the disruption
	disruptionNamespacedName := getDisruptionNamespacedName(disruption)

	var watcherManager Manager

	// If a mock watcher manager was passed in, use it
	if watcherManagerMock == nil {
		watcherManager = d.getWatcherManager(ctx, disruptionNamespacedName)
	} else {
		watcherManager = watcherManagerMock
	}

	// Save the watcher manager for later use
	d.watchersManagers[disruptionNamespacedName] = watcherManager

	// Calculate a hash of the disruption spec (excluding the count field)
	disSpecHash, err := disruption.Spec.HashNoCount()
	if err != nil {
		return err
	}

	// For each type of watcher we need to create
	for _, watcherName := range watchersNames {
		// Calculate a hash for the watcher
		watcherNameHash := disSpecHash + string(watcherName)

		// If a watcher with this hash has already been created
		if ok := watcherManager.GetWatcher(watcherNameHash); ok != nil {
			continue
		}

		// Otherwise add the new watcher for the disruption
		if err := d.addWatcher(disruption, watcherName, watcherNameHash, cacheMock, watcherManager); err != nil {
			return err
		}

		cLog.FromContext(ctx).Debugw("Watcher created", tags.WatcherNameKey, watcherName)
	}

	return nil
}

// RemoveAllWatchers removes all the Watchers associated with the given Disruption.
func (d disruptionsWatchersManager) RemoveAllWatchers(ctx context.Context, disruption *v1beta1.Disruption) {
	logger := cLog.FromContext(ctx)
	namespacedName := getDisruptionNamespacedName(disruption)

	// Get the Watcher Manager associated with the Disruption.
	watcherManager := d.watchersManagers[namespacedName]

	// If the Watcher Manager does not exist just do nothing.
	if watcherManager == nil {
		logger.Debugw("could not remove all watchers")
		return
	}

	watcherManager.RemoveAllWatchers()

	// Remove the Watcher Manager from the map.
	delete(d.watchersManagers, namespacedName)

	logger.Infow("all watchers have been removed")
}

// RemoveAllOrphanWatchers removes all Watchers associated with a none existing Disruption.
func (d disruptionsWatchersManager) RemoveAllOrphanWatchers(ctx context.Context) error {
	// For each stored watcher manager
	for namespacedName, watchersManager := range d.watchersManagers {
		// Check if the disruption still exists
		if err := d.reader.Get(ctx, namespacedName, &v1beta1.Disruption{}); err != nil {
			// If the error is not related to the disruption being missing, skip to the next watcher manager
			if err = client.IgnoreNotFound(err); err != nil {
				continue
			}

			// If the disruption is missing, remove all watchers for this watcher manager
			watchersManager.RemoveAllWatchers()

			// Remove the watcher manager from the stored managers
			delete(d.watchersManagers, namespacedName)

			cLog.FromContext(ctx).Infow("all watchers have been removed",
				tags.WatcherNameKey, namespacedName.Name,
				tags.WatcherNamespaceKey, namespacedName.Namespace,
			)
		}
	}

	return nil
}

// RemoveAllExpiredWatchers loops through all the watcher managers in the disruptionsWatchersManager
// and removes all the expired watchers for each watcher manager.
func (d disruptionsWatchersManager) RemoveAllExpiredWatchers(ctx context.Context) {
	for _, watchersManager := range d.watchersManagers {
		watchersManager.RemoveExpiredWatchers()
	}
}

// NewDisruptionsWatchersManager return a new DisruptionsWatchersManager instance
func NewDisruptionsWatchersManager(controller controller.Controller, factory Factory, reader client.Reader) DisruptionsWatchersManager {
	return disruptionsWatchersManager{
		watchersManagers: WatcherManagers{},
		controller:       controller,
		factory:          factory,
		reader:           reader,
	}
}

func (d disruptionsWatchersManager) addWatcher(disruption *v1beta1.Disruption, watcherName WatcherName, watcherNameHash string, cacheMock k8scontrollercache.Cache, watcherManager Manager) error {
	var (
		newWatcher Watcher
		err        error
	)

	// Create a new watcher based on its name
	switch watcherName {
	case DisruptionTargetWatcherName:
		newWatcher, err = d.factory.NewDisruptionTargetWatcher(watcherNameHash, true, disruption, cacheMock)
	case ChaosPodWatcherName:
		newWatcher, err = d.factory.NewChaosPodWatcher(watcherNameHash, disruption, cacheMock)
	}

	// Check for errors when creating the watcher
	if err != nil {
		return err
	}

	// Add the watcher to the watcher manager
	if err = watcherManager.AddWatcher(newWatcher); err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	return nil
}

func getDisruptionNamespacedName(disruption *v1beta1.Disruption) types.NamespacedName {
	return types.NamespacedName{
		Namespace: disruption.ObjectMeta.Namespace,
		Name:      disruption.ObjectMeta.Name,
	}
}

func (d disruptionsWatchersManager) getWatcherManager(ctx context.Context, disruptionNamespacedName types.NamespacedName) Manager {
	// If we have already created a watcher manager for this disruption, use it
	if cachedWatcherManager := d.watchersManagers[disruptionNamespacedName]; cachedWatcherManager != nil {
		cLog.FromContext(ctx).Debugw("Load watcher manager from the cache")

		return cachedWatcherManager
	}

	cLog.FromContext(ctx).Debugw("Creating a new watcher manager")

	// Otherwise, create a new watcher manager
	return NewManager(d.reader, d.controller)
}

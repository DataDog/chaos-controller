// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"
	"sync"

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
	mu               sync.RWMutex
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
func (d *disruptionsWatchersManager) CreateAllWatchers(ctx context.Context, disruption *v1beta1.Disruption, watcherManagerMock Manager, cacheMock k8scontrollercache.Cache) error {
	// Check that the disruption object has a name and namespace
	if disruption.Name == "" || disruption.Namespace == "" {
		return fmt.Errorf("the disruption is not valid. It should contain a name and a namespace")
	}

	// Get the namespaced name of the disruption
	disruptionNamespacedName := getDisruptionNamespacedName(disruption)

	// Retrieve or create the watcher manager for this disruption, storing it under the write lock.
	d.mu.Lock()

	var watcherManager Manager

	if watcherManagerMock == nil {
		watcherManager = d.getWatcherManagerLocked(ctx, disruptionNamespacedName)
	} else {
		watcherManager = watcherManagerMock
	}

	d.watchersManagers[disruptionNamespacedName] = watcherManager
	d.mu.Unlock()

	// Calculate a hash of the disruption spec (excluding the count field).
	// Watcher creation (I/O) happens outside the lock.
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
func (d *disruptionsWatchersManager) RemoveAllWatchers(ctx context.Context, disruption *v1beta1.Disruption) {
	logger := cLog.FromContext(ctx)
	namespacedName := getDisruptionNamespacedName(disruption)

	// Hold d.mu for the entire clean+delete so that a concurrent CreateAllWatchers for
	// the same key cannot slip watchers in between the cleanup call and the map deletion.
	// Lock order d.mu → m.mu is safe: CreateAllWatchers never holds both simultaneously
	// (it releases d.mu before acquiring m.mu for AddWatcher).
	d.mu.Lock()

	watcherManager := d.watchersManagers[namespacedName]

	if watcherManager == nil {
		d.mu.Unlock()
		logger.Debugw("could not remove all watchers")

		return
	}

	// Clean all watchers and remove the map entry atomically under d.mu, preventing
	// any concurrent CreateAllWatchers from adding new watchers to this manager after
	// cleanup but before the entry is removed.
	watcherManager.RemoveAllWatchers()
	delete(d.watchersManagers, namespacedName)

	d.mu.Unlock()

	logger.Infow("all watchers have been removed")
}

// RemoveAllOrphanWatchers removes all Watchers associated with a none existing Disruption.
func (d *disruptionsWatchersManager) RemoveAllOrphanWatchers(ctx context.Context) error {
	type watcherEntry struct {
		namespacedName types.NamespacedName
		manager        Manager
	}

	// Step 1: snapshot current (key, manager) pairs under read lock to avoid holding the lock
	// during K8s API calls.
	d.mu.RLock()

	snapshot := make([]watcherEntry, 0, len(d.watchersManagers))

	for k, m := range d.watchersManagers {
		snapshot = append(snapshot, watcherEntry{k, m})
	}

	d.mu.RUnlock()

	// Step 2: check which disruptions no longer exist — no lock held during network I/O.
	var toRemove []watcherEntry

	for _, e := range snapshot {
		if err := d.reader.Get(ctx, e.namespacedName, &v1beta1.Disruption{}); err != nil {
			// If the error is not related to the disruption being missing, skip to the next watcher manager
			if err = client.IgnoreNotFound(err); err != nil {
				continue
			}

			toRemove = append(toRemove, e)
		}
	}

	// Step 3: unregister orphaned managers from the map under the write lock, then call
	// RemoveAllWatchers outside the lock to avoid a lock-order inversion with the per-manager
	// mutex.
	// Only remove the entry if the manager instance in the map is the same one we snapshotted.
	// A different instance means a new disruption with the same name was created after Step 1,
	// and we must not tear down its watchers.
	var orphans []watcherEntry

	d.mu.Lock()

	for _, e := range toRemove {
		currentManager := d.watchersManagers[e.namespacedName]
		if currentManager == nil || currentManager != e.manager {
			// Either already removed concurrently, or replaced by a newly created disruption
			// with the same namespace/name — do not remove.
			continue
		}

		delete(d.watchersManagers, e.namespacedName)
		orphans = append(orphans, e)
	}

	d.mu.Unlock()

	// Call RemoveAllWatchers outside d.mu to prevent lock-order inversion.
	for _, orphan := range orphans {
		orphan.manager.RemoveAllWatchers()

		cLog.FromContext(ctx).Infow("all watchers have been removed",
			tags.WatcherNameKey, orphan.namespacedName.Name,
			tags.WatcherNamespaceKey, orphan.namespacedName.Namespace,
		)
	}

	return nil
}

// RemoveAllExpiredWatchers loops through all the watcher managers in the disruptionsWatchersManager
// and removes all the expired watchers for each watcher manager.
func (d *disruptionsWatchersManager) RemoveAllExpiredWatchers(ctx context.Context) {
	// Snapshot the managers under read lock; RemoveExpiredWatchers may do I/O.
	d.mu.RLock()

	managers := make([]Manager, 0, len(d.watchersManagers))

	for _, m := range d.watchersManagers {
		managers = append(managers, m)
	}

	d.mu.RUnlock()

	for _, m := range managers {
		m.RemoveExpiredWatchers()
	}
}

// NewDisruptionsWatchersManager return a new DisruptionsWatchersManager instance
func NewDisruptionsWatchersManager(controller controller.Controller, factory Factory, reader client.Reader) DisruptionsWatchersManager {
	return &disruptionsWatchersManager{
		watchersManagers: WatcherManagers{},
		controller:       controller,
		factory:          factory,
		reader:           reader,
	}
}

func (d *disruptionsWatchersManager) addWatcher(disruption *v1beta1.Disruption, watcherName WatcherName, watcherNameHash string, cacheMock k8scontrollercache.Cache, watcherManager Manager) error {
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
		Namespace: disruption.Namespace,
		Name:      disruption.Name,
	}
}

// getWatcherManagerLocked returns the cached watcher manager for the given disruption, or creates a new one.
// Caller must hold d.mu (at least a read lock, write lock required if result will be stored).
func (d *disruptionsWatchersManager) getWatcherManagerLocked(ctx context.Context, disruptionNamespacedName types.NamespacedName) Manager {
	// If we have already created a watcher manager for this disruption, use it
	if cachedWatcherManager := d.watchersManagers[disruptionNamespacedName]; cachedWatcherManager != nil {
		cLog.FromContext(ctx).Debugw("Load watcher manager from the cache")

		return cachedWatcherManager
	}

	cLog.FromContext(ctx).Debugw("Creating a new watcher manager")

	// Otherwise, create a new watcher manager
	return NewManager(d.reader, d.controller)
}

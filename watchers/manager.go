// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers

import (
	context "context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Manager defines the interface for a manager that can handle adding, removing, and retrieving Watchers.
type Manager interface {
	// AddWatcher adds a new Watcher instance to the Manager
	AddWatcher(watcher Watcher) error

	// RemoveWatcher removes an existing Watcher instance from the Manager
	RemoveWatcher(watcher Watcher) error

	// RemoveExpiredWatchers removes all expired Watcher instances from the Manager
	RemoveExpiredWatchers()

	// GetWatcher returns the Watcher instance with the specified name
	GetWatcher(name string) Watcher

	// RemoveOrphanWatchers removes all orphan Watcher instances from the Manager
	RemoveOrphanWatchers()

	// RemoveAllWatchers remove all Watchers instances from the Manager
	RemoveAllWatchers()
}

// Watchers represents a map of Watcher instances
type Watchers map[string]Watcher

// manager is a struct that implement the Manager interface.
type manager struct {
	// controller is used to take event of a source send them to a watcher
	controller controller.Controller

	// reader used to communicate with the k8s api. It allows us to check the existence of resources in k8s
	reader client.Reader

	// watchers represents a map of Watcher instances
	watchers Watchers
}

// NewManager creates a new Manager instance
func NewManager(r client.Reader, c controller.Controller) Manager {
	return &manager{
		controller: c,
		reader:     r,
		watchers:   Watchers{},
	}
}

// GetWatcher returns the Watcher instance with the given name
func (m manager) GetWatcher(name string) Watcher {
	return m.watchers[name]
}

// AddWatcher adds a new Watcher instance to the Manager
func (m manager) AddWatcher(w Watcher) error {
	watcherName := w.GetName()

	// Check if the Watcher instance already exists in the Manager
	if ok := m.watchers[watcherName]; ok != nil {
		return nil
	}

	// Start the Watcher instance
	if err := w.Start(); err != nil {
		return fmt.Errorf("could not start the watcher. Error: %w", err)
	}

	// Add the Watcher instance to the Manager's Watchers
	m.watchers[watcherName] = w

	// Get the cache source of the Watcher instance
	cacheSource, err := w.GetCacheSource()
	if err != nil {
		return fmt.Errorf("could not get the cache source of the watcher. Error: %w", err)
	}

	// Get the context tuple of the Watcher instance
	ctxTuple, _ := w.GetContextTuple()

	// Watch for changes in the cache
	return m.controller.Watch(
		cacheSource,
		handler.EnqueueRequestsFromMapFunc(
			func(_ context.Context, _ client.Object) []reconcile.Request {
				return []reconcile.Request{{NamespacedName: ctxTuple.DisruptionNamespacedName}}
			},
		),
	)
}

// RemoveWatcher removes a Watcher instance from the Manager
func (m manager) RemoveWatcher(w Watcher) error {
	watcherName := w.GetName()

	// Check if the Watcher instance exists in the Manager
	if ok := m.watchers[watcherName]; ok == nil {
		return fmt.Errorf("the watcher %s does not exist", watcherName)
	}

	// Stop and delete the Watcher instance
	w.Clean()
	delete(m.watchers, watcherName)

	return nil
}

// RemoveExpiredWatchers removes any expired Watcher instances from the Manager
func (m manager) RemoveExpiredWatchers() {
	for name, w := range m.watchers {
		if w.IsExpired() {
			w.Clean()
			delete(m.watchers, name)

			continue
		}
	}
}

// RemoveOrphanWatchers removes any Watcher instances without linked Disruption
func (m manager) RemoveOrphanWatchers() {
	for name, w := range m.watchers {
		ctxTuple, _ := w.GetContextTuple()

		if err := m.reader.Get(ctxTuple.Ctx, ctxTuple.DisruptionNamespacedName, &v1beta1.Disruption{}); err != nil {
			if err = client.IgnoreNotFound(err); err != nil {
				continue
			}

			w.Clean()

			delete(m.watchers, name)
		}
	}
}

// RemoveAllWatchers remove all Watchers instances from the Manager
func (m manager) RemoveAllWatchers() {
	for name, w := range m.watchers {
		w.Clean()

		delete(m.watchers, name)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"fmt"
	"sync"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
	mu       sync.RWMutex
	watchers map[string]*watcherEntry
}

// watcherEntry reserves a name during startup. Only ready entries may be inspected
// or cleaned; a detached pending entry is cleaned by AddWatcher after startup.
type watcherEntry struct {
	watcher Watcher
	ready   bool
}

// NewManager creates a new Manager instance
func NewManager(r client.Reader, c controller.Controller) Manager {
	return &manager{
		controller: c,
		reader:     r,
		watchers:   map[string]*watcherEntry{},
	}
}

// GetWatcher returns the initialized watcher with the given name.
func (m *manager) GetWatcher(name string) Watcher {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if entry := m.watchers[name]; entry != nil && entry.ready {
		return entry.watcher
	}
	return nil
}

// AddWatcher reserves the name before starting the watcher. Startup and controller
// registration may block, so neither runs with the map lock held.
func (m *manager) AddWatcher(w Watcher) error {
	name := w.GetName()
	entry := &watcherEntry{watcher: w}
	m.mu.Lock()
	if m.watchers[name] != nil {
		m.mu.Unlock()
		return nil
	}
	m.watchers[name] = entry
	m.mu.Unlock()

	if err := w.Start(); err != nil {
		m.detach(name, entry)
		return fmt.Errorf("could not start the watcher. Error: %w", err)
	}

	cacheSource, err := w.GetCacheSource()
	if err != nil {
		err = fmt.Errorf("could not get the cache source of the watcher. Error: %w", err)
	} else {
		err = m.controller.Watch(cacheSource)
	}

	m.mu.Lock()
	retained := m.watchers[name] == entry
	if retained {
		entry.ready = true
	}
	m.mu.Unlock()
	if !retained {
		w.Clean()
	}
	return err
}

// RemoveWatcher removes an existing watcher. Pending startup owns its cleanup.
func (m *manager) RemoveWatcher(w Watcher) error {
	name := w.GetName()
	m.mu.Lock()
	entry := m.watchers[name]
	delete(m.watchers, name)
	ready := entry != nil && entry.ready
	m.mu.Unlock()
	if entry == nil {
		return fmt.Errorf("the watcher %s does not exist", name)
	}
	if ready {
		entry.watcher.Clean()
	}
	return nil
}

// snapshot excludes pending entries so callbacks cannot race their initialization.
func (m *manager) snapshot() map[string]*watcherEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entries := make(map[string]*watcherEntry, len(m.watchers))
	for name, entry := range m.watchers {
		if entry.ready {
			entries[name] = entry
		}
	}
	return entries
}

// detach claims an entry for removal without deleting a replacement of that name.
func (m *manager) detach(name string, entry *watcherEntry) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.watchers[name] != entry {
		return false
	}
	delete(m.watchers, name)
	return true
}

// RemoveExpiredWatchers removes expired watchers without holding locks across callbacks.
func (m *manager) RemoveExpiredWatchers() {
	for name, entry := range m.snapshot() {
		if entry.watcher.IsExpired() && m.detach(name, entry) {
			entry.watcher.Clean()
		}
	}
}

// RemoveOrphanWatchers removes watchers whose linked disruption no longer exists.
func (m *manager) RemoveOrphanWatchers() {
	for name, entry := range m.snapshot() {
		ctxTuple, _ := entry.watcher.GetContextTuple()
		if err := m.reader.Get(ctxTuple.Ctx, ctxTuple.NamespacedName, &v1beta1.Disruption{}); err != nil && client.IgnoreNotFound(err) == nil && m.detach(name, entry) {
			entry.watcher.Clean()
		}
	}
}

// RemoveAllWatchers detaches the map atomically, then cleans initialized watchers.
func (m *manager) RemoveAllWatchers() {
	m.mu.Lock()
	entries := m.watchers
	m.watchers = map[string]*watcherEntry{}
	m.mu.Unlock()
	for _, entry := range entries {
		// Entries in the detached map can no longer be made ready by AddWatcher.
		if entry.ready {
			entry.watcher.Clean()
		}
	}
}

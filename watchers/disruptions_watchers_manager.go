// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/o11y/tracer"
	"github.com/DataDog/chaos-controller/o11y/tracer/attributes"
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

	// RemoveWatchersForDisruption removes all watchers for a single disruption identified by its NamespacedName.
	// This is a targeted O(1) cleanup used when reconcile determines the disruption no longer exists.
	RemoveWatchersForDisruption(ctx context.Context, namespacedName types.NamespacedName)
}

// WatcherManagers represents a map of Manager instances
type WatcherManagers map[types.NamespacedName]Manager

// disruptionsWatchersManager is the struct that implement the DisruptionsWatchersManager interface.
type disruptionsWatchersManager struct {
	controller       controller.Controller
	factory          Factory
	reader           client.Reader
	mu               sync.RWMutex
	watchersManagers map[types.NamespacedName]*watcherManagerEntry
}

// watcherManagerEntry keeps the manager and object identity together. Cleanup of a
// detached entry waits for its in-flight creators without blocking other entries.
type watcherManagerEntry struct {
	manager  Manager
	uid      types.UID
	creators int
	removed  bool
}

type WatcherName string

const (
	ChaosPodWatcherName         WatcherName = "ChaosPod"
	DisruptionTargetWatcherName WatcherName = "DisruptionTarget"
)

// Only DisruptionTargetWatcher is created per-disruption.
// Chaos pod watching is handled globally by SharedChaosPodHandler registered in
// SetupWithManager, eliminating one informer cache per active disruption.
var watchersNames = []WatcherName{
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

	entry := d.acquireManager(disruptionNamespacedName, disruption.UID, watcherManagerMock)
	defer d.releaseManager(entry)
	watcherManager := entry.manager

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

		ctx, addWatcherSpan := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.add_watcher",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String(attributes.DisruptionName, disruption.Name),
				attribute.String(attributes.DisruptionNamespace, disruption.Namespace),
				attribute.String(attributes.WatchersKind, string(watcherName)),
			))

		addErr := d.addWatcher(disruption, watcherName, watcherNameHash, cacheMock, watcherManager)
		endWatcherSpan(addWatcherSpan, addErr)

		if addErr != nil {
			return addErr
		}

		cLog.FromContext(ctx).Debugw("Watcher created", tags.WatcherNameKey, watcherName)
	}

	return nil
}

// RemoveAllWatchers removes all the Watchers associated with the given Disruption.
func (d *disruptionsWatchersManager) RemoveAllWatchers(ctx context.Context, disruption *v1beta1.Disruption) {
	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.remove_all_for_disruption",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(attributes.DisruptionName, disruption.Name),
			attribute.String(attributes.DisruptionNamespace, disruption.Namespace),
		))
	defer endWatcherSpan(span, nil)

	logger := cLog.FromContext(ctx)
	namespacedName := getDisruptionNamespacedName(disruption)

	found := d.removeManager(namespacedName, nil)
	span.SetAttributes(attribute.Bool(attributes.WatchersManagerFound, found))
	if !found {
		logger.Debugw("could not remove all watchers")
		return
	}

	logger.Infow("all watchers have been removed")
}

// RemoveAllOrphanWatchers removes all Watchers associated with a none existing Disruption.
func (d *disruptionsWatchersManager) RemoveAllOrphanWatchers(ctx context.Context) error {
	managers := d.snapshotManagers()
	if len(managers) == 0 {
		return nil
	}

	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.scan_orphan_managers",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.Int(attributes.WatchersStoredManagers, len(managers))))
	defer endWatcherSpan(span, nil)

	var orphansRemoved int

	defer func() {
		span.SetAttributes(attribute.Int(attributes.WatchersOrphansRemoved, orphansRemoved))
	}()

	// Single List call to fetch all existing disruptions (O(1) API calls instead of O(n))
	disruptionList := &v1beta1.DisruptionList{}
	if err := d.reader.List(ctx, disruptionList); err != nil {
		return err
	}

	// Build a set of existing disruptions for O(1) membership lookups
	existing := make(map[types.NamespacedName]struct{}, len(disruptionList.Items))
	for i := range disruptionList.Items {
		existing[types.NamespacedName{
			Namespace: disruptionList.Items[i].Namespace,
			Name:      disruptionList.Items[i].Name,
		}] = struct{}{}
	}

	// For each stored watcher manager, remove it if its disruption no longer exists
	for namespacedName, entry := range managers {
		if _, found := existing[namespacedName]; found {
			continue
		}

		dropCtx, dropSpan := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.drop_orphan_manager",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String(attributes.DisruptionName, namespacedName.Name),
				attribute.String(attributes.DisruptionNamespace, namespacedName.Namespace),
			))

		removed := d.removeManager(namespacedName, entry)
		endWatcherSpan(dropSpan, nil)
		if !removed {
			continue
		}
		orphansRemoved++

		cLog.FromContext(dropCtx).Infow("all watchers have been removed",
			tags.WatcherNameKey, namespacedName.Name,
			tags.WatcherNamespaceKey, namespacedName.Namespace,
		)
	}

	return nil
}

// RemoveWatchersForDisruption removes all watchers for the given disruption without scanning the full cache.
// It is called from the reconcile NotFound path so cleanup happens immediately, rather than waiting for
// the 5-minute orphan sweep that runs in the background.
func (d *disruptionsWatchersManager) RemoveWatchersForDisruption(ctx context.Context, namespacedName types.NamespacedName) {
	if !d.removeManager(namespacedName, nil) {
		return
	}

	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.remove_for_disruption",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(attributes.DisruptionName, namespacedName.Name),
			attribute.String(attributes.DisruptionNamespace, namespacedName.Namespace),
		))
	defer endWatcherSpan(span, nil)

	cLog.FromContext(ctx).Infow("targeted watcher cleanup completed",
		tags.WatcherNameKey, namespacedName.Name,
		tags.WatcherNamespaceKey, namespacedName.Namespace,
	)
}

// RemoveAllExpiredWatchers loops through all the watcher managers in the disruptionsWatchersManager
// and removes all the expired watchers for each watcher manager.
func (d *disruptionsWatchersManager) RemoveAllExpiredWatchers(ctx context.Context) {
	for _, entry := range d.snapshotManagers() {
		entry.manager.RemoveExpiredWatchers()
	}
}

// NewDisruptionsWatchersManager return a new DisruptionsWatchersManager instance
func NewDisruptionsWatchersManager(controller controller.Controller, factory Factory, reader client.Reader) DisruptionsWatchersManager {
	return &disruptionsWatchersManager{
		watchersManagers: map[types.NamespacedName]*watcherManagerEntry{},
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

// acquireManager publishes the manager before startup and tracks creators so a
// concurrent removal cannot leave a newly started watcher in a detached manager.
func (d *disruptionsWatchersManager) acquireManager(name types.NamespacedName, uid types.UID, supplied Manager) *watcherManagerEntry {
	d.mu.Lock()
	entry := d.watchersManagers[name]
	var obsolete Manager
	if entry != nil && entry.uid != uid {
		entry.removed = true
		if entry.creators == 0 {
			obsolete = entry.manager
		}
		entry = nil
	}
	if entry == nil || supplied != nil {
		manager := supplied
		if manager == nil {
			manager = NewManager(d.reader, d.controller)
		}
		entry = &watcherManagerEntry{manager: manager, uid: uid}
		d.watchersManagers[name] = entry
	}
	entry.creators++
	d.mu.Unlock()
	if obsolete != nil {
		obsolete.RemoveAllWatchers()
	}
	return entry
}

func (d *disruptionsWatchersManager) releaseManager(entry *watcherManagerEntry) {
	d.mu.Lock()
	entry.creators--
	cleanup := entry.removed && entry.creators == 0
	d.mu.Unlock()
	if cleanup {
		entry.manager.RemoveAllWatchers()
	}
}

func (d *disruptionsWatchersManager) snapshotManagers() map[types.NamespacedName]*watcherManagerEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entries := make(map[types.NamespacedName]*watcherManagerEntry, len(d.watchersManagers))
	for name, entry := range d.watchersManagers {
		entries[name] = entry
	}
	return entries
}

// removeManager claims cleanup only if the snapshot still identifies this entry.
// In particular, an orphan List response cannot evict a concurrently replaced manager.
func (d *disruptionsWatchersManager) removeManager(name types.NamespacedName, expected *watcherManagerEntry) bool {
	d.mu.Lock()
	entry := d.watchersManagers[name]
	if entry == nil || (expected != nil && entry != expected) {
		d.mu.Unlock()
		return false
	}
	delete(d.watchersManagers, name)
	entry.removed = true
	cleanup := entry.creators == 0
	d.mu.Unlock()
	if cleanup {
		entry.manager.RemoveAllWatchers()
	}
	return true
}

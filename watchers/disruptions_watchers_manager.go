// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"github.com/DataDog/chaos-controller/o11y/tracer"
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
	watchersManagers WatcherManagers
	managerUIDs      map[types.NamespacedName]types.UID
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
	if disruption.Name == "" || disruption.Namespace == "" {
		return fmt.Errorf("the disruption is not valid. It should contain a name and a namespace")
	}

	// Get the namespaced name of the disruption
	disruptionNamespacedName := getDisruptionNamespacedName(disruption)

	// Evict the cached manager if the disruption was recreated under the same namespace/name.
	// A UID mismatch means the Kubernetes object is new; the old cached manager must not be reused.
	if cachedUID, ok := d.managerUIDs[disruptionNamespacedName]; ok && cachedUID != disruption.UID {
		d.watchersManagers[disruptionNamespacedName].RemoveAllWatchers()
		delete(d.watchersManagers, disruptionNamespacedName)
		delete(d.managerUIDs, disruptionNamespacedName)
	}

	var watcherManager Manager

	// If a mock watcher manager was passed in, use it
	if watcherManagerMock == nil {
		watcherManager = d.getWatcherManager(ctx, disruptionNamespacedName)
	} else {
		watcherManager = watcherManagerMock
	}

	// Save the watcher manager for later use
	d.watchersManagers[disruptionNamespacedName] = watcherManager
	d.managerUIDs[disruptionNamespacedName] = disruption.UID

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
				attribute.String("disruption.name", disruption.Name),
				attribute.String("disruption.namespace", disruption.Namespace),
				attribute.String("chaos.watchers.kind", string(watcherName)),
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
func (d disruptionsWatchersManager) RemoveAllWatchers(ctx context.Context, disruption *v1beta1.Disruption) {
	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.remove_all_for_disruption",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("disruption.name", disruption.Name),
			attribute.String("disruption.namespace", disruption.Namespace),
		))
	defer endWatcherSpan(span, nil)

	logger := cLog.FromContext(ctx)
	namespacedName := getDisruptionNamespacedName(disruption)

	// Get the Watcher Manager associated with the Disruption.
	watcherManager := d.watchersManagers[namespacedName]

	// If the Watcher Manager does not exist just do nothing.
	if watcherManager == nil {
		span.SetAttributes(attribute.Bool("chaos.watchers.manager_found", false))
		logger.Debugw("could not remove all watchers")

		return
	}

	span.SetAttributes(attribute.Bool("chaos.watchers.manager_found", true))

	watcherManager.RemoveAllWatchers()

	// Remove the Watcher Manager from the map.
	delete(d.watchersManagers, namespacedName)
	delete(d.managerUIDs, namespacedName)

	logger.Infow("all watchers have been removed")
}

// RemoveAllOrphanWatchers removes all Watchers associated with a none existing Disruption.
func (d disruptionsWatchersManager) RemoveAllOrphanWatchers(ctx context.Context) error {
	if len(d.watchersManagers) == 0 {
		return nil
	}

	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.scan_orphan_managers",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.Int("chaos.watchers.stored_managers", len(d.watchersManagers))))
	defer endWatcherSpan(span, nil)

	var orphansRemoved int

	defer func() {
		span.SetAttributes(attribute.Int("chaos.watchers.orphans_removed", orphansRemoved))
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
	for namespacedName, watchersManager := range d.watchersManagers {
		if _, found := existing[namespacedName]; found {
			continue
		}

		dropCtx, dropSpan := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.drop_orphan_manager",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("disruption.name", namespacedName.Name),
				attribute.String("disruption.namespace", namespacedName.Namespace),
			))

		watchersManager.RemoveAllWatchers()
		delete(d.watchersManagers, namespacedName)
		delete(d.managerUIDs, namespacedName)

		orphansRemoved++

		endWatcherSpan(dropSpan, nil)

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
func (d disruptionsWatchersManager) RemoveWatchersForDisruption(ctx context.Context, namespacedName types.NamespacedName) {
	watcherManager, ok := d.watchersManagers[namespacedName]
	if !ok {
		return
	}

	ctx, span := otel.Tracer(tracer.InstrumentationScopeDisruption).Start(ctx, "disruption.watchers.remove_for_disruption",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("disruption.name", namespacedName.Name),
			attribute.String("disruption.namespace", namespacedName.Namespace),
		))
	defer endWatcherSpan(span, nil)

	watcherManager.RemoveAllWatchers()
	delete(d.watchersManagers, namespacedName)
	delete(d.managerUIDs, namespacedName)

	cLog.FromContext(ctx).Infow("targeted watcher cleanup completed",
		tags.WatcherNameKey, namespacedName.Name,
		tags.WatcherNamespaceKey, namespacedName.Namespace,
	)
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
		managerUIDs:      map[types.NamespacedName]types.UID{},
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
		Namespace: disruption.Namespace,
		Name:      disruption.Name,
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

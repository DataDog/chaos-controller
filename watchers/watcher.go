// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/tags"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	k8sclientcache "k8s.io/client-go/tools/cache"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8scontrollercache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type WatcherEventType string

const (
	WatcherAddEvent    WatcherEventType = "Add"
	WatcherUpdateEvent WatcherEventType = "Update"
	WatcherDeleteEvent WatcherEventType = "Delete"
)

// Watcher is an interface that describes the methods provided by a Kubernetes resource watcher.
type Watcher interface {
	// Clean cancels the context associated with the watcher, stopping the syncing of resources and freeing up resources.
	Clean()

	// GetCacheSource returns the syncing source of the watcher's cache.
	GetCacheSource() (source.SyncingSource, error)

	// GetContextTuple returns the context tuple associated with the watcher.
	GetContextTuple() (CtxTuple, error)

	// GetName returns the name of the watcher.
	GetName() string

	// IsExpired returns true if the context associated with the watcher has expired, meaning the watcher is no longer active.
	IsExpired() bool

	// Start starts the watcher by creating an informer from the watcher's cache and adding a resource event handler to the informer.
	Start() error

	// GetConfig return the current config of the Watcher instance.
	GetConfig() WatcherConfig
}

// WatcherConfig holds configuration values used to create a new Watcher instance.
type WatcherConfig struct {
	// CacheOptions for configuring the cache. Ignored when CachePool is set.
	CacheOptions k8scontrollercache.Options

	// CachePool, when set, provides a shared namespace cache instead of creating
	// a dedicated cache for this watcher.
	CachePool *NamespaceCachePool

	// Namespace is the target namespace used as the pool key when CachePool is set.
	// Use "" for cluster-scoped (node-level) disruptions.
	Namespace string

	// LabelSelector, when set, filters reconcile enqueue requests so that only
	// pod/node events matching the selector trigger a reconcile for this disruption.
	LabelSelector labels.Selector

	// Handler function that will be called when an event occurs.
	Handler k8sclientcache.ResourceEventHandler

	// Log for logging.
	Log *zap.SugaredLogger

	// Name of the watcher instance.
	Name string

	// NamespacedName of the disruption this watcher belongs to.
	NamespacedName types.NamespacedName

	// ObjectType of the object to watch.
	ObjectType client.Object
}

// CtxTuple is a struct that holds a context and its cancel function, as well as a NamespacedName that identifies a resource.
type CtxTuple struct {
	// CancelFunc is a function that can be called to cancel the associated context.
	CancelFunc context.CancelFunc

	// Ctx is a context.Context object that is used to manage the lifetime of an operation.
	Ctx context.Context

	// NamespacedName is the namespaced name of a resource that is associated with this CtxTuple.
	NamespacedName types.NamespacedName
}

// CacheContextFunc is a function that returns a context and a cancel function.
type CacheContextFunc func() (ctx context.Context, cancel context.CancelFunc)

// watcher represents a Kubernetes resource watcher, including its configuration, cache, cache source, and context tuple.
type watcher struct {
	// The Kubernetes resource cache that stores watched resources
	cache k8scontrollercache.Cache

	// informer is stored so that the per-disruption event handler can be removed when the watcher
	// is cleaned up while the shared namespace cache (pool) is still alive for other disruptions.
	informer k8scontrollercache.Informer

	// handlerReg is the registration returned by AddEventHandler. Used to remove the handler on cleanup.
	handlerReg k8sclientcache.ResourceEventHandlerRegistration

	// The source that provides the cache with Kubernetes resource updates
	cacheSource source.SyncingSource

	// The configuration used to create the watcher
	config WatcherConfig

	// The context tuple that contains the per-watcher context used for lifecycle tracking.
	ctxTuple CtxTuple

	// cacheCancelFunc cancels the cache goroutine. Only set when this watcher owns its cache.
	// Nil when using a shared cache from NamespaceCachePool.
	cacheCancelFunc context.CancelFunc
}

// NewWatcher is a function that creates a new watcher instance based on the given configuration values.
func NewWatcher(config WatcherConfig, cacheMock k8scontrollercache.Cache, cacheContextMockFunc CacheContextFunc) (Watcher, error) {
	watcherInstance := watcher{
		config: config,
	}

	switch {
	case cacheMock != nil:
		// Unit test: use provided mock cache.
		watcherInstance.cache = cacheMock

	case config.CachePool != nil:
		// Shared cache: get-or-create from the pool. The pool owns the cache lifecycle.
		sharedCache, err := config.CachePool.GetOrCreate(config.Namespace, config.ObjectType)
		if err != nil {
			return nil, fmt.Errorf("error getting shared cache from pool: %w", err)
		}

		watcherInstance.cache = sharedCache

	default:
		// Own cache: create a dedicated cache for this watcher.
		cache, err := k8scontrollercache.New(
			controllerruntime.GetConfigOrDie(),
			config.CacheOptions,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating cache: %w", err)
		}

		watcherInstance.cache = cache
	}

	// Used by unit test to allow mocking
	if cacheContextMockFunc != nil {
		cacheCtx, cacheCancelFunc := cacheContextMockFunc()
		watcherInstance.ctxTuple = CtxTuple{cacheCancelFunc, cacheCtx, config.NamespacedName}
	}

	return &watcherInstance, nil
}

// Clean stops the watcher. For own-cache watchers, it cancels the cache goroutine.
// For shared-cache watchers, it removes the per-disruption event handler from the shared
// informer (so it stops firing for this disruption) and releases the pool reference.
func (w *watcher) Clean() {
	w.ctxTuple.CancelFunc()

	if w.cacheCancelFunc != nil {
		w.cacheCancelFunc()
	} else if w.config.CachePool != nil {
		// Remove the event handler so it stops processing events for this disruption
		// even if the shared cache remains alive for other disruptions.
		if w.informer != nil && w.handlerReg != nil {
			_ = w.informer.RemoveEventHandler(w.handlerReg)
		}

		w.config.CachePool.Release(w.config.Namespace)
	}
}

// GetContextTuple is a method that returns a CtxTuple instance.
// It returns an error if the CtxTuple has not been initialized yet.
func (w *watcher) GetContextTuple() (CtxTuple, error) {
	if w.ctxTuple.Ctx == nil {
		return w.ctxTuple, fmt.Errorf("the watcher should be started with its Start method in order to initialize the context tuple")
	}

	return w.ctxTuple, nil
}

// GetCacheSource returns the syncing source associated with the watcher. This method returns an error if the cache source
// has not been initialized yet (i.e., if the watcher has not been started with the Start method).
func (w *watcher) GetCacheSource() (source.SyncingSource, error) {
	if w.cacheSource == nil {
		return nil, fmt.Errorf("the watcher should be started with its Start method in order to initialise the cache source")
	}

	return w.cacheSource, nil
}

// GetName returns the name of the watcher
func (w *watcher) GetName() string {
	return w.config.Name
}

// IsExpired returns a boolean indicating if the watcher has expired or not.
// A watcher is considered expired if its context has been cancelled or if its context deadline has been exceeded.
func (w *watcher) IsExpired() bool {
	err := w.ctxTuple.Ctx.Err()

	return err != nil
}

// Start starts the watcher and sets up the cache and event handlers.
func (w *watcher) Start() error {
	// get informer from cache
	info, err := w.cache.GetInformer(context.Background(), w.config.ObjectType)
	if err != nil {
		return fmt.Errorf("error getting informer from cache: %w", err)
	}

	// Store the informer and register the per-disruption event handler.
	// The registration is kept so it can be removed in Clean() when the watcher
	// is torn down while the shared namespace cache (pool) stays alive.
	w.informer = info

	reg, err := info.AddEventHandler(w.config.Handler)
	if err != nil {
		return fmt.Errorf("error adding event handler to the informer: %w", err)
	}

	w.handlerReg = reg

	// Per-watcher context used for lifecycle tracking (IsExpired, Clean).
	// This is separate from the cache context so that cleaning a single watcher
	// does not stop a shared cache.
	watcherCtx, watcherCancelFunc := context.WithCancel(context.Background()) //nolint:gosec // G118 - cancel func is stored in ctxTuple and called on watcher cleanup
	w.ctxTuple = CtxTuple{watcherCancelFunc, watcherCtx, w.config.NamespacedName}

	if w.config.CachePool == nil {
		// Own cache: start it in a background goroutine.
		cacheCtx, cacheCancelFunc := context.WithCancel(context.Background()) //nolint:gosec // G118 - cancel func stored in cacheCancelFunc and called on Clean
		w.cacheCancelFunc = cacheCancelFunc

		go func() {
			if err := w.cache.Start(cacheCtx); err != nil {
				w.config.Log.Errorw("could not start the watcher", tags.ErrorKey, err)
			}
		}()
	}
	// Shared cache: already started by the pool.

	// Build the SyncingSource.
	// No label-selector filter is applied here intentionally:
	//  - Own caches: the cache is already restricted to matching pods via CacheOptions,
	//    so only matching pod events arrive — filtering would be a no-op.
	//  - Shared namespace caches: filtering only the NEW labels would miss update events
	//    where a pod is relabeled OUT of the selector, preventing RemoveDeadTargets from
	//    running and leaving chaos pods on targets the user removed from the selector.
	//    Allowing all namespace events through is correct; the reconciler handles it.
	// The watcher context check stops enqueuing after the watcher is cleaned up.
	w.cacheSource = source.Kind(w.cache, w.config.ObjectType, handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, obj client.Object) []reconcile.Request {
		if w.ctxTuple.Ctx.Err() != nil {
			return nil
		}

		return []reconcile.Request{{NamespacedName: w.ctxTuple.NamespacedName}}
	}))

	return nil
}

// GetConfig return the configuration of the Watcher instance
func (w *watcher) GetConfig() WatcherConfig {
	return w.config
}

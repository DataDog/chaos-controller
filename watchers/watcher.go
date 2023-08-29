// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	k8sclientcache "k8s.io/client-go/tools/cache"
	controllerruntime "sigs.k8s.io/controller-runtime"
	k8scontrollercache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	// CacheOptions for configuring the cache.
	CacheOptions k8scontrollercache.Options

	// Handler function that will be called when an event occurs.
	Handler k8sclientcache.ResourceEventHandler

	// Log for logging.
	Log *zap.SugaredLogger

	// Name of the watcher instance.
	Name string

	// Namespace of the resource to watch.
	DisruptionNamespacedName types.NamespacedName

	// ObjectType of the object to watch.
	ObjectType client.Object
}

// CtxTuple is a struct that holds a context and its cancel function, as well as a NamespacedName that identifies a Disruption resource.
type CtxTuple struct {
	// CancelFunc is a function that can be called to cancel the associated context.
	CancelFunc context.CancelFunc

	// Ctx is a context.Context object that is used to manage the lifetime of an operation.
	Ctx context.Context

	// DisruptionNamespacedName is the namespaced name of a Disruption resource that is associated with this CtxTuple.
	DisruptionNamespacedName types.NamespacedName
}

// CacheContextFunc is a function that returns a context and a cancel function.
type CacheContextFunc func() (ctx context.Context, cancel context.CancelFunc)

// watcher represents a Kubernetes resource watcher, including its configuration, cache, cache source, and context tuple.
type watcher struct {
	// The Kubernetes resource cache that stores watched resources
	cache k8scontrollercache.Cache

	// The source that provides the cache with Kubernetes resource updates
	cacheSource source.SyncingSource

	// The configuration used to create the watcher
	config WatcherConfig

	// The context tuple that contains the context and cancellation function used to cancel the watcher
	ctxTuple CtxTuple
}

// NewWatcher is a function that creates a new watcher instance based on the given configuration values.
func NewWatcher(config WatcherConfig, cacheMock k8scontrollercache.Cache, cacheContextMockFunc CacheContextFunc) (Watcher, error) {
	watcherInstance := watcher{
		config: config,
	}

	// Used by unit test to allow mocking
	if cacheMock == nil {
		// create cache if it hasn't been set yet
		cache, err := k8scontrollercache.New(
			controllerruntime.GetConfigOrDie(),
			config.CacheOptions,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating cache: %w", err)
		}

		watcherInstance.cache = cache
	} else {
		watcherInstance.cache = cacheMock
	}

	// Used by unit test to allow mocking
	if cacheContextMockFunc != nil {
		cacheCtx, cacheCancelFunc := cacheContextMockFunc()
		watcherInstance.ctxTuple = CtxTuple{cacheCancelFunc, cacheCtx, config.DisruptionNamespacedName}
	}

	return &watcherInstance, nil
}

// Clean is a method that cancels the context of a watcher instance.
func (w *watcher) Clean() {
	w.ctxTuple.CancelFunc()
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
		return fmt.Errorf("error getting informer from cache. Error: %w", err)
	}

	// add event handler to informer
	_, err = info.AddEventHandler(w.config.Handler)
	if err != nil {
		return fmt.Errorf("error adding event handler to the informer. Error: %w", err)
	}

	// create context and cancel function for the watcher
	cacheCtx, cacheCancelFunc := context.WithCancel(context.Background())
	w.ctxTuple = CtxTuple{cacheCancelFunc, cacheCtx, w.config.DisruptionNamespacedName}

	// start the cache in a goroutine
	go func() {
		if err := w.cache.Start(cacheCtx); err != nil {
			w.config.Log.Errorw("could not start the watcher", "error", err)
		}
	}()

	// create a SyncingSource for the controller
	w.cacheSource = source.NewKindWithCache(w.config.ObjectType, w.cache)

	return nil
}

// GetConfig return the configuration of the Watcher instance
func (w *watcher) GetConfig() WatcherConfig {
	return w.config
}

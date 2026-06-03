// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/rest"
	k8scontrollercache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceCachePool manages one controller-runtime cache per target namespace.
// Multiple DisruptionTargetWatcher instances targeting the same namespace share a
// single cache, reducing the number of goroutines and Watch connections from
// O(disruptions) to O(namespaces).
type NamespaceCachePool struct {
	mu        sync.Mutex
	entries   map[string]*poolEntry
	k8sConfig *rest.Config
}

type poolEntry struct {
	cache    k8scontrollercache.Cache
	ctx      context.Context
	cancel   context.CancelFunc
	refCount int
}

// NewNamespaceCachePool creates an empty pool. k8sConfig is used when creating new
// namespace-scoped caches.
func NewNamespaceCachePool(k8sConfig *rest.Config) *NamespaceCachePool {
	return &NamespaceCachePool{
		entries:   make(map[string]*poolEntry),
		k8sConfig: k8sConfig,
	}
}

// GetOrCreate returns the shared cache for the given namespace, creating one if it
// does not yet exist. The caller must call Release when it no longer needs the cache.
// For cluster-scoped resources (nodes), pass namespace="" to get a cluster-wide cache.
func (p *NamespaceCachePool) GetOrCreate(namespace string, objectType client.Object) (k8scontrollercache.Cache, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if entry, ok := p.entries[namespace]; ok {
		entry.refCount++
		return entry.cache, nil
	}

	opts := k8scontrollercache.Options{}
	if namespace != "" {
		opts.ByObject = map[client.Object]k8scontrollercache.ByObject{
			objectType: {
				Namespaces: map[string]k8scontrollercache.Config{namespace: {}},
			},
		}
	}

	cache, err := k8scontrollercache.New(p.k8sConfig, opts)
	if err != nil {
		return nil, fmt.Errorf("namespace cache pool: failed to create cache for %q: %w", namespace, err)
	}

	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // G118 - cancel is stored and called on Release

	go func() {
		// Ignoring error: cache.Start only errors on context cancellation,
		// which is the normal clean shutdown path via Release().
		_ = cache.Start(ctx) //nolint:errcheck
	}()

	p.entries[namespace] = &poolEntry{
		cache:    cache,
		ctx:      ctx,
		cancel:   cancel,
		refCount: 1,
	}

	return cache, nil
}

// Release decrements the reference count for the given namespace. When it reaches
// zero the cache is stopped and removed from the pool.
func (p *NamespaceCachePool) Release(namespace string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.entries[namespace]
	if !ok {
		return
	}

	entry.refCount--

	if entry.refCount <= 0 {
		entry.cancel()
		delete(p.entries, namespace)
	}
}

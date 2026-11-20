// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	cLog "github.com/DataDog/chaos-controller/log"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type concurrencyWatcher struct {
	Watcher
	name    string
	start   func() error
	clean   func()
	expired func() bool
	cleaned atomic.Int32
}

func (w *concurrencyWatcher) GetName() string { return w.name }
func (w *concurrencyWatcher) Start() error {
	if w.start != nil {
		return w.start()
	}
	return nil
}
func (w *concurrencyWatcher) GetCacheSource() (source.SyncingSource, error) { return nil, nil }
func (w *concurrencyWatcher) GetContextTuple() (CtxTuple, error) {
	return CtxTuple{Ctx: context.Background()}, nil
}
func (w *concurrencyWatcher) IsExpired() bool {
	if w.expired != nil {
		return w.expired()
	}
	return true
}
func (w *concurrencyWatcher) Clean() {
	w.cleaned.Add(1)
	if w.clean != nil {
		w.clean()
	}
}

type concurrencyController struct{ controller.Controller }

func (concurrencyController) Watch(source.Source) error { return nil }

type concurrencyFactory struct {
	Factory
	create func(string) *concurrencyWatcher
}

func (f concurrencyFactory) NewDisruptionTargetWatcher(name string, _ bool, _ *v1beta1.Disruption, _ cache.Cache) (Watcher, error) {
	return f.create(name), nil
}

type concurrencyReader struct {
	client.Reader
	list func(context.Context, client.ObjectList, ...client.ListOption) error
	get  func(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error
}

func (r concurrencyReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if r.list != nil {
		return r.list(ctx, list, opts...)
	}
	return r.Reader.List(ctx, list, opts...)
}
func (r concurrencyReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if r.get != nil {
		return r.get(ctx, key, obj, opts...)
	}
	return r.Reader.Get(ctx, key, obj, opts...)
}

func waitConcurrency(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("operation blocked; possible lock held across callback")
	}
}

func TestManagerConcurrentChurn(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	reader := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewManager(reader, concurrencyController{})
	var created sync.Map
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				w := &concurrencyWatcher{name: fmt.Sprintf("watcher-%d", worker)}
				created.Store(w, true)
				if err := m.AddWatcher(w); err != nil {
					t.Error(err)
					return
				}
				m.GetWatcher(w.name)
				_ = m.RemoveWatcher(w)
			}
		}(worker)
	}
	for _, sweep := range []func(){m.RemoveExpiredWatchers, m.RemoveOrphanWatchers, m.RemoveAllWatchers} {
		wg.Add(1)
		go func(sweep func()) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				sweep()
			}
		}(sweep)
	}
	wg.Wait()
	m.RemoveAllWatchers()
	created.Range(func(key, _ interface{}) bool {
		if got := key.(*concurrencyWatcher).cleaned.Load(); got != 1 {
			t.Errorf("watcher cleaned %d times; want 1", got)
		}
		return true
	})
}

func TestDisruptionsManagerConcurrentChurn(t *testing.T) {
	ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
	scheme := runtime.NewScheme()
	if err := v1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	reader := fake.NewClientBuilder().WithScheme(scheme).Build()
	var created sync.Map
	factory := concurrencyFactory{create: func(name string) *concurrencyWatcher {
		w := &concurrencyWatcher{name: name}
		created.Store(w, true)
		return w
	}}
	d := NewDisruptionsWatchersManager(concurrencyController{}, factory, reader)
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				disruption := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("disruption-%d", worker), Namespace: "test", UID: types.UID(fmt.Sprint(i)),
				}}
				if err := d.CreateAllWatchers(ctx, disruption, nil, nil); err != nil {
					t.Error(err)
					return
				}
				if i%2 == 0 {
					d.RemoveAllWatchers(ctx, disruption)
				} else {
					d.RemoveWatchersForDisruption(ctx, getDisruptionNamespacedName(disruption))
				}
			}
		}(worker)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			d.RemoveAllExpiredWatchers(ctx)
			if err := d.RemoveAllOrphanWatchers(ctx); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()
	if err := d.RemoveAllOrphanWatchers(ctx); err != nil {
		t.Fatal(err)
	}
	created.Range(func(key, _ interface{}) bool {
		w := key.(*concurrencyWatcher)
		if got := w.cleaned.Load(); got != 1 {
			t.Errorf("watcher cleaned %d times; want 1", got)
		}
		return true
	})
}

func TestManagerRemovalDuringStartup(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("start-fails-%t", fail), func(t *testing.T) {
			entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			m := NewManager(nil, concurrencyController{})
			w := &concurrencyWatcher{name: "pending", start: func() error {
				close(entered)
				<-resume
				if fail {
					return errors.New("synthetic start failure")
				}
				return nil
			}}
			go func() {
				defer close(done)
				err := m.AddWatcher(w)
				if (err != nil) != fail {
					t.Errorf("unexpected start result: %v", err)
				}
			}()
			waitConcurrency(t, entered)
			m.RemoveExpiredWatchers()
			m.RemoveAllWatchers()
			replacement := &concurrencyWatcher{name: "pending"}
			if err := m.AddWatcher(replacement); err != nil {
				t.Fatal(err)
			}
			close(resume)
			waitConcurrency(t, done)
			if got := m.GetWatcher("pending"); got != replacement {
				t.Fatal("startup completion removed replacement")
			}
			want := int32(1)
			if fail {
				want = 0
			}
			if got := w.cleaned.Load(); got != want {
				t.Fatalf("pending cleaned %d times; want %d", got, want)
			}
			m.RemoveAllWatchers()
		})
	}
}

func TestManagerCallbacksOutsideMapLock(t *testing.T) {
	m := NewManager(nil, concurrencyController{})
	w := &concurrencyWatcher{name: "callback"}
	w.start = func() error { m.GetWatcher(w.name); return nil }
	w.expired = func() bool { m.GetWatcher(w.name); return true }
	w.clean = func() { m.GetWatcher(w.name) }
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := m.AddWatcher(w); err != nil {
			t.Error(err)
		}
		m.RemoveExpiredWatchers()
	}()
	waitConcurrency(t, done)
	if w.cleaned.Load() != 1 {
		t.Fatal("expected exactly one cleanup")
	}
}

func TestManagerStaleExpiryCannotRemoveReplacement(t *testing.T) {
	m := NewManager(nil, concurrencyController{})
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	old := &concurrencyWatcher{name: "same", expired: func() bool { close(entered); <-resume; return true }}
	if err := m.AddWatcher(old); err != nil {
		t.Fatal(err)
	}
	go func() { defer close(done); m.RemoveExpiredWatchers() }()
	waitConcurrency(t, entered)
	m.RemoveAllWatchers()
	replacement := &concurrencyWatcher{name: "same"}
	if err := m.AddWatcher(replacement); err != nil {
		t.Fatal(err)
	}
	close(resume)
	waitConcurrency(t, done)
	if m.GetWatcher("same") != replacement {
		t.Fatal("stale expiry removed replacement")
	}
	if old.cleaned.Load() != 1 || replacement.cleaned.Load() != 0 {
		t.Fatal("cleanup ownership violated")
	}
	m.RemoveAllWatchers()
}

func TestDisruptionsManagerOrphanSnapshotAndTargetedRemoval(t *testing.T) {
	ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	reader := concurrencyReader{list: func(context.Context, client.ObjectList, ...client.ListOption) error {
		close(entered)
		<-resume
		return nil
	}}
	var created []*concurrencyWatcher
	factory := concurrencyFactory{create: func(name string) *concurrencyWatcher {
		w := &concurrencyWatcher{name: name}
		created = append(created, w)
		return w
	}}
	d := NewDisruptionsWatchersManager(concurrencyController{}, factory, reader)
	disruption := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{Name: "same", Namespace: "test", UID: "old"}}
	if err := d.CreateAllWatchers(ctx, disruption, nil, nil); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(done)
		if err := d.RemoveAllOrphanWatchers(ctx); err != nil {
			t.Error(err)
		}
	}()
	waitConcurrency(t, entered)
	d.RemoveWatchersForDisruption(ctx, getDisruptionNamespacedName(disruption))
	disruption.UID = "new"
	if err := d.CreateAllWatchers(ctx, disruption, nil, nil); err != nil {
		t.Fatal(err)
	}
	close(resume)
	waitConcurrency(t, done)
	if created[0].cleaned.Load() != 1 || created[1].cleaned.Load() != 0 {
		t.Fatal("orphan snapshot removed replacement or repeated cleanup")
	}
	d.RemoveAllWatchers(ctx, disruption)
	if created[1].cleaned.Load() != 1 {
		t.Fatal("replacement was not retained")
	}
}

func TestDisruptionsManagerRemovalDuringCreation(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("start-fails-%t", fail), func(t *testing.T) {
			ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
			entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			w := &concurrencyWatcher{start: func() error {
				close(entered)
				<-resume
				if fail {
					return errors.New("synthetic start failure")
				}
				return nil
			}}
			factory := concurrencyFactory{create: func(name string) *concurrencyWatcher { w.name = name; return w }}
			d := NewDisruptionsWatchersManager(concurrencyController{}, factory, nil)
			disruption := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "test"}}
			go func() {
				defer close(done)
				err := d.CreateAllWatchers(ctx, disruption, nil, nil)
				if (err != nil) != fail {
					t.Errorf("unexpected create result: %v", err)
				}
			}()
			waitConcurrency(t, entered)
			d.RemoveAllExpiredWatchers(ctx)
			d.RemoveAllWatchers(ctx, disruption)
			close(resume)
			waitConcurrency(t, done)
			want := int32(1)
			if fail {
				want = 0
			}
			if w.cleaned.Load() != want {
				t.Fatalf("cleanup count %d; want %d", w.cleaned.Load(), want)
			}
		})
	}
}

func TestDisruptionsManagerCleanupOutsideMapLock(t *testing.T) {
	ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
	entered, resume, removed, created := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	old := &concurrencyWatcher{clean: func() { close(entered); <-resume }}
	replacement := &concurrencyWatcher{}
	next := old
	factory := concurrencyFactory{create: func(name string) *concurrencyWatcher { w := next; w.name = name; return w }}
	d := NewDisruptionsWatchersManager(concurrencyController{}, factory, nil)
	disruption := &v1beta1.Disruption{ObjectMeta: metav1.ObjectMeta{Name: "same", Namespace: "test"}}
	if err := d.CreateAllWatchers(ctx, disruption, nil, nil); err != nil {
		t.Fatal(err)
	}
	go func() { defer close(removed); d.RemoveAllWatchers(ctx, disruption) }()
	waitConcurrency(t, entered)
	next = replacement
	go func() {
		defer close(created)
		if err := d.CreateAllWatchers(ctx, disruption, nil, nil); err != nil {
			t.Error(err)
		}
	}()
	waitConcurrency(t, created)
	close(resume)
	waitConcurrency(t, removed)
	d.RemoveAllWatchers(ctx, disruption)
	if old.cleaned.Load() != 1 || replacement.cleaned.Load() != 1 {
		t.Fatal("cleanup ownership violated")
	}
}

func TestManagerOrphanReadOutsideMapLock(t *testing.T) {
	var m Manager
	reader := concurrencyReader{get: func(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
		m.GetWatcher("retained")
		return errors.New("synthetic API failure")
	}}
	m = NewManager(reader, concurrencyController{})
	w := &concurrencyWatcher{name: "retained"}
	if err := m.AddWatcher(w); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); m.RemoveOrphanWatchers() }()
	waitConcurrency(t, done)
	if m.GetWatcher(w.name) != w || w.cleaned.Load() != 0 {
		t.Fatal("API failure removed watcher")
	}
	m.RemoveAllWatchers()
}

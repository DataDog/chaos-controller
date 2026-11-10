// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scontrollercache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// stubManager is a minimal Manager implementation used to drive the disruptions
// watchers manager in tests without the informer/controller dependencies of a real
// per-disruption Manager.
type stubManager struct{}

func (stubManager) AddWatcher(_ Watcher) error    { return nil }
func (stubManager) RemoveWatcher(_ Watcher) error { return nil }
func (stubManager) RemoveExpiredWatchers()        {}
func (stubManager) GetWatcher(_ string) Watcher   { return nil }
func (stubManager) RemoveOrphanWatchers()         {}
func (stubManager) RemoveAllWatchers()            {}

// stubWatcher is a minimal Watcher implementation returned by stubFactory.
type stubWatcher struct{}

func (stubWatcher) Clean()                                        {}
func (stubWatcher) GetCacheSource() (source.SyncingSource, error) { return nil, nil }
func (stubWatcher) GetContextTuple() (CtxTuple, error)            { return CtxTuple{}, nil }
func (stubWatcher) GetName() string                               { return "stub" }
func (stubWatcher) IsExpired() bool                               { return false }
func (stubWatcher) Start() error                                  { return nil }
func (stubWatcher) GetConfig() WatcherConfig                      { return WatcherConfig{} }

// stubFactory is a minimal Factory implementation that avoids the nil factory
// pointer the production code would otherwise dereference.
type stubFactory struct{}

func (stubFactory) NewChaosPodWatcher(_ string, _ *v1beta1.Disruption, _ k8scontrollercache.Cache) (Watcher, error) {
	return stubWatcher{}, nil
}

func (stubFactory) NewDisruptionTargetWatcher(_ string, _ bool, _ *v1beta1.Disruption, _ k8scontrollercache.Cache) (Watcher, error) {
	return stubWatcher{}, nil
}

var _ = Describe("Disruptions watchers manager concurrency", func() {
	When("the expiry sweep runs while reconcile removes and re-adds watchers for a disruption", func() {
		It("must not race on the watchers manager map", func() {
			manager := NewDisruptionsWatchersManager(nil, stubFactory{}, nil).(disruptionsWatchersManager)

			// Disruptions whose watchers are continuously removed and re-created by reconcile
			const names = 20

			for i := range names {
				nn := types.NamespacedName{Namespace: "ns", Name: fmt.Sprintf("disruption-%d", i)}
				manager.watchersManagers[nn] = stubManager{}
				manager.managerUIDs[nn] = types.UID(fmt.Sprintf("uid-%d", i))
			}

			var wg sync.WaitGroup

			wg.Add(2)

			go func() { // background sweep goroutine (main.go ticker)
				defer wg.Done()

				for range 100 {
					manager.RemoveAllExpiredWatchers(context.Background())
				}
			}()

			go func() { // reconcile goroutine: NotFound cleanup + re-creation through production paths
				defer wg.Done()

				for i := range 100 {
					nn := types.NamespacedName{Namespace: "ns", Name: fmt.Sprintf("disruption-%d", i%names)}
					manager.RemoveWatchersForDisruption(context.Background(), nn)

					disruption := &v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      nn.Name,
							Namespace: nn.Namespace,
							UID:       types.UID(fmt.Sprintf("uid-%d", i%names)),
						},
					}

					Expect(manager.CreateAllWatchers(context.Background(), disruption, stubManager{}, nil)).To(Succeed())
				}
			}()

			wg.Wait()

			Expect(manager.watchersManagers).ToNot(BeEmpty())
		})
	})
})

// listingReader returns no disruptions and runs onList while the sweep is
// between its snapshot and its removals, as a concurrent reconcile would.
type listingReader struct {
	client.Reader
	onList func()
}

func (r listingReader) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	r.onList()
	return nil
}

var _ = Describe("Disruptions watchers manager orphan sweep", func() {
	When("reconcile creates a manager while the sweep lists disruptions", func() {
		It("does not remove the manager created after the sweep started", func() {
			created := types.NamespacedName{Namespace: "ns", Name: "created-during-sweep"}
			orphan := types.NamespacedName{Namespace: "ns", Name: "orphan"}

			var manager disruptionsWatchersManager

			manager = NewDisruptionsWatchersManager(nil, stubFactory{}, listingReader{onList: func() {
				manager.mu.Lock()
				defer manager.mu.Unlock()

				manager.watchersManagers[created] = stubManager{}
				manager.managerUIDs[created] = "uid-created"
			}}).(disruptionsWatchersManager)

			manager.watchersManagers[orphan] = stubManager{}
			manager.managerUIDs[orphan] = "uid-orphan"

			Expect(manager.RemoveAllOrphanWatchers(context.Background())).To(Succeed())

			Expect(manager.watchersManagers).ToNot(HaveKey(orphan))
			Expect(manager.watchersManagers).To(HaveKey(created))
		})
	})
})

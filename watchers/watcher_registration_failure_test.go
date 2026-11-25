// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers_test

import (
	"errors"

	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/watchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Watcher registration failure rollback", func() {
	DescribeTable("cleans an incompletely registered watcher and allows retry", func(sourceFails bool) {
		controller := mocks.NewRuntimeControllerMock[reconcile.Request](GinkgoT())
		manager := watchers.NewManager(nil, controller)
		source := mocks.NewSyncingSourceMock[reconcile.Request](GinkgoT())
		failure := errors.New("registration failed")
		first := watchers.NewWatcherMock(GinkgoT())
		first.EXPECT().GetName().Return("targets").Once()
		first.EXPECT().Start().Return(nil).Once()
		first.EXPECT().Clean().Once()
		if sourceFails {
			first.EXPECT().GetCacheSource().Return(nil, failure).Once()
		} else {
			first.EXPECT().GetCacheSource().Return(source, nil).Once()
			controller.EXPECT().Watch(source).Return(failure).Once()
		}
		Expect(manager.AddWatcher(first)).To(MatchError(ContainSubstring(failure.Error())))
		Expect(manager.GetWatcher("targets")).To(BeNil())

		second := watchers.NewWatcherMock(GinkgoT())
		second.EXPECT().GetName().Return("targets").Once()
		second.EXPECT().Start().Return(nil).Once()
		second.EXPECT().GetCacheSource().Return(source, nil).Once()
		controller.EXPECT().Watch(source).Return(nil).Once()
		Expect(manager.AddWatcher(second)).To(Succeed())
		Expect(manager.GetWatcher("targets")).To(Equal(second))
		second.EXPECT().Clean().Once()
		manager.RemoveAllWatchers()
	},
		Entry("GetCacheSource failure", true),
		Entry("controller.Watch failure", false),
	)
})

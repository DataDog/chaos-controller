// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import (
	"context"
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Install in watchers/target_observer_enable_flag_regression_test.go.
// The zero-value observer option is false. This also compiles on the original
// implementation, whose unconditional true argument violates the expectation.
func TestCreateAllWatchersPreservesDisabledObserver(t *testing.T) {
	disruption := &v1beta1.Disruption{
		ObjectMeta: metav1.ObjectMeta{Name: "experiment", Namespace: "targets"},
	}
	factory := NewFactoryMock(t)
	manager := NewManagerMock(t)
	watcher := NewWatcherMock(t)

	factory.EXPECT().NewDisruptionTargetWatcher(mock.Anything, false, disruption, nil).Return(watcher, nil).Once()
	manager.EXPECT().GetWatcher(mock.Anything).Return(nil).Once()
	manager.EXPECT().AddWatcher(watcher).Return(nil).Once()

	d := NewDisruptionsWatchersManager(nil, factory, nil, false)
	if err := d.CreateAllWatchers(context.Background(), disruption, manager, nil); err != nil {
		t.Fatal(err)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/cenkalti/backoff"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestCleanupReportsOnlyActualOutcomes(t *testing.T) {
	for _, reinjection := range []bool{false, true} {
		metricName := "MetricCleaned"
		if reinjection {
			metricName = "MetricCleanedForReinjection"
		}
		t.Run(metricName, func(t *testing.T) {
			oldLog, oldSink, oldInjectors := log, ms, injectors
			t.Cleanup(func() { log, ms, injectors = oldLog, oldSink, oldInjectors })
			log = zap.NewNop().Sugar()
			sink := metrics.NewSinkMock(t)
			ms = sink
			var outcomes []bool
			sink.On(metricName, mock.Anything, chaostypes.DisruptionKindNetworkDisruption, mock.Anything).
				Run(func(args mock.Arguments) { outcomes = append(outcomes, args.Get(0).(bool)) }).Return(nil)
			failed := injector.NewInjectorMock(t)
			failed.EXPECT().Clean().Return(backoff.Permanent(errors.New("cleanup failed"))).Once()
			healthy := injector.NewInjectorMock(t)
			healthy.EXPECT().Clean().Return(nil).Once()
			healthy.EXPECT().GetDisruptionKind().Return(chaostypes.DisruptionKindName(chaostypes.DisruptionKindNetworkDisruption)).Once()
			injectors = []targetInjector{{config: 0, Injector: failed}, {config: 1, Injector: healthy}}

			if clean(chaostypes.DisruptionKindNetworkDisruption, true, reinjection) {
				t.Fatal("cleanup should report the failed injector")
			}
			if want := []bool{false, true}; !reflect.DeepEqual(outcomes, want) {
				t.Fatalf("cleanup metric outcomes = %v; want %v (one result per injector)", outcomes, want)
			}
		})
	}
}

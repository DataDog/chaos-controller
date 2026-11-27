// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"errors"
	"os"
	"testing"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type deadlineProbe struct {
	calls int
}

func (p *deadlineProbe) Inject() error {
	p.calls++
	return errors.New("unexpected injection after deadline")
}

func (*deadlineProbe) GetDisruptionKind() chaostypes.DisruptionKindName {
	return chaostypes.DisruptionKindContainerFailure
}

func (*deadlineProbe) TargetName() string           { return "target" }
func (*deadlineProbe) UpdateConfig(injector.Config) {}
func (*deadlineProbe) Clean() error                 { return nil }

func TestInjectAndWaitDeadline(t *testing.T) {
	for _, test := range []struct {
		name         string
		expiresAfter time.Duration
		initialDelay time.Duration
		startDelay   time.Duration
	}{
		{name: "already expired", expiresAfter: -time.Second},
		{name: "expires during initial delay", expiresAfter: 10 * time.Millisecond, initialDelay: time.Hour},
		{name: "expires before synchronized start", expiresAfter: 10 * time.Millisecond, startDelay: time.Hour},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldLog, oldMetrics := log, ms
			oldArgs, oldInjectors := disruptionArgs, injectors
			oldReady, oldSignals := readyToInject, signals
			oldDeadline, oldParentPID := deadline, parentPID
			t.Cleanup(func() {
				log, ms = oldLog, oldMetrics
				disruptionArgs, injectors = oldArgs, oldInjectors
				readyToInject, signals = oldReady, oldSignals
				deadline, parentPID = oldDeadline, oldParentPID
			})

			log = zap.NewNop().Sugar()
			ms = noop.New(log)
			disruptionArgs = chaosapi.DisruptionArgs{
				Level:             chaostypes.DisruptionLevelPod,
				PulseInitialDelay: test.initialDelay,
			}
			if test.startDelay != 0 {
				disruptionArgs.NotInjectedBefore = time.Now().Add(test.startDelay)
			}
			readyToInject = true
			signals = make(chan os.Signal, 1)
			parentPID = 0
			deadline = time.Now().Add(test.expiresAfter)
			probe := &deadlineProbe{}
			injectors = []targetInjector{{config: 0, Injector: probe}}

			done := make(chan struct{})
			go func() {
				injectAndWait(&cobra.Command{Use: "container-failure"}, nil)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				signals <- os.Interrupt
				<-done
				t.Fatal("pre-injection wait did not stop at the deadline")
			}
			if probe.calls != 0 {
				t.Fatalf("deadline expired: injection attempts=%d", probe.calls)
			}
		})
	}
}

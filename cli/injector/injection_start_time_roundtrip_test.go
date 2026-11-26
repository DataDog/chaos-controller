// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"reflect"
	"testing"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestInjectionStartTimeRoundTrip(t *testing.T) {
	cmd := cpuPressureCmd
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	reset := func(t *testing.T) {
		t.Helper()
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			var err error
			if value, ok := flag.Value.(pflag.SliceValue); ok {
				err = value.Replace(nil)
			} else {
				err = flag.Value.Set(flag.DefValue)
			}
			if err != nil {
				t.Fatalf("reset %s: %v", flag.Name, err)
			}
			flag.Changed = false
		})
	}
	t.Cleanup(func() { reset(t) })
	for _, offset := range []time.Duration{250 * time.Millisecond, 1500 * time.Millisecond, 10 * time.Nanosecond, time.Second} {
		t.Run(offset.String(), func(t *testing.T) {
			reset(t)
			count := intstr.FromInt(1)
			cpuCount := intstr.FromString("50%")
			disruption := v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.NewTime(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC))},
				Spec: v1beta1.DisruptionSpec{
					Count: &count, Level: chaostypes.DisruptionLevelPod,
					Selector:    map[string]string{"app": "target"},
					CPUPressure: &v1beta1.CPUPressureSpec{Count: &cpuCount},
					Triggers:    &v1beta1.DisruptionTriggers{Inject: v1beta1.DisruptionTrigger{Offset: v1beta1.DisruptionDuration(offset.String())}},
				},
			}
			if err := disruption.Spec.Validate(); err != nil {
				t.Fatalf("input spec is invalid: %v", err)
			}
			wantStart := disruption.TimeToInject()
			// These are the same timing producer and common-argument serializer used by GenerateChaosPodsOfDisruption.
			common := chaosapi.DisruptionArgs{
				Level: disruption.Spec.Level, Kind: chaostypes.DisruptionKindCPUPressure,
				MetricsSink: "noop", NotInjectedBefore: wantStart,
			}
			args := common.CreateCmdArgs(disruption.Spec.CPUPressure.GenerateArgs())
			if err := cmd.ParseFlags(args[1:]); err != nil {
				t.Fatalf("parse generated arguments %q: %v", args, err)
			}
			parsedCount, err := cmd.Flags().GetString("count")
			if err != nil {
				t.Fatal(err)
			}
			parsedCPUCount := intstr.Parse(parsedCount)
			parsedCPU := v1beta1.CPUPressureSpec{Count: &parsedCPUCount}
			if !reflect.DeepEqual(parsedCPU, *disruption.Spec.CPUPressure) {
				t.Fatalf("CPU spec changed: got %#v, want %#v", parsedCPU, *disruption.Spec.CPUPressure)
			}
			if !disruptionArgs.NotInjectedBefore.Equal(wantStart) {
				t.Fatalf("start time changed: got %s, want %s (injection can start %s early)",
					disruptionArgs.NotInjectedBefore.Format(time.RFC3339Nano), wantStart.Format(time.RFC3339Nano), wantStart.Sub(disruptionArgs.NotInjectedBefore))
			}
		})
	}
}

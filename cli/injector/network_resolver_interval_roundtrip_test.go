// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/o11y/metrics/noop"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNetworkResolverIntervalRoundTrip(t *testing.T) {
	cmd := networkDisruptionCmd
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	reset := func(t *testing.T) {
		t.Helper()
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			var err error
			if value, ok := flag.Value.(pflag.SliceValue); ok {
				var defaults []string
				if flag.Name == "path" {
					defaults = []string{v1beta1.DefaultHTTPPathFilter}
				}
				err = value.Replace(defaults)
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
	for _, interval := range []time.Duration{time.Second, 1500 * time.Millisecond, 5 * time.Minute, time.Minute} {
		t.Run(interval.String(), func(t *testing.T) {
			reset(t)
			count := intstr.FromInt(1)
			percentage := 100
			disruption := v1beta1.Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "roundtrip", Namespace: "default", UID: "roundtrip-uid", CreationTimestamp: metav1.Now()},
				Spec: v1beta1.DisruptionSpec{
					Count: &count, Level: chaostypes.DisruptionLevelPod, Duration: "1m",
					Selector: map[string]string{"app": "target"},
					Network: &v1beta1.NetworkDisruptionSpec{
						Drop:         100,
						Hosts:        []v1beta1.NetworkDisruptionHostSpec{{Host: "192.0.2.1", Port: 443, Protocol: "tcp", Percentage: &percentage}},
						AllowedHosts: []v1beta1.NetworkDisruptionHostSpec{}, Services: []v1beta1.NetworkDisruptionServiceSpec{},
					},
				},
			}
			if err := disruption.Spec.Validate(); err != nil {
				t.Fatalf("input spec is invalid: %v", err)
			}
			// JSON keeps this regression compilable before HostResolveInterval exists
			// on the service config. The base ignores the missing transport field.
			encoded, err := json.Marshal(map[string]time.Duration{"HostResolveInterval": interval})
			if err != nil {
				t.Fatal(err)
			}
			var injectorConfig services.ChaosPodServiceInjectorConfig
			if err := json.Unmarshal(encoded, &injectorConfig); err != nil {
				t.Fatal(err)
			}
			svc, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
				Client: fake.NewClientBuilder().Build(), ChaosNamespace: "chaos-testing",
				MetricsSink: noop.New(zap.NewNop().Sugar()), Injector: injectorConfig,
			})
			if err != nil {
				t.Fatal(err)
			}
			// This invokes the real NetworkDisruptionSpec.GenerateArgs and common
			// argument assembly, then obtains the exact injector container argv.
			pods, err := svc.GenerateChaosPodsOfDisruption(&disruption, "target", "node", map[string]string{"app": "containerd://test"}, "192.0.2.2")
			if err != nil || len(pods) != 1 {
				t.Fatalf("generate pods: count=%d err=%v", len(pods), err)
			}
			args := svc.GetPodInjectorArgs(context.Background(), pods[0])
			if err := cmd.ParseFlags(args[1:]); err != nil {
				t.Fatalf("parse generated arguments %q: %v", args, err)
			}
			getInt := func(name string) int {
				value, err := cmd.Flags().GetInt(name)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			getUint := func(name string) uint {
				value, err := cmd.Flags().GetUint(name)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			getSlice := func(name string) []string {
				value, err := cmd.Flags().GetStringSlice(name)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			getArray := func(name string) []string {
				value, err := cmd.Flags().GetStringArray(name)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			hosts, err := v1beta1.NetworkDisruptionHostSpecFromString(getSlice("hosts"))
			if err != nil {
				t.Fatal(err)
			}
			allowed, err := v1beta1.NetworkDisruptionHostSpecFromString(getSlice("allowed-hosts"))
			if err != nil {
				t.Fatal(err)
			}
			selectedServices, err := v1beta1.NetworkDisruptionServiceSpecFromString(getSlice("services"))
			if err != nil {
				t.Fatal(err)
			}
			paths := v1beta1.HTTPPaths{}
			for _, path := range getArray("path") {
				paths = append(paths, v1beta1.HTTPPath(path))
			}
			parsed := v1beta1.NetworkDisruptionSpec{
				Hosts: hosts, AllowedHosts: allowed, Services: selectedServices,
				Drop: getInt("drop"), Corrupt: getInt("corrupt"), Duplicate: getInt("duplicate"),
				Delay: getUint("delay"), DelayJitter: getUint("delay-jitter"), BandwidthLimit: getInt("bandwidth-limit"),
				HTTP: &v1beta1.NetworkHTTPFilters{Methods: getArray("method"), Paths: paths},
			}
			want := disruption.Spec.Network.DeepCopy()
			want.HTTP = &v1beta1.NetworkHTTPFilters{Methods: v1beta1.HTTPMethods{}, Paths: v1beta1.HTTPPaths{v1beta1.DefaultHTTPPathFilter}}
			if !reflect.DeepEqual(parsed, *want) {
				t.Fatalf("network spec changed: got %#v, want %#v", parsed, *want)
			}
			got, err := cmd.Flags().GetDuration("host-resolve-interval")
			if err != nil {
				t.Fatal(err)
			}
			if got != interval {
				t.Fatalf("host resolve interval changed: got %s, want %s", got, interval)
			}
		})
	}
}

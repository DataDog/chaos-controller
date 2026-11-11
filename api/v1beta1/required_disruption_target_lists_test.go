// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
)

func TestDisruptionRequiresNonemptyFailureTargets(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		kind    string
		wantErr bool
	}{
		{"empty_disk_paths", `"diskFailure":{"paths":[]}`, true},
		{"empty_grpc_endpoints", `"grpc":{"port":50051,"endpoints":[]}`, true},
		{"one_disk_path", `"diskFailure":{"paths":["/data"]}`, false},
		{"one_grpc_endpoint", `"grpc":{"port":50051,"endpoints":[{"endpoint":"/example.Service/Get","error":"UNAVAILABLE","queryPercent":100}]}`, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// Decode JSON to reproduce explicitly present, non-nil empty arrays.
			data := fmt.Sprintf(`{"count":1,"selector":{"app":"target"},"level":"pod","duration":"1m",%s}`, testCase.kind)
			var spec v1beta1.DisruptionSpec
			if err := json.Unmarshal([]byte(data), &spec); err != nil {
				t.Fatal(err)
			}
			if err := spec.Validate(); (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() = %v; want error = %t", err, testCase.wantErr)
			}
		})
	}
}

func TestStandaloneFailureSpecsRequireTargets(t *testing.T) {
	for _, testCase := range []struct {
		name string
		spec interface{ Validate() error }
	}{
		{"nil_disk_paths", &v1beta1.DiskFailureSpec{}},
		{"empty_disk_paths", &v1beta1.DiskFailureSpec{Paths: []string{}}},
		{"nil_grpc_endpoints", v1beta1.GRPCDisruptionSpec{Port: 50051}},
		{"empty_grpc_endpoints", v1beta1.GRPCDisruptionSpec{Port: 50051, Endpoints: []v1beta1.EndpointAlteration{}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := testCase.spec.Validate(); err == nil {
				t.Fatal("Validate() accepted a disruption without any failure targets")
			}
		})
	}
}

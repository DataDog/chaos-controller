// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package builderstest_test

import (
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

func TestWithDisruptionKindDiskFullSetsEmptySpec(t *testing.T) {
	disruption := NewDisruptionBuilder().WithDisruptionKind(chaostypes.DisruptionKindDiskFull).Build()

	if disruption.Spec.DiskFull == nil {
		t.Fatal("expected Spec.DiskFull to be set")
	}

	if *disruption.Spec.DiskFull != (v1beta1.DiskFullSpec{}) {
		t.Errorf("expected empty DiskFullSpec (like every other kind), got %+v", *disruption.Spec.DiskFull)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"testing"
	"time"

	"github.com/DataDog/chaos-controller/injector"
)

// A disk pressure on a pod whose first container lacks the path builds no
// injector for that container, so the only injector belongs to the second
// target. Reinjection must give it the second target's refreshed
// configuration instead of indexing injectors by target position.
func TestUpdateInjectorConfigsFollowsSkippedTargets(t *testing.T) {
	skipped := injector.Config{DisruptionDeadline: time.Unix(1, 0)}
	target := injector.Config{DisruptionDeadline: time.Unix(2, 0)}

	injectorMock := injector.NewInjectorMock(t)
	injectorMock.EXPECT().UpdateConfig(target).Once()

	updateInjectorConfigs(
		[]targetInjector{{Injector: injectorMock, config: 1}},
		map[int]injector.Config{0: skipped, 1: target},
	)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package api

import (
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// DisruptionArgsGenerator generates args for the given disruption
type DisruptionArgsGenerator interface {
	GenerateArgs(level chaostypes.DisruptionLevel, containersID []string, sink string, dryRun bool) []string
}

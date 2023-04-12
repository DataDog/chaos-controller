// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package stress

//go:generate mockery --name=Stresser --filename=stress_mock.go

// Stresser is a component stresser
type Stresser interface {
	// Stress function should not be blocking (and should start goroutines by itself if needed)
	Stress(exit <-chan struct{})
}

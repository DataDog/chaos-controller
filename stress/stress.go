// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package stress

// Stresser is a component stresser
type Stresser interface {
	// Stress function should not be blocking (and should start goroutines by itself if needed)
	Stress(exit <-chan struct{})
}

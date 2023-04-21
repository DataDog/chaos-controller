// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package types

// SinkDriver represents a sink driver to use
type SinkDriver string

const (
	// SinkDriverDatadog is the Datadog driver
	SinkDriverDatadog SinkDriver = "datadog"

	// SinkDriverNoop is a noop driver mainly used for testing
	SinkDriverNoop SinkDriver = "noop"
)

// SinkApp is the application name using the sink
type SinkApp string

const (
	// SinkAppController is the chaos controller
	SinkAppController SinkApp = "chaos-controller"

	// SinkAppInjector is the chaos injector
	SinkAppInjector SinkApp = "chaos-injector"
)

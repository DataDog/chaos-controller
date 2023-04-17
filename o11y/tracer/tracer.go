// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package tracer

import (
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/tracer/datadog"
	"github.com/DataDog/chaos-controller/o11y/tracer/noop"
	"github.com/DataDog/chaos-controller/o11y/tracer/types"
)

// Sink describes a tracer
type Sink interface {
	GetSinkName() string
	Start()
	Stop()
}

// GetSink returns an initiated tracer sink
func GetSink(driver types.SinkDriver) (Sink, error) {
	switch driver {
	case types.SinkDriverDatadog:
		return datadog.New(), nil
	case types.SinkDriverNoop:
		return noop.New(), nil
	default:
		return nil, fmt.Errorf("unsupported tracer: %s", driver)
	}
}

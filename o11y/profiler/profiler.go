// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package profiler

import (
	"fmt"

	"github.com/DataDog/chaos-controller/o11y/profiler/datadog"
	"github.com/DataDog/chaos-controller/o11y/profiler/noop"
	"github.com/DataDog/chaos-controller/o11y/profiler/types"
	"go.uber.org/zap"
)

// Sink describes a profiler
type Sink interface {
	GetSinkName() string
	Stop()
}

// GetSink returns an initiated profiler sink
func GetSink(log *zap.SugaredLogger, driver types.SinkDriver) (Sink, error) {
	switch driver {
	case types.SinkDriverDatadog:
		return datadog.New(log)
	case types.SinkDriverNoop:
		return noop.New(log), nil
	default:
		return nil, fmt.Errorf("unsupported profiler: %s", driver)
	}
}

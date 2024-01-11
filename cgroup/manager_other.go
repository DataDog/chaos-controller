// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

//go:build !linux
// +build !linux

package cgroup

import (
	"errors"

	"go.uber.org/zap"
)

func newAllCGroupManager(cgroupFile string, cgroupMount string, log *zap.SugaredLogger) (allCGroupManager, error) {
	return nil, errors.New("not implemented")
}

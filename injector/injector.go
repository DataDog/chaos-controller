// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-fi-controller/container"
	"go.uber.org/zap"
)

// Injector represents a generic failure injector
type Injector struct {
	Log *zap.SugaredLogger
	UID string
}

// ContainerInjector represents an injector for containers
type ContainerInjector struct {
	Injector
	Container container.Container
}

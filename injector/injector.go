// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import (
	"github.com/DataDog/chaos-fi-controller/container"
	"go.uber.org/zap"
)

type Injector interface {
	Inject()
	Clean()
}

// injector represents a generic failure injector
type injector struct {
	log *zap.SugaredLogger
	uid string
}

// containerInjector represents an injector for containers
type containerInjector struct {
	injector
	container container.Container
}

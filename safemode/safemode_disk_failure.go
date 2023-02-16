// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package safemode

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DiskFailure struct {
	dis    v1beta1.Disruption
	client client.Client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *DiskFailure) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.dis = disruption
	sm.client = client
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package safemode

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Node struct {
	dis    v1beta1.Disruption
	client client.Client
}

// CreationSafetyNets Refer to safemode.Safemode interface for documentation
func (sm *Node) CreationSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}

	return safetyNetResponses, nil
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Node) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.dis = disruption
}

// GetKubeClient Refer to safemode.Safemode interface for documentation
func (sm *Node) GetKubeClient(client client.Client) {
	sm.client = client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *Node) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.GetTypeSpec(disruption)
	sm.GetKubeClient(client)
}

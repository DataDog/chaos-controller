// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package safemode

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Safemode interface {
	// Init will grab the necessary spec depending on the type (e.g. SafemodeNetwork will grab NetworkDisruptionSpec)
	// and grab the disruption itself for data such as the kubernetes namespace the disruption is running on
	// It will also grab the kube client for functions that require state information from k8s system
	Init(disruption v1beta1.Disruption, client client.Client)
}

// AddAllSafemodeObjects will populate a list of Safemode objects with Safemode's related to the disruptions described
// in the disruption in question. Once populated, the list can be traversed and run with the same function which applies
// all safety nets for each Safemode object.
func AddAllSafemodeObjects(disruption v1beta1.Disruption, k8sClient client.Client) []Safemode {
	safemodeList := []Safemode{}

	// first add generic safemode object
	safemodeGeneric := Generic{}
	safemodeGeneric.Init(disruption, k8sClient)
	safemodeList = append(safemodeList, &safemodeGeneric)

	if disruption.Spec.Network != nil {
		safemodeNetwork := Network{}
		safemodeNetwork.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeNetwork)
	}

	if disruption.Spec.DiskPressure != nil {
		safemodeDiskPressure := DiskPressure{}
		safemodeDiskPressure.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeDiskPressure)
	}

	if disruption.Spec.DiskFailure != nil {
		safemodeDiskFailure := DiskFailure{}
		safemodeDiskFailure.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeDiskFailure)
	}

	if disruption.Spec.ContainerFailure != nil {
		safemodeContainerFailure := ContainerFailure{}
		safemodeContainerFailure.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeContainerFailure)
	}

	if disruption.Spec.CPUPressure != nil {
		safemodeCPU := CPU{}
		safemodeCPU.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeCPU)
	}

	if disruption.Spec.DNS != nil {
		safemodeDNS := DNS{}
		safemodeDNS.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeDNS)
	}

	if disruption.Spec.GRPC != nil {
		safemodeGRPC := GRPC{}
		safemodeGRPC.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeGRPC)
	}

	if disruption.Spec.NodeFailure != nil {
		safemodeNode := Node{}
		safemodeNode.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeNode)
	}

	return safemodeList
}

// Reinit resets the saved disruption configs for each safety net in the case where the same disruption is updated with new parameters
func Reinit(safetyNets []Safemode, disruption v1beta1.Disruption, k8sClient client.Client) {
	for _, safetyNet := range safetyNets {
		safetyNet.Init(disruption, k8sClient)
	}
}

type Generic struct {
	dis    v1beta1.Disruption
	client client.Client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *Generic) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.dis = disruption
	sm.client = client
}

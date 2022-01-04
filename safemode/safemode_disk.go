// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package safemode

import (
	"context"
	"fmt"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Disk struct {
	dis         v1beta1.Disruption
	client      client.Client
}

// CreationSafetyNets Refer to safemode.Safemode interface for documentation
func (sm *Disk) CreationSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}
	// run through the list of safety nets
	if caught, err := sm.safetyNetSpecificContainDisk(); err != nil {
		return nil, err
	} else {
		if caught {
			safetyNetResponses = append(safetyNetResponses, " The specified disruption specifies containers to target on a disk disruption which will target ALL containers, not just the ones specified.")
		}
	}
	return safetyNetResponses, nil
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Disk) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.dis = disruption
}

// GetKubeClient Refer to safemode.Safemode interface for documentation
func (sm *Disk) GetKubeClient(client client.Client) {
	sm.client = client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *Disk) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.GetTypeSpec(disruption)
	sm.GetKubeClient(client)
}

// safetyNetSpecificContainDisk is the safety net regarding running a disk disruption (which hits all containers) while a user targets specific containers.
// This is a misunderstanding of the disk disruption as disk disruption disrupt all containers and a user asking to disrupt specific containers will have unforeseen consequences.
func (sm *Disk) safetyNetSpecificContainDisk() (bool, error) {
	if len(sm.dis.Spec.Containers) == 0 {
		// No specified containers in the disruption, safety net is avoided in this case
		return false, nil
	}

	if sm.dis.Spec.Level != chaostypes.DisruptionLevelPod {
		// Node level disruption should be clear because choosing specific containers has a null affect
		return false, nil
	}

	if sm.dis.Spec.Safemode.IgnoreSpecificContainDisk {
		return false,nil
	}

	pods := &corev1.PodList{}
	listOptions := &client.ListOptions{
		Namespace: sm.dis.ObjectMeta.Namespace,
		LabelSelector: labels.SelectorFromValidatedSet(sm.dis.Spec.Selector),
	}
	err := sm.client.List(context.Background(), pods, listOptions)
	if err != nil {
		return false, fmt.Errorf("error listing target pods: %w", err)
	}

	// if the number of containers in the targets != the number of containers described in the spec, safety net goes off
	for _, pod := range pods.Items {
		if len(pod.Spec.Containers) != len(sm.dis.Spec.Containers){
			return true, nil
		}
	}


	return false, nil
}

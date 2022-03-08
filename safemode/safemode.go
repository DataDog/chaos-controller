// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package safemode

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Safemode interface {
	// CreationSafetyNets will look at all the specific safety nets corresponding to the creation of a new disruption
	CheckInitialSafetyNets() ([]string, error)
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
		safemodeDisk := Disk{}
		safemodeDisk.Init(disruption, k8sClient)
		safemodeList = append(safemodeList, &safemodeDisk)
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

// CreationSafetyNets Refer to safemode.Safemode interface for documentation
func (sm *Generic) CheckInitialSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}

	if caught, err := sm.safetyNetCountNotTooLarge(); err != nil {
		return nil, err
	} else if caught {
		safetyNetResponses = append(safetyNetResponses, " The specified count represents a large percentage of targets in either the namespace or the kubernetes cluster")
	}

	return safetyNetResponses, nil
}

// safetyNetCountNotTooLarge is the safety net regarding the count of targets
// it will check against the number of targets being targeted and the number of targets in the k8s system
// > 66% of the k8s system being targeted warrants a safety check if we assume each of our targets are replicated
// at least twice. > 80% in a namespace also warrants a safety check as namespaces may be shared between services.
// returning true indicates the safety net caught something
func (sm *Generic) safetyNetCountNotTooLarge() (bool, error) {
	if sm.dis.Spec.Unsafemode != nil && sm.dis.Spec.Unsafemode.DisableCountTooLarge {
		return false, nil
	}

	userCount := sm.dis.Spec.Count
	totalCount := 0
	namespaceCount := 0
	namespaceThreshold := 0.8
	clusterThreshold := 0.66

	if sm.dis.Spec.Unsafemode.Config != nil && sm.dis.Spec.Unsafemode.Config.CountTooLarge != nil {
		if sm.dis.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold != 0 {
			namespaceThreshold = float64(sm.dis.Spec.Unsafemode.Config.CountTooLarge.NamespaceThreshold) / 100.0
		}
		if sm.dis.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold != 0 {
			clusterThreshold = float64(sm.dis.Spec.Unsafemode.Config.CountTooLarge.ClusterThreshold) / 100.0
		}
	}

	if sm.dis.Spec.Level == chaostypes.DisruptionLevelPod {
		pods := &corev1.PodList{}
		listOptions := &client.ListOptions{
			Namespace: sm.dis.ObjectMeta.Namespace,
		}

		err := sm.client.List(context.Background(), pods, listOptions)
		if err != nil {
			return false, fmt.Errorf("error listing target pods: %w", err)
		}

		namespaceCount = len(pods.Items)

		err = sm.client.List(context.Background(), pods)
		if err != nil {
			return false, fmt.Errorf("error listing target pods: %w", err)
		}

		totalCount = len(pods.Items)
	} else {
		nodes := &corev1.NodeList{}

		err := sm.client.List(context.Background(), nodes)
		if err != nil {
			return false, fmt.Errorf("error listing target pods: %w", err)
		}

		totalCount = len(nodes.Items)
	}

	userCountVal := float64(userCount.IntVal)

	if userCount.Type != intstr.Int {
		userCountInt, err := strconv.Atoi(strings.TrimSuffix(userCount.StrVal, "%"))
		if err != nil {
			return false, fmt.Errorf("failed to convert percentage to int: %w", err)
		}

		if namespaceCount != 0 {
			userCountVal = float64(userCountInt) / 100.0 * float64(namespaceCount)
		} else {
			userCountVal = float64(userCountInt) / 100.0 * float64(totalCount)
		}
	}

	// we check to see if the count represents > 80 percent of all pods in the existing namepsace
	// and if the count represents > 66 percent of all pods in the cluster (2/3s)
	if userCountVal/float64(namespaceCount) > namespaceThreshold {
		return true, nil
	}

	if userCountVal/float64(totalCount) > clusterThreshold {
		return true, nil
	}

	return false, nil
}

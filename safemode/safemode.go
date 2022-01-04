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
	CreationSafetyNets() ([]string, error)
	// GetTypeSpec will grab the necessary spec depending on the type (e.g. SafemodeNetwork will grab NetworkDisruptionSpec)
	// and grab the disruption itself for data such as the kubernetes namespace the disruption is running on
	GetTypeSpec(disruption v1beta1.Disruption)
	// GetKubeClient grab the kube client for functions that require state information from k8s system
	GetKubeClient(client client.Client)
	// Init will run GetTypeSpec and GetKubeClient together to cimplify further code
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

type Generic struct {
	dis    v1beta1.Disruption
	client client.Client
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Generic) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.dis = disruption
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Generic) GetKubeClient(client client.Client) {
	sm.client = client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *Generic) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.GetTypeSpec(disruption)
	sm.GetKubeClient(client)
}

// CreationSafetyNets Refer to safemode.Safemode interface for documentation
func (sm *Generic) CreationSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}

	if caught, err := sm.safetyNetCountNotTooLarge(); err != nil {
		return nil, err
	} else if caught {
		safetyNetResponses = append(safetyNetResponses, " The specified count represents a large percentage of targets in either the namespace or the kubernetes cluster")
	}

	if caught, err := sm.safetyNetSporadicTargets(); err != nil {
		return nil, err
	} else if caught {
		safetyNetResponses = append(safetyNetResponses, " The target environment's size is changing sporadically, this is a sign of instability and applying additional disruption can be dangerous")
	}

	return safetyNetResponses, nil
}

// safetyNetSporadicTargets is the safety net regarding sporadic targets
// in an environment which count is constantly changing, we are looking at an environment where targets are being
// destroyed and created frequently which is a very bad sign in terms of reliability and a applying further
// disruptions is probably not the best idea.
// In this function we run a check against the count of the target environment 3 times. If the count is different each of those
// 3 times, we assume sporadic behaviour of the target environment and raise a flag.
func (sm *Generic) safetyNetSporadicTargets() (bool, error) {
	if sm.dis.Spec.Safemode.IgnoreSporadicTargets {
		return false, nil
	}

	failures := 0
	oldTargetCount := -1

	for i := 0; i < 4; i++ {
		if sm.dis.Spec.Level == chaostypes.DisruptionLevelPod {
			pods := &corev1.PodList{}
			listOptions := &client.ListOptions{
				Namespace: sm.dis.ObjectMeta.Namespace,
			}
			err := sm.client.List(context.Background(), pods, listOptions)

			if err != nil {
				return false, fmt.Errorf("error listing target pods: %w", err)
			}

			if oldTargetCount == -1 {
				oldTargetCount = len(pods.Items)
			} else if oldTargetCount != len(pods.Items) {
				failures++
				oldTargetCount = len(pods.Items)
			}
		} else {
			nodes := &corev1.NodeList{}

			err := sm.client.List(context.Background(), nodes)
			if err != nil {
				return false, fmt.Errorf("error listing target pods: %w", err)
			}

			if oldTargetCount == -1 {
				oldTargetCount = len(nodes.Items)
			} else if oldTargetCount != len(nodes.Items) {
				failures++
				oldTargetCount = len(nodes.Items)
			}
		}
	}

	if failures < 3 {
		return false, nil
	}

	return true, nil
}

// safetyNetCountNotTooLarge is the safety net regarding the count of targets
// it will check against the number of targets being targeted and the number of targets in the k8s system
// > 66% of the k8s system being targeted warrants a safety check if we assume each of our targets are replicated
// at least twice. > 80% in a namespace also warrants a safety check as typically namespaces are shared between services.
// returning true indicates the safety net caught something
func (sm *Generic) safetyNetCountNotTooLarge() (bool, error) {
	if sm.dis.Spec.Safemode.IgnoreCountNotTooLarge {
		return false, nil
	}

	userCount := sm.dis.Spec.Count
	totalCount := 0
	namespaceCount := 0

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
	if userCountVal/float64(namespaceCount) > 0.8 {
		return true, nil
	}

	if userCountVal/float64(totalCount) > 0.66 {
		return true, nil
	}

	return false, nil
}

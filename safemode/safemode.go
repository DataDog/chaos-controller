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
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type Safemode interface {
	// CheckTypeSafetyNets will look at all the specific safety nets in place depending on the type of Disruption
	CheckTypeSafetyNets() ([]string, error)
	// GetTypeSpec will grab the necessary spec depending on the type (e.g. SafemodeNetwork will grab NetworkDisruptionSpec)
	// and grab the disruption itself for data such as the kubernetes namespace the disruption is running on
	GetTypeSpec(disruption v1beta1.Disruption)
	// GetKubeClient grab the kube client for functions that require state information from k8s system
	GetKubeClient(client client.Client)
}

func AddAllSafemodeObjects(disruption v1beta1.Disruption, k8sClient client.Client) []Safemode {
	safemodeList := []Safemode{}

	// first add generic safemode object
	safemodeGeneric := Generic{}
	safemodeGeneric.GetTypeSpec(disruption)
	safemodeGeneric.GetKubeClient(k8sClient)
	safemodeList = append(safemodeList, &safemodeGeneric)
	if disruption.Spec.Network != nil {

	}
	if disruption.Spec.DiskPressure != nil {

	}
	if disruption.Spec.ContainerFailure != nil {

	}
	if disruption.Spec.CPUPressure != nil {

	}
	if disruption.Spec.DNS != nil {

	}
	if disruption.Spec.GRPC != nil {

	}
	if disruption.Spec.NodeFailure != nil {

	}

	return safemodeList

}

type Generic struct {
	dis                    v1beta1.Disruption
	client                 client.Client
	countNotTooLargePassed bool
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Generic) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.dis = disruption
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Generic) GetKubeClient(client client.Client) {
	sm.client = client
}

// CheckTypeSafetyNets will apply generic safety net rules (rules on values not related specifically to any one type
// of disruption such as count).
func (sm *Generic) CheckTypeSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}
	if err := sm.safetyNetCountNotTooLarge(); err != nil {
		return nil, err
	} else {
		if sm.countNotTooLargePassed {
			safetyNetResponses = append(safetyNetResponses, " The specified count represents a large percentage of targets in either the namespace or the kubernetes cluster")
		}
	}

	return safetyNetResponses, nil
}

// safetyNetCountNotTooLarge is the safety net regarding the count of targets
// it will check against the number of targets being targeted and the number of targets in the k8s system
// > 66% of the k8s system being targeted warrants a safety check if we assume each of our targets are replicated
// at least twice. > 80% in a namespace also warrants a safety check as typically namespaces are shared between services.
// returning true indicates the safety net caught something
func (sm *Generic) safetyNetCountNotTooLarge() error {
	userCount := sm.dis.Spec.Count
	totalCount := 0
	namespaceCount := 0
	if sm.dis.Spec.Level == chaostypes.DisruptionLevelPod{
		pods := &corev1.PodList{}
		listOptions := &client.ListOptions{
			Namespace: sm.dis.ObjectMeta.Namespace,
		}

		err := sm.client.List(context.Background(), pods, listOptions)
		if err != nil {
			sm.countNotTooLargePassed = false
			return fmt.Errorf("error listing target pods: %w", err)
		}

		namespaceCount = len(pods.Items)
		err = sm.client.List(context.Background(), pods)
		if err != nil {
			sm.countNotTooLargePassed = false
			return fmt.Errorf("error listing target pods: %w", err)
		}

		totalCount = len(pods.Items)
	} else {
		nodes := &corev1.NodeList{}

		err := sm.client.List(context.Background(), nodes)
		if err != nil {
			sm.countNotTooLargePassed = false
			return fmt.Errorf("error listing target pods: %w", err)
		}

		totalCount = len(nodes.Items)
	}

	userCountVal := float64(userCount.IntVal)
	if userCount.Type != intstr.Int {
		userCountInt, err := strconv.Atoi(strings.TrimSuffix(userCount.StrVal, "%"))
		if err != nil {
			sm.countNotTooLargePassed = false
			return fmt.Errorf("failed to convert percentage to int: %w", err)
		}
		if namespaceCount != 0 {
			userCountVal = float64(userCountInt) / 100.0 * float64(namespaceCount)
		} else {
			userCountVal = float64(userCountInt) / 100.0 * float64(totalCount)
		}
	}

	// we check to see if the count represents > 80 percent of all pods in the existing namepsace
	// and if the count represents > 70 percent of all pods in the cluster
	if float64(userCountVal)/float64(namespaceCount) > 0.4 {
		sm.countNotTooLargePassed = true
		return nil
	}
	if float64(userCountVal)/float64(totalCount) > 0.66 {
		sm.countNotTooLargePassed = true
		return nil
	}

	sm.countNotTooLargePassed = false
	return nil
}

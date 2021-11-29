// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package safemode

import (
	"fmt"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Network struct {
	dis         *v1beta1.Disruption
	networkSpec *v1beta1.NetworkDisruptionSpec
	client      *client.Client
}

// CheckTypeSafetyNets Refer to safemode.Safemode interface for documentation
func (sm Network) CheckTypeSafetyNets() ([]string, error) {
	responses := []string{}
	// run through the list of safety nets
	caught, err := sm.safetyNetNoHostNoPort()
	if err != nil {
		return nil, fmt.Errorf("failed to check safetyNetNoHostNoPort")
	}
	if caught {
		responses = append(responses, "There exist a host with either no port or not hostname leading to ambiguity and larger scope of failure.")
	}
	return nil, nil
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm Network) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.networkSpec = disruption.Spec.Network
	sm.dis = &disruption
}

// GetKubeClient Refer to safemode.Safemode interface for documentation
func (sm Network) GetKubeClient(client client.Client) {
	sm.client = &client
}

// GenerateSafetyNetOutput Refer to safemode.Safemode interface for documentation
func (sm Network) GenerateSafetyNetOutput(spec v1beta1.DisruptionSpec) {
	return
}

// safetyNetNoHostNoPort is the safety net regarding missing host or port values.
// it will check against all defined hosts in the network disruption spec to see if any of them have a host or
// port missing. The more generic a hosts tuple is (Omitting fields such as port), the bigger the blast radius.
func (sm Network) safetyNetNoHostNoPort() (bool, error) {
	if sm.networkSpec == nil {
		return false, fmt.Errorf("network spec has not been set.")
	}

	for _, host := range sm.networkSpec.Hosts {
		if host.Port == 0 || host.Host == "" {
			return false, nil
		}
	}

	return true, nil
}

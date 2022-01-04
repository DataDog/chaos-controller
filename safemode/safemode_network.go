// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package safemode

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Network struct {
	dis         v1beta1.Disruption
	client      client.Client
}

// CreationSafetyNets Refer to safemode.Safemode interface for documentation
func (sm *Network) CreationSafetyNets() ([]string, error) {
	safetyNetResponses := []string{}
	// run through the list of initial safety nets
	if caught, err := sm.safetyNetNoHostNoPort(); err != nil {
		return nil, err
	} else {
		if caught {
			safetyNetResponses = append(safetyNetResponses, " The specified disruption contains a Host which only has either a port or a host. The more ambiguous, the larger the blast radius. ")
		}
	}
	return safetyNetResponses, nil
}

// GetTypeSpec Refer to safemode.Safemode interface for documentation
func (sm *Network) GetTypeSpec(disruption v1beta1.Disruption) {
	sm.dis = disruption
}

// GetKubeClient Refer to safemode.Safemode interface for documentation
func (sm *Network) GetKubeClient(client client.Client) {
	sm.client = client
}

// Init Refer to safemode.Safemode interface for documentation
func (sm *Network) Init(disruption v1beta1.Disruption, client client.Client) {
	sm.GetTypeSpec(disruption)
	sm.GetKubeClient(client)
}

// safetyNetNoHostNoPort is the safety net regarding missing host or port values.
// it will check against all defined hosts in the network disruption spec to see if any of them have a host or
// port missing. The more generic a hosts tuple is (Omitting fields such as port), the bigger the blast radius.
func (sm *Network) safetyNetNoHostNoPort() (bool, error) {
	if sm.dis.Spec.Safemode.IgnoreNoPortOrHost {
		return false,nil
	}

	for _, host := range sm.dis.Spec.Network.Hosts {
		if host.Port == 0 || host.Host == "" {
			return true, nil
		}
	}

	return false, nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
)

const (
	// FlowEgress is the string representation of network disruptions applied to outgoing packets
	FlowEgress = "egress"
	// FlowIngress is the string representation of network disruptions applied to incoming packets
	FlowIngress = "ingress"
	// this limitation does not come from TC itself but from the net scheduler of the kernel.
	// When not specifying an index for the hashtable created when we use u32 filters, the default id for this hashtable is 0x800.
	// However, the maximum id being 0xFFF, we can only have 2048 different ids, so 2048 tc filters with u32.
	// https://github.com/torvalds/linux/blob/v5.19/net/sched/cls_u32.c#L689-L690
	MaximumTCFilters = 2048
)

// NetworkDisruptionSpec represents a network disruption injection
// +ddmark:validation:AtLeastOneOf={BandwidthLimit,Drop,Delay,Corrupt,Duplicate}
type NetworkDisruptionSpec struct {
	// +nullable
	Hosts []NetworkDisruptionHostSpec `json:"hosts,omitempty"`
	// +nullable
	AllowedHosts []NetworkDisruptionHostSpec `json:"allowedHosts,omitempty"`
	// +nullable
	Services []NetworkDisruptionServiceSpec `json:"services,omitempty"`
	// +nullable
	Cloud *NetworkDisruptionCloudSpec `json:"cloud,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=100
	Drop int `json:"drop,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=100
	Duplicate int `json:"duplicate,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=100
	Corrupt int `json:"corrupt,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=60000
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=60000
	Delay uint `json:"delay,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=100
	DelayJitter uint `json:"delayJitter,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +ddmark:validation:Minimum=0
	BandwidthLimit int `json:"bandwidthLimit,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=65535
	// +nullable
	DeprecatedPort *int `json:"port,omitempty"`
	// +kubebuilder:validation:Enum=egress;ingress
	// +ddmark:validation:Enum=egress;ingress
	DeprecatedFlow string `json:"flow,omitempty"`
}

type NetworkDisruptionHostSpec struct {
	Host string `json:"host,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +ddmark:validation:Minimum=0
	// +ddmark:validation:Maximum=65535
	Port int `json:"port,omitempty"`
	// +kubebuilder:validation:Enum=tcp;udp;""
	// +ddmark:validation:Enum=tcp;udp;""
	Protocol string `json:"protocol,omitempty"`
	// +kubebuilder:validation:Enum=ingress;egress;""
	// +ddmark:validation:Enum=ingress;egress;""
	Flow string `json:"flow,omitempty"`
	// +kubebuilder:validation:Enum=new;est;""
	// +ddmark:validation:Enum=new;est;""
	ConnState string `json:"connState,omitempty"`
}

type NetworkDisruptionServiceSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +ddmark:validation:AtLeastOneOf={AWSServiceList}
type NetworkDisruptionCloudSpec struct {
	AWSServiceList *[]NetworkDisruptionCloudServiceSpec `json:"aws,omitempty"`
}

type NetworkDisruptionCloudServiceSpec struct {
	// +ddmark:validation:Required=true
	ServiceName string `json:"service"`
	// +kubebuilder:validation:Enum=tcp;udp;""
	// +ddmark:validation:Enum=tcp;udp;""
	Protocol string `json:"protocol,omitempty"`
	// +kubebuilder:validation:Enum=ingress;egress;""
	// +ddmark:validation:Enum=ingress;egress;""
	Flow string `json:"flow,omitempty"`
}

// Validate validates args for the given disruption
func (s *NetworkDisruptionSpec) Validate() (retErr error) {
	if k8sClient != nil {
		if err := validateServices(k8sClient, s.Services); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	for _, host := range s.Hosts {
		if err := host.Validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	for _, host := range s.AllowedHosts {
		if err := host.Validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	// ensure deprecated fields are not used
	if s.DeprecatedPort != nil {
		retErr = multierror.Append(retErr, fmt.Errorf("the port specification at the network disruption level is deprecated; apply to network disruption hosts instead"))
	}

	if s.DeprecatedFlow != "" {
		retErr = multierror.Append(retErr, fmt.Errorf("the flow specification at the network disruption level is deprecated; apply to network disruption hosts instead"))
	}

	return multierror.Prefix(retErr, "Network:")
}

// GenerateArgs generates injection or cleanup pod arguments for the given spec
func (s *NetworkDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"network-disruption",
		"--corrupt",
		strconv.Itoa(s.Corrupt),
		"--drop",
		strconv.Itoa(s.Drop),
		"--duplicate",
		strconv.Itoa(s.Duplicate),
		"--delay",
		strconv.Itoa(int(s.Delay)),
		"--delay-jitter",
		strconv.Itoa(int(s.DelayJitter)),
		"--bandwidth-limit",
		strconv.Itoa(s.BandwidthLimit),
	}

	// append hosts
	for _, host := range s.Hosts {
		args = append(args, "--hosts", fmt.Sprintf("%s;%d;%s;%s;%s", host.Host, host.Port, host.Protocol, host.Flow, host.ConnState))
	}

	// append allowed hosts
	for _, host := range s.AllowedHosts {
		args = append(args, "--allowed-hosts", fmt.Sprintf("%s;%d;%s;%s;%s", host.Host, host.Port, host.Protocol, host.Flow, host.ConnState))
	}

	// append services
	for _, service := range s.Services {
		args = append(args, "--services", fmt.Sprintf("%s;%s", service.Name, service.Namespace))
	}

	return args
}

// NetworkDisruptionHostSpecFromString parses the given hosts to host specs
// The expected format for hosts is <host>;<port>;<protocol>;<flow>;<connState>
func NetworkDisruptionHostSpecFromString(hosts []string) ([]NetworkDisruptionHostSpec, error) {
	var err error

	parsedHosts := []NetworkDisruptionHostSpec{}

	// parse given hosts
	for _, host := range hosts {
		port := 0
		protocol := ""
		flow := ""
		connState := ""

		// parse host with format <host>;<port>;<protocol>;<flow>;<connState>
		parsedHost := strings.SplitN(host, ";", 5)

		// cast port to int if specified
		if len(parsedHost) > 1 && parsedHost[1] != "" {
			port, err = strconv.Atoi(parsedHost[1])
			if err != nil {
				return nil, fmt.Errorf("unexpected port parameter in %s: %v", host, err)
			}
		}

		// get protocol if specified
		if len(parsedHost) > 2 {
			protocol = parsedHost[2]
		}

		// get flow if specified
		if len(parsedHost) > 3 && parsedHost[3] != "" {
			flow = parsedHost[3]
		}

		// get conn state if specified
		if len(parsedHost) > 4 && parsedHost[4] != "" {
			connState = parsedHost[4]
		}

		// generate host spec
		parsedHosts = append(parsedHosts, NetworkDisruptionHostSpec{
			Host:      parsedHost[0],
			Port:      port,
			Protocol:  protocol,
			Flow:      flow,
			ConnState: connState,
		})
	}

	return parsedHosts, nil
}

// NetworkDisruptionServiceSpecFromString parses the given services to service specs
// The expected format for services is <serviceName>;<serviceNamespace>
func NetworkDisruptionServiceSpecFromString(services []string) ([]NetworkDisruptionServiceSpec, error) {
	parsedServices := []NetworkDisruptionServiceSpec{}

	// parse given services
	for _, service := range services {
		// parse service with format <name>;<namespace>
		parsedService := strings.Split(service, ";")
		if len(parsedService) != 2 {
			return nil, fmt.Errorf("unexpected service format: %s", service)
		}

		// generate service spec
		parsedServices = append(parsedServices, NetworkDisruptionServiceSpec{
			Name:      parsedService[0],
			Namespace: parsedService[1],
		})
	}

	return parsedServices, nil
}

func (h NetworkDisruptionHostSpec) Validate() error {
	if h.Flow != "" {
		if h.Host == "" && h.Port == 0 {
			return errors.New("host or port fields must be set when the flow field is set")
		}
	}

	return nil
}

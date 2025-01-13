// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/hashicorp/go-multierror"
	v1 "k8s.io/api/core/v1"
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
	MaximumTCFilters             = 2048
	MaxNetworkPathCharacters     = 90
	MaxNetworkPaths              = 20
	MaxNetworkMethods            = 9
	DefaultHTTPPathFilter        = "/"
	HTTPMethodsFilterErrorPrefix = "the methods specification at the network disruption level is not valid; "
	HTTPPathsFilterErrorPrefix   = "the paths specification at the network disruption level is not valid; "
)

var allowedHTTPMethods = map[string]struct{}{
	http.MethodPost:    {},
	http.MethodGet:     {},
	http.MethodTrace:   {},
	http.MethodOptions: {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodConnect: {},
	http.MethodHead:    {},
	http.MethodDelete:  {},
}

type HTTPMethods []string

type HTTPPaths []HTTPPath

type HTTPPath string

// NetworkDisruptionSpec represents a network disruption injection
type NetworkDisruptionSpec struct {
	// +nullable
	Hosts []NetworkDisruptionHostSpec `json:"hosts,omitempty"`
	// +nullable
	AllowedHosts               []NetworkDisruptionHostSpec `json:"allowedHosts,omitempty"`
	DisableDefaultAllowedHosts bool                        `json:"disableDefaultAllowedHosts,omitempty"`
	// +nullable
	Services []NetworkDisruptionServiceSpec `json:"services,omitempty"`
	// +nullable
	Cloud *NetworkDisruptionCloudSpec `json:"cloud,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Drop int `json:"drop,omitempty" validate:"omitempty,gte=0,lte=100"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Duplicate int `json:"duplicate,omitempty" validate:"omitempty,gte=0,lte=100"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Corrupt int `json:"corrupt,omitempty" validate:"omitempty,gte=0,lte=100"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=60000
	Delay uint `json:"delay,omitempty" validate:"omitempty,gte=0,lte=60000"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	DelayJitter uint `json:"delayJitter,omitempty" validate:"omitempty,gte=0,lte=100"`
	// +kubebuilder:validation:Minimum=0
	BandwidthLimit int `json:"bandwidthLimit,omitempty" validate:"omitempty,gte=0"`
	// +nullable
	HTTP *NetworkHTTPFilters `json:"http,omitempty"`
}

// NetworkHTTPFilters contains http filters
type NetworkHTTPFilters struct {
	Methods HTTPMethods `json:"methods,omitempty"`
	Paths   HTTPPaths   `json:"paths,omitempty"`
}

type NetworkDisruptionHostSpec struct {
	Host string `json:"host,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port,omitempty" validate:"omitempty,gte=0,lte=65535"`
	// +kubebuilder:validation:Enum=tcp;udp;""
	Protocol string `json:"protocol,omitempty" validate:"omitempty,oneofci=udp tcp"`
	// +kubebuilder:validation:Enum=ingress;egress;""
	Flow string `json:"flow,omitempty" validate:"omitempty,oneofci=ingress egress"`
	// +kubebuilder:validation:Enum=new;est;""
	ConnState string `json:"connState,omitempty" validate:"omitempty,oneofci=new est"`
}

type NetworkDisruptionServiceSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	// +optional
	Ports []NetworkDisruptionServicePortSpec `json:"ports,omitempty"`
}

type NetworkDisruptionServicePortSpec struct {
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port,omitempty" validate:"omitempty,gte=0,lte=65535"`
}

type NetworkDisruptionCloudSpec struct {
	AWSServiceList     *[]NetworkDisruptionCloudServiceSpec `json:"aws,omitempty"`
	GCPServiceList     *[]NetworkDisruptionCloudServiceSpec `json:"gcp,omitempty"`
	DatadogServiceList *[]NetworkDisruptionCloudServiceSpec `json:"datadog,omitempty"`
}

type NetworkDisruptionCloudServiceSpec struct {
	// +kubebuilder:validation:Required
	ServiceName string `json:"service" validate:"required"`
	// +kubebuilder:validation:Enum=tcp;udp;""
	Protocol string `json:"protocol,omitempty" validate:"omitempty,oneofci=tcp udp"`
	// +kubebuilder:validation:Enum=ingress;egress;""
	Flow string `json:"flow,omitempty" validate:"omitempty,oneofci=ingress egress"`
	// +kubebuilder:validation:Enum=new;est;""
	ConnState string `json:"connState,omitempty" validate:"omitempty,oneofci=new est"`
}

func (p HTTPPath) validate() error {
	if len(p) > MaxNetworkPathCharacters {
		return fmt.Errorf("the paths specification at the network disruption level is not valid; should not exceed %d characters", MaxNetworkPathCharacters)
	}

	if regexp.MustCompile(`\s`).MatchString(string(p)) {
		return fmt.Errorf("the paths specification at the network disruption level is not valid; should not contains spaces")
	}

	if string(p[0]) != DefaultHTTPPathFilter {
		return fmt.Errorf("the paths specification at the network disruption level is not valid; should start with a /")
	}

	return nil
}

func (paths HTTPPaths) isNotDefault() bool {
	if len(paths) == 0 || len(paths) > 1 {
		return false
	}

	return paths[0] != DefaultHTTPPathFilter
}

func (h HTTPMethods) isNotEmpty() bool {
	return len(h) >= 1
}

// validate validates args for the given http filters.
func (s *NetworkHTTPFilters) validate() (retErr error) {
	retErr = s.validatePaths(retErr)

	retErr = s.validateMethods(retErr)

	return retErr
}

func (s *NetworkHTTPFilters) validateMethods(retErr error) error {
	if len(s.Methods) > MaxNetworkMethods {
		retErr = multierror.Append(retErr, fmt.Errorf(HTTPMethodsFilterErrorPrefix+"the number of methods must not be greater than %d; Number of methods: %d", MaxNetworkMethods, len(s.Methods)))

		return retErr
	}

	if len(s.Methods) == 0 {
		return retErr
	}

	visitedMethods := make(map[string]struct {
		count int
	})

	for _, method := range s.Methods {
		if _, ok := allowedHTTPMethods[method]; !ok {
			err := fmt.Errorf(HTTPMethodsFilterErrorPrefix+"should be a GET, DELETE, POST, PUT, HEAD, PATCH, CONNECT, OPTIONS or TRACE. Invalid value: %s", method)
			retErr = multierror.Append(retErr, err)

			continue
		}

		visitedMethod, isVisited := visitedMethods[method]
		if isVisited {
			visitedMethod.count++
			visitedMethods[method] = visitedMethod

			continue
		}

		visitedMethod.count++
		visitedMethods[method] = visitedMethod
	}

	for method, visitedMethod := range visitedMethods {
		if visitedMethod.count > 1 {
			retErr = multierror.Append(retErr, fmt.Errorf(HTTPMethodsFilterErrorPrefix+"should not contain duplicated methods. Count: %d; Method: %s", visitedMethod.count, method))
		}
	}

	return retErr
}

func (s *NetworkHTTPFilters) validatePaths(retErr error) error {
	if len(s.Paths) > MaxNetworkPaths {
		retErr = multierror.Append(retErr, fmt.Errorf(HTTPPathsFilterErrorPrefix+"the number of paths must not be greater than %d; Number of paths: %d", MaxNetworkPaths, len(s.Paths)))

		return retErr
	}

	if len(s.Paths) == 0 {
		return retErr
	}

	visitedPaths := make(map[HTTPPath]struct {
		count int
	})

	isMultiplePath := false
	if len(s.Paths) > 1 {
		isMultiplePath = true
	}

	for _, path := range s.Paths {
		visitedPath, isVisited := visitedPaths[path]
		if isVisited {
			visitedPath.count++
			visitedPaths[path] = visitedPath

			continue
		}

		visitedPath.count++
		visitedPaths[path] = visitedPath

		if isMultiplePath && path == DefaultHTTPPathFilter {
			retErr = multierror.Append(retErr, fmt.Errorf(HTTPPathsFilterErrorPrefix+"no needs to define other paths if the / path is defined because it already catches all paths"))
		}

		if err := path.validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	for path, visitedPath := range visitedPaths {
		if visitedPath.count > 1 {
			retErr = multierror.Append(retErr, fmt.Errorf(HTTPPathsFilterErrorPrefix+"should not contain duplicated paths. Count: %d; Path: %s", visitedPath.count, path))
		}
	}

	return retErr
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

	if s.Drop == 0 && s.Delay == 0 && s.BandwidthLimit == 0 && s.Corrupt == 0 && s.Duplicate == 0 {
		retErr = multierror.Append(retErr, fmt.Errorf("when applying a network disruption, at least one of network.drop, network.delay, network.corrupt, network.duplicate, or network.bandwidthLimit must be set"))
	}

	if s.HTTP != nil {
		if err := s.HTTP.validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	if s.BandwidthLimit > 0 && s.BandwidthLimit < 32 {
		retErr = multierror.Append(retErr, fmt.Errorf("bandwidthLimits below 32 bytes are not supported"))
	}

	if s.Cloud != nil {
		if err := s.Cloud.Validate(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
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
		ports := ""
		for _, port := range service.Ports {
			ports += fmt.Sprintf(";%d-%s", port.Port, port.Name)
		}

		args = append(args, "--services", fmt.Sprintf("%s;%s%s", service.Name, service.Namespace, ports))
	}

	if s.HTTP != nil {
		for _, path := range s.HTTP.Paths {
			args = append(args, "--path", string(path))
		}

		for _, method := range s.HTTP.Methods {
			args = append(args, "--method", method)
		}
	}

	return args
}

// Format describe a NetworkDisruptionSpec
func (s *NetworkDisruptionSpec) Format() string {
	networkVerbs := []string{}
	addOfWord := false // know whether or not we should suffix the verbs with a "of" word. example: delaying of 100ms the traffic vs dropping 100% of the traffic

	if s.Delay != 0 {
		networkVerbs = append(networkVerbs, fmt.Sprintf("delaying of %dms", s.Delay))
	}

	if s.Drop != 0 {
		addOfWord = true

		networkVerbs = append(networkVerbs, fmt.Sprintf("dropping %d%%", s.Drop))
	}

	if s.Duplicate != 0 {
		addOfWord = true

		networkVerbs = append(networkVerbs, fmt.Sprintf("duplicating %d%%", s.Duplicate))
	}

	if s.Corrupt != 0 {
		addOfWord = true

		networkVerbs = append(networkVerbs, fmt.Sprintf("corrupting %d%%", s.Corrupt))
	}

	if len(networkVerbs) == 0 {
		return ""
	}

	networkDescription := "Network disruption " + strings.Join(networkVerbs, ", ")

	if addOfWord {
		networkDescription += " of"
	}

	networkDescription += " the traffic"

	if s.DelayJitter != 0 {
		networkDescription += fmt.Sprintf(" with %dms of delay jitter", s.DelayJitter)
	}

	filterDescriptions := []string{}

	// Add host to description
	for _, host := range s.Hosts {
		descr := ""

		if host.Flow == FlowIngress {
			descr += " coming from "
		} else {
			descr += " going to "
		}

		descr += host.Host

		if host.Port != 0 {
			descr += fmt.Sprintf(":%d", host.Port)
		}

		if host.Protocol != "" {
			descr += fmt.Sprintf(" with protocol %s", host.Protocol)
		}

		filterDescriptions = append(filterDescriptions, descr)
	}

	// Add services to description
	for _, service := range s.Services {
		portsDescription := ""

		for _, port := range service.Ports {
			portsDescription = fmt.Sprintf("%s%s/%d,", portsDescription, port.Name, port.Port)
		}

		if len(service.Ports) > 0 {
			portsDescription = fmt.Sprintf(" on port(s) %s", portsDescription[:len(portsDescription)-1])
		}

		filterDescriptions = append(filterDescriptions, fmt.Sprintf(" going to %s/%s%s", service.Name, service.Namespace, portsDescription))
	}

	// Add cloud services to description
	if s.Cloud != nil {
		services := []NetworkDisruptionCloudServiceSpec{}

		if s.Cloud.AWSServiceList != nil {
			services = append(services, *s.Cloud.AWSServiceList...)
		}

		if s.Cloud.DatadogServiceList != nil {
			services = append(services, *s.Cloud.DatadogServiceList...)
		}

		if s.Cloud.GCPServiceList != nil {
			services = append(services, *s.Cloud.GCPServiceList...)
		}

		for _, service := range services {
			descr := ""

			if service.Flow == FlowIngress {
				descr += " coming from "
			} else {
				descr += " going to "
			}

			descr += service.ServiceName

			if service.Protocol != "" {
				descr += fmt.Sprintf(" with protocol %s", service.Protocol)
			}

			filterDescriptions = append(filterDescriptions, descr)
		}
	}

	networkDescription += strings.Join(filterDescriptions[:len(filterDescriptions)-1], ",")

	// Last filter uses and instead of a comma
	if len(filterDescriptions) > 1 {
		networkDescription += " and"
	}

	networkDescription += filterDescriptions[len(filterDescriptions)-1]

	return networkDescription
}

// HasHTTPFilters return true if a custom method or path is defined, else return false
func (s *NetworkDisruptionSpec) HasHTTPFilters() bool {
	return s.HTTP != nil && (s.HTTP.Methods.isNotEmpty() || s.HTTP.Paths.isNotDefault())
}

func (s *NetworkDisruptionCloudSpec) Validate() error {
	if s.GCPServiceList == nil && s.DatadogServiceList == nil && s.AWSServiceList == nil {
		return fmt.Errorf("if network.cloud is specified, at least one of cloud.aws, cloud.gcp, or cloud.datadog must be set")
	}

	return nil
}

// TransformToCloudMap for ease of computing when transforming the cloud services ip ranges to a list of hosts to disrupt
func (s *NetworkDisruptionCloudSpec) TransformToCloudMap() map[string][]NetworkDisruptionCloudServiceSpec {
	clouds := map[string][]NetworkDisruptionCloudServiceSpec{}

	if s.AWSServiceList != nil {
		clouds["AWS"] = *s.AWSServiceList
	}

	if s.GCPServiceList != nil {
		clouds["GCP"] = *s.GCPServiceList
	}

	if s.DatadogServiceList != nil {
		clouds["Datadog"] = *s.DatadogServiceList
	}

	return clouds
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
				return nil, fmt.Errorf("unexpected port parameter in %s: %w", host, err)
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
		// parse service with format <name>;<namespace>;<port-value>-<port-name>;<port-value>-<port-name>...
		parsedService := strings.Split(service, ";")
		if len(parsedService) < 2 {
			return nil, fmt.Errorf("service format is expected to follow '<name>;<namespace>;<port-value>-<port-name>;<port-value>-<port-name>', unexpected format detected: %s", service)
		}

		ports := []NetworkDisruptionServicePortSpec{}

		for _, unparsedPort := range parsedService[2:] {
			// <port-value>-<port-name>
			portValue, portName, ok := strings.Cut(unparsedPort, "-")
			if !ok {
				return nil, fmt.Errorf("service port format is expected to follow '<port-value>-<port-name>', unexpected format detected: %s", unparsedPort)
			}

			port, err := strconv.Atoi(portValue)
			if err != nil {
				return nil, fmt.Errorf("port format is expected to be a valid integer, unexpected format detected in service port: %s", unparsedPort)
			}

			ports = append(ports, NetworkDisruptionServicePortSpec{
				Port: port,
				Name: portName,
			})
		}

		// generate service spec
		parsedServices = append(parsedServices, NetworkDisruptionServiceSpec{
			Name:      parsedService[0],
			Namespace: parsedService[1],
			Ports:     ports,
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

func (s NetworkDisruptionServiceSpec) ExtractAffectedPortsInServicePorts(k8sService *v1.Service) ([]v1.ServicePort, []NetworkDisruptionServicePortSpec) {
	if len(s.Ports) == 0 {
		return k8sService.Spec.Ports, nil
	}

	servicePortsDic := map[string]v1.ServicePort{}
	goodPorts, notFoundPorts := []v1.ServicePort{}, []NetworkDisruptionServicePortSpec{}

	// Convert service ports from found k8s service to a dictionary in order to facilitate the filtering of the ports
	for _, port := range k8sService.Spec.Ports {
		servicePortsDic[fmt.Sprintf("port-%d", port.Port)] = port
		if port.Name != "" {
			servicePortsDic[fmt.Sprintf("name-%s", port.Name)] = port
		}
	}

	for _, allowedPort := range s.Ports {
		if allowedPort.Port != 0 {
			servicePort, ok := servicePortsDic[fmt.Sprintf("port-%d", allowedPort.Port)]

			if !ok || (allowedPort.Name != "" && allowedPort.Name != servicePort.Name) {
				notFoundPorts = append(notFoundPorts, allowedPort)

				continue
			}

			goodPorts = append(goodPorts, servicePort)
		} else if allowedPort.Name != "" {
			servicePort, ok := servicePortsDic[fmt.Sprintf("name-%s", allowedPort.Name)]

			if !ok || servicePort.Port == int32(allowedPort.Port) {
				notFoundPorts = append(notFoundPorts, allowedPort)

				continue
			}

			goodPorts = append(goodPorts, servicePort)
		}
	}

	return goodPorts, notFoundPorts
}

// UpdateHostsOnCloudDisruption from a cloud spec disruption, get all ip ranges of services provided and appends them into the s.Hosts slice
func (s *NetworkDisruptionSpec) UpdateHostsOnCloudDisruption(cloudManager cloudservice.CloudServicesProvidersManager) error {
	if s == nil || s.Cloud == nil {
		return nil
	}

	if s.Hosts == nil {
		s.Hosts = []NetworkDisruptionHostSpec{}
	}

	clouds := s.Cloud.TransformToCloudMap()

	for cloudName, serviceList := range clouds {
		var serviceListNames []string

		for _, service := range serviceList {
			serviceListNames = append(serviceListNames, service.ServiceName)
		}

		ipRangesPerService, err := cloudManager.GetServicesIPRanges(types.CloudProviderName(cloudName), serviceListNames)
		if err != nil {
			return err
		}

		for _, serviceSpec := range serviceList {
			for _, ipRange := range ipRangesPerService[serviceSpec.ServiceName] {
				host := NetworkDisruptionHostSpec{
					Host:      ipRange,
					Protocol:  serviceSpec.Protocol,
					Flow:      serviceSpec.Flow,
					ConnState: serviceSpec.ConnState,
				}

				s.Hosts = append(s.Hosts, host)
			}
		}
	}

	return nil
}

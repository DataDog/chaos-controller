package injector

import (
	"fmt"
	"net"
	"strings"

	v1 "k8s.io/api/core/v1"
)

// Handle tc filters logic for dynamic resolution of services, pods and nodes

// tcPriority the lowest priority set by tc automatically when adding a tc filter
var tcPriority = uint32(49149)

// NetworkEndpoint describes a parsed Kubernetes service, representing an (ip, port, protocol) tuple
type NetworkEndpoint struct {
	ip       *net.IPNet
	port     int
	protocol string
}

func (n NetworkEndpoint) String() string {
	ip := ""
	if n.ip != nil {
		ip = n.ip.String()
	}

	return fmt.Sprintf("ip=%s; port=%d; protocol=%s", ip, n.port, n.protocol)
}

type SimplifiedServicePort struct {
	port     int
	protocol string
}

func NewServicePorts(spList []v1.ServicePort) []SimplifiedServicePort {
	simplified := []SimplifiedServicePort{}

	for _, sp := range spList {
		simplified = append(simplified, NewServicePort(sp))
	}

	return simplified
}

func NewServicePort(sp v1.ServicePort) SimplifiedServicePort {
	return SimplifiedServicePort{
		port:     int(sp.TargetPort.IntVal),
		protocol: string(sp.Protocol),
	}
}

// TcFilter describes a tc filter, representing the service filtered and its priority
type TcFilter struct {
	Endpoint NetworkEndpoint
	Priority uint32 // one priority per tc filters applied, the priority is the same for all interfaces
}

func (i *networkDisruptionInjector) getNewPriority() uint32 {
	priority := uint32(0)

	i.tcFilterMutex.Lock()
	i.tcFilterPriority++
	priority = i.tcFilterPriority
	i.tcFilterMutex.Unlock()

	return priority
}

func (i *networkDisruptionInjector) findTcFilter(tcFilters []TcFilter, toFind TcFilter) int {
	for idx, tcFilter := range tcFilters {
		if tcFilter.Endpoint.String() == toFind.Endpoint.String() {
			return idx
		}
	}

	return -1
}

// addTcFilters adds a list of service tc filters on a list of interfaces
func (i *networkDisruptionInjector) addTcFilters(filters []TcFilter, interfaces []string, flowid string) ([]TcFilter, error) {
	builtServices := []TcFilter{}

	for _, filter := range filters {
		filter.Priority = i.getNewPriority()

		i.config.Log.Infow("found endpoint", "endpoint", filter.Endpoint.String())

		err := i.config.TrafficController.AddFilter(interfaces, "1:0", filter.Priority, 0, nil, filter.Endpoint.ip, 0, filter.Endpoint.port, filter.Endpoint.protocol, flowid)
		if err != nil {
			return nil, err
		}

		builtServices = append(builtServices, filter)
	}

	return builtServices, nil
}

// removeTcFilter delete tc filters using its priority
func (i *networkDisruptionInjector) removeTcFilter(interfaces []string, tcFilter TcFilter) error {
	for _, iface := range interfaces {
		if err := i.config.TrafficController.DeleteFilter(iface, tcFilter.Priority); err != nil {
			return err
		}
	}

	i.config.Log.Infow(fmt.Sprintf("deleted a tc filter on %s", tcFilter.Endpoint.String()), "interfaces", strings.Join(interfaces, ", "))

	return nil
}

// removeTcFiltersInList delete a list of tc filters inside of another list of tc filters
func (i *networkDisruptionInjector) removeTcFiltersInList(interfaces []string, tcFilters []TcFilter, tcFiltersToRemove []TcFilter) ([]TcFilter, error) {
	for _, serviceToRemove := range tcFiltersToRemove {
		if deletedIdx := i.findTcFilter(tcFilters, serviceToRemove); deletedIdx >= 0 {
			if err := i.removeTcFilter(interfaces, tcFilters[deletedIdx]); err != nil {
				return nil, err
			}

			tcFilters = append(tcFilters[:deletedIdx], tcFilters[deletedIdx+1:]...)
		}
	}

	return tcFilters, nil
}

// buildTcFiltersFromPod builds a list of tc filters per pod endpoint using the service ports
func (i *networkDisruptionInjector) buildTcFiltersFromPod(pod v1.Pod, servicePorts []SimplifiedServicePort) []TcFilter {
	// compute endpoint IP (pod IP)
	_, endpointIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", pod.Status.PodIP))

	endpointsToWatch := []TcFilter{}

	for _, port := range servicePorts {
		filter := TcFilter{
			Endpoint: NetworkEndpoint{
				ip:       endpointIP,
				port:     port.port,
				protocol: port.protocol,
			},
		}

		if i.findTcFilter(endpointsToWatch, filter) == -1 { // forbid duplication
			endpointsToWatch = append(endpointsToWatch, filter)
		}
	}

	return endpointsToWatch
}

// buildTcFiltersFromService builds a list of tc filters per service using the service ports
func (i *networkDisruptionInjector) buildTcFiltersFromService(service v1.Service, servicePorts []v1.ServicePort) []TcFilter {
	// compute service IP (cluster IP)
	_, serviceIP, _ := net.ParseCIDR(fmt.Sprintf("%s/32", service.Spec.ClusterIP))

	endpointsToWatch := []TcFilter{}

	for _, port := range servicePorts {
		filter := TcFilter{
			Endpoint: NetworkEndpoint{
				ip:       serviceIP,
				port:     int(port.Port),
				protocol: string(port.Protocol),
			},
		}

		if i.findTcFilter(endpointsToWatch, filter) == -1 { // forbid duplication
			endpointsToWatch = append(endpointsToWatch, filter)
		}
	}

	return endpointsToWatch
}

// handle updating of tc filters

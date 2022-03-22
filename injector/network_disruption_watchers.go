package injector

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

// Handle watcher logic for dynamic resolution of services, pods or nodes

type ServiceWatcher struct {
	isDummy      bool
	name         string
	namespace    string
	eventChannel <-chan watch.Event
	tcFilters    []TcFilter
}

type PodWatcher struct {
	isDummy        bool
	namespace      string
	ports          []SimplifiedServicePort
	selector       string
	eventChannel   <-chan watch.Event
	tcFilters      []TcFilter
	podsWithoutIPs []string
}

// watchChanges for every changes happening in the kubernetes destination resources (services, pods, nodes), we update the tc service filters
func (i *networkDisruptionInjector) watchChanges(serviceWatcher ServiceWatcher, podWatcher PodWatcher, interfaces []string, flowid string) {
	for {
		// We create the watcher channels when it's closed
		if !serviceWatcher.isDummy && serviceWatcher.eventChannel == nil {
			channel, err := i.newServiceWatcher(serviceWatcher.namespace, serviceWatcher.name)
			if err != nil {
				i.config.Log.Errorw("couldn't watch service", "error", err)

				return
			}

			serviceWatcher.eventChannel = channel
		}

		if !podWatcher.isDummy && podWatcher.eventChannel == nil {
			channel, err := i.newPodListWatcher(podWatcher.namespace, podWatcher.selector)
			if err != nil {
				i.config.Log.Errorw("couldn't watch service", "error", err)

				return
			}

			podWatcher.eventChannel = channel
		}

		select {
		case state := <-i.config.State.State:
			if state == Cleaned {
				return
			}
		case event, ok := <-serviceWatcher.eventChannel: // We have changes in the service watched
			if !ok { // channel is closed
				serviceWatcher.eventChannel = nil
			} else {
				i.config.Log.Infow(fmt.Sprintf("changes in service watched %s/%s", serviceWatcher.name, serviceWatcher.namespace), "eventType", event.Type)

				if err := i.updateTcFiltersOnServiceChange(event, &serviceWatcher, &podWatcher, interfaces, flowid); err != nil {
					i.config.Log.Errorf("couldn't apply changes to tc filters: %w... Rebuilding watcher", err)

					if _, err = i.removeTcFiltersInList(interfaces, serviceWatcher.tcFilters, serviceWatcher.tcFilters); err != nil {
						i.config.Log.Errorf("couldn't clean list of tc filters: %w", err)
					}

					serviceWatcher.eventChannel = nil // restart the watcher in case of error
					serviceWatcher.tcFilters = []TcFilter{}
				}
			}
		case event, ok := <-podWatcher.eventChannel: // We have changes in the pods watched
			if !ok { // channel is closed
				podWatcher.eventChannel = nil
			} else {
				i.config.Log.Infow("changes in pods watched", "eventType", event.Type)

				if err := i.updateTcFiltersOnPodsChange(event, &podWatcher, interfaces, flowid); err != nil {
					i.config.Log.Errorf("couldn't apply changes to tc filters: %w... Rebuilding watcher", err)

					if _, err = i.removeTcFiltersInList(interfaces, podWatcher.tcFilters, podWatcher.tcFilters); err != nil {
						i.config.Log.Errorf("couldn't clean list of tc filters: %w", err)
					}

					podWatcher.eventChannel = nil // restart the watcher in case of error
					podWatcher.tcFilters = []TcFilter{}
				}
			}
		}
	}
}

// Build watchers

func (i *networkDisruptionInjector) newPodListWatcher(namespace string, selector string) (<-chan watch.Event, error) {
	podsWatcher, err := i.config.K8sClient.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		i.config.Log.Errorf("error watching the list of pods with selector %s: %w", selector, err)

		return nil, err
	}

	i.config.Log.Infow("starting kubernetes pods watch", "podNamespace", namespace)
	return podsWatcher.ResultChan(), nil
}

func (i *networkDisruptionInjector) newServiceWatcher(namespace string, name string) (<-chan watch.Event, error) {
	serviceWatcher, err := i.config.K8sClient.CoreV1().Services(namespace).Watch(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("name=%s", name),
	})
	if err != nil {
		i.config.Log.Errorf("error watching the changes for the given kubernetes service (%s/%s): %w", namespace, name, err)

		return nil, err
	}

	i.config.Log.Infow("starting kubernetes service watch", "serviceName", name, "serviceNamespace", namespace)
	return serviceWatcher.ResultChan(), nil
}

// updatePodTcFiltersOnServiceChange on service changes, delete old filters with the wrong service ports and create new filters
func (i *networkDisruptionInjector) updatePodTcFiltersOnServiceChange(service v1.Service, podWatcher *PodWatcher, interfaces []string, flowid string) error {
	tcFiltersToCreate, finalTcFilters := []TcFilter{}, []TcFilter{}

	// Update selector and ports
	podWatcher.selector = labels.SelectorFromValidatedSet(service.Spec.Selector).String()

	podList, err := i.config.K8sClient.CoreV1().Pods(podWatcher.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: podWatcher.selector,
	})
	if err != nil {
		return fmt.Errorf("error watching the list of pods for the given kubernetes service (%s/%s): %w", service.Namespace, service.Name, err)
	}

	podWatcher.ports = NewServicePorts(service.Spec.Ports)

	// Update tc filters
	for _, pod := range podList.Items {
		if pod.Status.PodIP != "" { // pods without ip are newly created and will be picked up in the other watcher
			tcFiltersToCreate = append(tcFiltersToCreate, i.buildTcFiltersFromPod(pod, podWatcher.ports)...) // we build the updated list of tc filters
		}
	}

	// update the list of tc filters by deleting old ones not in the new list of tc filters and creating new tc filters
	for _, oldFilter := range podWatcher.tcFilters {
		if idx := i.findTcFilter(tcFiltersToCreate, oldFilter); idx >= 0 {
			finalTcFilters = append(finalTcFilters, oldFilter)
			tcFiltersToCreate = append(tcFiltersToCreate[:idx], tcFiltersToCreate[idx+1:]...)
		} else { // delete tc filters which are not in the updated list of tc filters
			if err := i.removeTcFilter(interfaces, oldFilter); err != nil {
				return err
			}
		}
	}

	createdTcFilters, err := i.addTcFilters(tcFiltersToCreate, interfaces, flowid)
	if err != nil {
		return err
	}

	podWatcher.tcFilters = append(finalTcFilters, createdTcFilters...)

	return nil
}

// updateTcFiltersOnServiceChange for every change happening in the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) updateTcFiltersOnServiceChange(event watch.Event, serviceWatcher *ServiceWatcher, podWatcher *PodWatcher, interfaces []string, flowid string) error {
	var err error

	if event.Type == watch.Error {
		return i.handleWatchError(event)
	}

	service, ok := event.Object.(*v1.Service)
	if !ok {
		return fmt.Errorf("couldn't watch service in namespace, invalid type of watched object received")
	}

	if err := i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	// Deal with changes in pods when there is a change in service.
	// e.g: service could have changed ports so we need to update all the filters on the pods to change the ports
	i.updatePodTcFiltersOnServiceChange(*service, podWatcher, interfaces, flowid)

	tcFiltersFromService := i.buildTcFiltersFromService(*service, service.Spec.Ports)

	switch event.Type {
	case watch.Added:
		createdTcFilters, err := i.addTcFilters(tcFiltersFromService, interfaces, flowid)
		if err != nil {
			return err
		}

		serviceWatcher.tcFilters = append(serviceWatcher.tcFilters, createdTcFilters...)
	case watch.Modified:
		if _, err := i.removeTcFiltersInList(interfaces, serviceWatcher.tcFilters, serviceWatcher.tcFilters); err != nil {
			return err
		}

		serviceWatcher.tcFilters, err = i.addTcFilters(tcFiltersFromService, interfaces, flowid)
		if err != nil {
			return err
		}
	case watch.Deleted:
		serviceWatcher.tcFilters, err = i.removeTcFiltersInList(interfaces, serviceWatcher.tcFilters, tcFiltersFromService)
		if err != nil {
			return err
		}
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// updateTcFiltersOnPodsChange for every changes happening in the pods related to the kubernetes service destination, we update the tc service filters
func (i *networkDisruptionInjector) updateTcFiltersOnPodsChange(event watch.Event, watcher *PodWatcher, interfaces []string, flowid string) error {
	var err error

	if event.Type == watch.Error {
		return i.handleWatchError(event)
	}

	pod, ok := event.Object.(*v1.Pod)
	if !ok {
		return fmt.Errorf("couldn't watch pods in namespace, invalid type of watched object received")
	}

	if err = i.config.Netns.Enter(); err != nil {
		return fmt.Errorf("unable to enter the given container network namespace: %w", err)
	}

	tcFiltersFromPod := i.buildTcFiltersFromPod(*pod, watcher.ports)

	switch event.Type {
	case watch.Added:
		// if the filter already exists, we do nothing
		if i.findTcFilter(watcher.tcFilters, tcFiltersFromPod[0]) >= 0 {
			break
		}

		if pod.Status.PodIP != "" {
			createdTcFilters, err := i.addTcFilters(tcFiltersFromPod, interfaces, flowid)
			if err != nil {
				return err
			}

			watcher.tcFilters = append(watcher.tcFilters, createdTcFilters...)
		} else {
			i.config.Log.Infow("newly created destination port has no IP yet, adding to the watch list of pods", "destinationPodName", pod.Name)

			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs, pod.Name)
		}
	case watch.Modified:
		// From the list of pods without IPs that has been added, we create the one that got the IP assigned
		podToCreateIdx := -1

		for idx, podName := range watcher.podsWithoutIPs {
			if podName == pod.Name && pod.Status.PodIP != "" {
				podToCreateIdx = idx

				break
			}
		}

		if podToCreateIdx > -1 {
			tcFilters, err := i.addTcFilters(tcFiltersFromPod, interfaces, flowid)
			if err != nil {
				return err
			}

			watcher.tcFilters = append(watcher.tcFilters, tcFilters...)
			watcher.podsWithoutIPs = append(watcher.podsWithoutIPs[:podToCreateIdx], watcher.podsWithoutIPs[podToCreateIdx+1:]...)
		}
	case watch.Deleted:
		watcher.tcFilters, err = i.removeTcFiltersInList(interfaces, watcher.tcFilters, tcFiltersFromPod)
		if err != nil {
			return err
		}
	}

	if err := i.config.Netns.Exit(); err != nil {
		return fmt.Errorf("unable to exit the given container network namespace: %w", err)
	}

	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package injector

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type KindWatcher string
type KindKubernetesWatcher string

const (
	ServiceWatcher KindWatcher = "serviceWatcher"
	PodWatcher     KindWatcher = "podWatcher"
	HostWatcher    KindWatcher = "hostWatcher"
	NodeWatcher    KindWatcher = "nodeWatcher"
)

const (
	ServiceK8sWatcher KindKubernetesWatcher = "serviceWatcher"
	PodK8sWatcher     KindKubernetesWatcher = "podWatcher"
	HostK8sWatcher    KindKubernetesWatcher = "hostWatcher"
	NodeK8sWatcher    KindKubernetesWatcher = "nodeWatcher"
)

// watcher A watcher will keep the information about what is watched
type watcher struct {
	kind KindWatcher

	serviceSpec *v1beta1.NetworkDisruptionServiceSpec
	podSpec     *v1beta1.NetworkDisruptionPodSpec
	nodeSpec    *v1beta1.NetworkDisruptionNodeSpec

	tcFilters          []tcServiceFilter
	kubernetesWatchers map[KindKubernetesWatcher]<-chan watch.Event // event listener matching by type
}

// watch watch changes in pods
func (i *networkDisruptionInjector) watchChanges(interfaces []string, flowid string) error {
	watchers := []watcher{}

	for _, serviceSpec := range i.spec.Services {
		serviceWatcher := watcher{
			kind:               ServiceWatcher,
			serviceSpec:        &serviceSpec,
			tcFilters:          []tcServiceFilter{},
			kubernetesWatchers: make(map[KindKubernetesWatcher]<-chan watch.Event),
		}

		serviceWatcher.kubernetesWatchers[ServiceK8sWatcher] = nil
		serviceWatcher.kubernetesWatchers[PodK8sWatcher] = nil

		watchers = append(watchers, serviceWatcher)
	}

	for _, podSpec := range i.spec.Pods {
		// retrieve serviceSpec
		_, err := i.config.K8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
			LabelSelector: podSpec.Selector.String(),
		})
		if err != nil {
			return fmt.Errorf("error getting the given kubernetes pods (%s): %w", podSpec.Selector.String(), err)
		}

		podWatcher := watcher{
			kind:               PodWatcher,
			podSpec:            &podSpec,
			tcFilters:          []tcServiceFilter{},
			kubernetesWatchers: make(map[KindKubernetesWatcher]<-chan watch.Event),
		}

		podWatcher.kubernetesWatchers[PodK8sWatcher] = nil

		watchers = append(watchers, podWatcher)
	}

	return nil
}

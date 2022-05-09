// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"fmt"

	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
)

const eventTemplate string = "[disruption %s] %s"
const containerErrorTemplate string = "Container %s is in a %s state: %s"

func (h DisruptionSelectorHandler) PostDisruptionStateEvent(obj runtime.Object, eventType string, eventMessage string) {
	errorMessage := fmt.Sprintf(eventTemplate, h.disruption.UID, eventMessage)

	h.reconciler.Recorder.Event(obj, corev1.EventTypeWarning, eventType, errorMessage)
}

func (h DisruptionSelectorHandler) getSentPodEvents(pod corev1.Pod) ([]corev1.Event, error) {
	fieldSelector := fields.Set{
		"involvedObject.kind": "Pod",
		"involvedObject.name": pod.Name,
		"source":              "disruption-controller",
	}

	eventList, err := h.reconciler.DirectClient.CoreV1().Events(pod.Namespace).List(
		context.Background(),
		v1.ListOptions{
			FieldSelector: fieldSelector.AsSelector().String(),
		})
	if err != nil {
		return nil, err
	}

	return eventList.Items, nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventbroadcaster

import (
	"regexp"
	"strings"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

func eventAggregatorMessage(event *v1.Event) string {
	return event.Message
}

func eventAggregatorKeyFuncForStatusChanges(event *v1.Event) (string, string) {
	if v1beta1.IsDisruptionEvent(*event, "Warning") {
		return record.EventAggregatorByReasonFunc(event)
	}

	r, err := regexp.Compile(`(\((\w* ?)*)( (?P<disruptionName>(.*)):) (?P<message>.*)`)
	if err != nil {
		return record.EventAggregatorByReasonFunc(event)
	}

	match := r.FindStringSubmatch(event.Message)

	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i < len(match) && name != "" {
			result[name] = match[i]
		}
	}

	if result["disruptionName"] != "" {
		return strings.Join([]string{
				event.Source.Component,
				event.Source.Host,
				event.InvolvedObject.Kind,
				event.InvolvedObject.Namespace,
				result["disruptionName"],
				event.InvolvedObject.APIVersion,
				event.Type,
				event.Reason,
			},
				""), strings.Join([]string{
				event.InvolvedObject.Name,
				event.Message,
			}, "")
	}

	return record.EventAggregatorByReasonFunc(event)
}

func EventBroadcaster() record.EventBroadcaster {
	correlator := record.CorrelatorOptions{
		MaxEvents:            2,
		MaxIntervalInSeconds: 120,
		//		MessageFunc:          eventAggregatorMessage,
		KeyFunc: eventAggregatorKeyFuncForStatusChanges,
	}
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(correlator)

	return eventBroadcaster
}

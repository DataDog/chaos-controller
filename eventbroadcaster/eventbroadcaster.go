// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventbroadcaster

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

func eventMessageAggregator(event *v1.Event) string {
	return "Agg: " + event.Message
}

func EventBroadcaster() record.EventBroadcaster {
	correlator := record.CorrelatorOptions{
		MaxEvents:            2,
		MaxIntervalInSeconds: 60,
		MessageFunc:          eventMessageAggregator,
	}
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(correlator)

	return eventBroadcaster
}

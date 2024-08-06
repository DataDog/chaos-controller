// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package eventbroadcaster

import (
	"k8s.io/client-go/tools/record"
)

func EventBroadcaster() record.EventBroadcaster {
	correlator := record.CorrelatorOptions{
		MaxEvents:            2,
		MaxIntervalInSeconds: 300,
	}
	eventBroadcaster := record.NewBroadcasterWithCorrelatorOptions(correlator)

	return eventBroadcaster
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventsinks

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// type EventSink interface {
// 	Create(event *v1.Event) (*v1.Event, error)
// 	Update(event *v1.Event) (*v1.Event, error)
// 	Patch(oldEvent *v1.Event, data []byte) (*v1.Event, error)
// }

type EventSink struct{}

func (e *EventSink) Create(event *v1.Event) (*v1.Event, error) {
	fmt.Println("NOOPEVENTSINK CALLED CREATE")
	fmt.Printf("\t%v\n", event)
	return event, nil
}

func (e *EventSink) Update(event *v1.Event) (*v1.Event, error) {
	fmt.Println("NOOPEVENTSINK CALLED UPDATE")
	return event, nil
}

func (e *EventSink) Patch(oldEvent *v1.Event, data []byte) (*v1.Event, error) {
	fmt.Println("NOOPEVENTSINK CALLED PATCH")
	return oldEvent, nil
}

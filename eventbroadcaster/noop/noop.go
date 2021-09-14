// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package noop

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NoopSink struct {
	Client client.Client
}

func (e *NoopSink) Create(event *v1.Event) (*v1.Event, error) {
	fmt.Printf("%v - %v - %v - %v\n", "NOOPEVENTSINK CALLED CREATE", event.Reason, event.Count, event.Message)

	dis := &v1beta1.Disruption{}
	namespaceName := types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}

	err := e.Client.Get(context.Background(), namespaceName, dis)
	if err != nil {
		return event, err
	}

	_, err = fmt.Printf("Userinfo: %v\n", dis.Status.UserInfo)
	if err != nil {
		return event, err
	}

	return event, nil
}

func (e *NoopSink) Update(event *v1.Event) (*v1.Event, error) {
	fmt.Println("NOOPEVENTSINK CALLED UPDATE")
	return event, nil
}

func (e *NoopSink) Patch(oldEvent *v1.Event, data []byte) (*v1.Event, error) {
	fmt.Printf("%v - %v - %v - %v\n", "NOOPEVENTSINK CALLED PATCH", oldEvent.Reason, oldEvent.Count, oldEvent.Message)
	fmt.Printf("\t%v\n", data)
	return oldEvent, nil
}

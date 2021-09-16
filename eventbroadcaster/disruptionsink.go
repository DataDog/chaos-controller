// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package eventbroadcaster

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier"

	chaostypes "github.com/DataDog/chaos-controller/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DisruptionNotifierSink struct {
	Client   client.Client
	Notifier eventnotifier.Notifier
}

func (s *DisruptionNotifierSink) Create(event *v1.Event) (*v1.Event, error) {
	fmt.Printf("%v - %v - %v - %v\n", "NOOPEVENTSINK CALLED CREATE", event.Reason, event.Count, event.Message)
	dis, err := s.getDisruption(event)

	if err != nil {
		return event, nil
	}

	err = s.parseEvent(event, dis)

	if err != nil {
		return event, nil
	}

	return event, nil
}

func (s *DisruptionNotifierSink) Update(event *v1.Event) (*v1.Event, error) {
	fmt.Println("NOOPEVENTSINK CALLED UPDATE")
	return event, nil
}

func (s *DisruptionNotifierSink) Patch(oldEvent *v1.Event, data []byte) (*v1.Event, error) {
	fmt.Printf("%v - %v - %v - %v\n", "NOOPEVENTSINK CALLED PATCH", oldEvent.Reason, oldEvent.Count, oldEvent.Message)
	fmt.Printf("\t%v\n", data)
	return oldEvent, nil
}

func (s *DisruptionNotifierSink) getDisruption(event *v1.Event) (v1beta1.Disruption, error) {

	dis := v1beta1.Disruption{}
	err := s.Client.Get(context.Background(), types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, &dis)
	if err != nil {
		return v1beta1.Disruption{}, err
	}

	_, err = fmt.Printf("Userinfo: %v\n", dis.Status.UserInfo)
	if err != nil {
		return v1beta1.Disruption{}, err
	}

	return dis, nil
}

func (s *DisruptionNotifierSink) parseEvent(event *v1.Event, dis v1beta1.Disruption) error {
	switch event.Reason {
	case string(chaostypes.DisruptionInjectionStatusNotInjected):
		s.Notifier.NotifyNotInjected(dis)
	case string(chaostypes.DisruptionInjectionStatusInjected):
		s.Notifier.NotifyInjected(dis)
	case "CleanedUp":
		s.Notifier.NotifyCleanedUp(dis)
	case "NoTarget":
		s.Notifier.NotifyNoTarget(dis)
	case "StuckOnRemoval":
		s.Notifier.NotifyStuckOnRemoval(dis)
	default:
		return fmt.Errorf("event: not a chaos disruption event")
	}

	return nil
}

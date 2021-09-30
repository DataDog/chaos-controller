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
	notifiertypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotifierSink struct {
	client   client.Client
	notifier eventnotifier.Notifier
}

// RegisterNotifierSinks builds notifiers sinks and registers them on the given broadcaster
func RegisterNotifierSinks(mgr ctrl.Manager, broadcaster record.EventBroadcaster, filePath string, driverTypes ...string) error {
	var resError error = nil
	client := mgr.GetClient()

	for _, driver := range driverTypes {
		ndriver := notifiertypes.NotifierDriver(driver)
		notifier, err := eventnotifier.GetNotifier(ndriver, filePath)
		if err != nil {
			if resError == nil {
				resError = err
			} else {
				resError = fmt.Errorf("%w; "+err.Error(), resError)
			}
		} else {
			broadcaster.StartRecordingToSink(&NotifierSink{client: client, notifier: notifier})
		}
	}

	return resError
}

func (s *NotifierSink) Create(event *corev1.Event) (*corev1.Event, error) {
	dis, err := s.getDisruption(event)

	if err != nil {
		return event, nil
	}

	err = s.parseEventToNotifier(event, dis)

	if err != nil {
		fmt.Println(err)
		return event, nil
	}

	return event, nil
}

func (s *NotifierSink) Update(event *corev1.Event) (*corev1.Event, error) {
	return event, nil
}

func (s *NotifierSink) Patch(oldEvent *corev1.Event, data []byte) (*corev1.Event, error) {
	return oldEvent, nil
}

// getDisruption fetches the disruption object of the event from the controller-runtime client
func (s *NotifierSink) getDisruption(event *corev1.Event) (v1beta1.Disruption, error) {
	dis := v1beta1.Disruption{}
	if event.InvolvedObject.Kind != "Disruption" {
		return v1beta1.Disruption{}, fmt.Errorf("eventnotifier: not a disruption")
	}

	err := s.client.Get(context.Background(), types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, &dis)
	if err != nil {
		return v1beta1.Disruption{}, err
	}

	if dis.Status.UserInfo == nil {
		return v1beta1.Disruption{}, fmt.Errorf("eventnotifier: no userinfo in disruption %v", dis)
	}

	return dis, nil
}

// parseEventToNotifier contains the event parsing and notification logic
func (s *NotifierSink) parseEventToNotifier(event *corev1.Event, dis v1beta1.Disruption) (err error) {
	switch event.Type {
	case corev1.EventTypeWarning:
		err = s.notifier.NotifyWarning(dis, *event)
	case corev1.EventTypeNormal:
		err = nil
	default:
		err = fmt.Errorf("event: not a notifiable event")
	}

	return
}

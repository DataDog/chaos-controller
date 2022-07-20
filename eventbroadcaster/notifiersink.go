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
	notifTypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"

	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotifierSink struct {
	client   client.Client
	notifier eventnotifier.Notifier
	logger   *zap.SugaredLogger
}

// RegisterNotifierSinks builds notifiers sinks and registers them on the given broadcaster
func RegisterNotifierSinks(mgr ctrl.Manager, broadcaster record.EventBroadcaster, notifiersConfig eventnotifier.NotifiersConfig, logger *zap.SugaredLogger) (err error) {
	err = nil

	client := mgr.GetClient()

	notifiers, err := eventnotifier.GetNotifiers(notifiersConfig, logger)

	for _, notifier := range notifiers {
		logger.Infof("notifier %s enabled", notifier.GetNotifierName())

		broadcaster.StartRecordingToSink(&NotifierSink{client: client, notifier: notifier, logger: logger})
	}

	return
}

func (s *NotifierSink) Create(event *corev1.Event) (*corev1.Event, error) {
	dis, err := s.getDisruption(event)
	if err != nil {
		return event, nil
	}

	if err = s.parseEventToNotifier(event, dis); err != nil {
		s.logger.Error(err)
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

	if err := s.client.Get(context.Background(), types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, &dis); err != nil {
		return v1beta1.Disruption{}, err
	}

	return dis, nil
}

// parseEventToNotifier contains the event parsing and notification logic
func (s *NotifierSink) parseEventToNotifier(event *corev1.Event, dis v1beta1.Disruption) (err error) {
	switch event.Type {
	case corev1.EventTypeWarning:
		err = s.notifier.Notify(dis, *event, notifTypes.NotificationWarning)
	case corev1.EventTypeNormal:
		if v1beta1.IsNotifiableEvent(*event) {
			if v1beta1.IsRecoveryEvent(*event) {
				err = s.notifier.Notify(dis, *event, notifTypes.NotificationSuccess)
			} else {
				err = s.notifier.Notify(dis, *event, notifTypes.NotificationInfo)
			}
		} else {
			err = nil
		}
	default:
		err = fmt.Errorf("notifier: not a notifiable event")
	}

	return
}

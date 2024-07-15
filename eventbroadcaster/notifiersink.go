// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package eventbroadcaster

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier"
	notifTypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotifierSink struct {
	client   client.Client
	notifier eventnotifier.Notifier
	logger   *zap.SugaredLogger
}

// RegisterNotifierSinks registers notifiers sinks on the given broadcaster
func RegisterNotifierSinks(mgr ctrl.Manager, broadcaster record.EventBroadcaster, notifiers []eventnotifier.Notifier, logger *zap.SugaredLogger) {
	for _, notifier := range notifiers {
		logger.Infof("notifier %s enabled", notifier.GetNotifierName())

		broadcaster.StartRecordingToSink(&NotifierSink{client: mgr.GetClient(), notifier: notifier, logger: logger})
	}

	corev1Client, _ := corev1client.NewForConfig(mgr.GetConfig())

	broadcaster.StartRecordingToSink(&corev1client.EventSinkImpl{Interface: corev1Client.Events("")})
}

func (s *NotifierSink) Create(event *corev1.Event) (*corev1.Event, error) {
	s.logger.Debugf("CREATE event received: %s", event.Message)

	var obj client.Object

	switch event.InvolvedObject.Kind {
	case v1beta1.DisruptionKind:
		disruption, err := s.getDisruption(event)
		if err != nil {
			s.logger.Warn(err)
			return event, fmt.Errorf("eventnotifier: unable to get Disruption object")
		}

		obj = &disruption
	case v1beta1.DisruptionCronKind:
		disruptionCron, err := s.getDisruptionCron(event)
		if err != nil {
			s.logger.Warn(err)
			return event, fmt.Errorf("eventnotifier: unable to get DisruptionCron object")
		}

		obj = &disruptionCron
	default:
		s.logger.Warnf("eventnotifier: not a disruption/disruptioncron event: kind %s", event.InvolvedObject.Kind)
		return event, nil
	}

	if err := s.parseEventToNotifier(event, obj); err != nil {
		s.logger.Error(err)
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

	if err := s.client.Get(context.Background(), types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, &dis); err != nil {
		return v1beta1.Disruption{}, err
	}

	return dis, nil
}

// getDisruptionCron fetches the disruption cron object of the event from the controller-runtime client
func (s *NotifierSink) getDisruptionCron(event *corev1.Event) (v1beta1.DisruptionCron, error) {
	disruptionCron := v1beta1.DisruptionCron{}

	if err := s.client.Get(context.Background(), types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, &disruptionCron); err != nil {
		return v1beta1.DisruptionCron{}, err
	}

	return disruptionCron, nil
}

// parseEventToNotifier contains the event parsing and notification logic
func (s *NotifierSink) parseEventToNotifier(event *corev1.Event, obj client.Object) (err error) {
	switch event.Type {
	case corev1.EventTypeWarning:
		err = s.notifier.Notify(obj, *event, notifTypes.NotificationWarning)
	case corev1.EventTypeNormal:
		if v1beta1.IsNotifiableEvent(*event) {
			switch {
			case v1beta1.IsRecoveryEvent(*event):
				err = s.notifier.Notify(obj, *event, notifTypes.NotificationSuccess)
			case v1beta1.IsCompletionEvent(*event):
				err = s.notifier.Notify(obj, *event, notifTypes.NotificationCompletion)
			default:
				err = s.notifier.Notify(obj, *event, notifTypes.NotificationInfo)
			}
		} else {
			err = nil
		}
	default:
		err = fmt.Errorf("notifier: not a notifiable event")
	}

	return
}

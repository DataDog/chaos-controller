// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package eventbroadcaster

import (
	"context"
	"encoding/json"
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
	s.logger.Debugw("CREATE event received:", "event", event)

	obj, err := s.getObject(event)
	if err != nil {
		s.logger.Warn(err)

		return event, nil
	}

	notificationType, err := s.getNotificationType(event)
	if err != nil {
		s.logger.Warnw("notifier: not a notifiable event")
		return event, nil
	}

	if err := s.notifier.Notify(obj, *event, notificationType); err != nil {
		return event, fmt.Errorf("notifier: failed to notify: %w", err)
	}

	return event, nil
}

func (s *NotifierSink) getObject(event *corev1.Event) (client.Object, error) {
	switch event.InvolvedObject.Kind {
	case v1beta1.DisruptionKind:
		var (
			disruption v1beta1.Disruption
			err        error
		)

		annotations := event.GetAnnotations()

		// If the event has the annotation, use it to unmarshal the disruption
		if disruptionString, ok := annotations[v1beta1.EventDisruptionAnnotation]; ok {
			if disruptionString == "" {
				return nil, fmt.Errorf("eventnotifier: empty disruption annotation")
			}

			if err := json.Unmarshal([]byte(disruptionString), &disruption); err != nil {
				return nil, fmt.Errorf("eventnotifier: failed to unmarshal disruption from annotation: %w", err)
			}
		} else {
			disruption, err = s.getDisruption(event)
			if err != nil {
				// If we can't get the disruption, we can't notify about it
				return nil, fmt.Errorf("eventnotifier: failed to get disruption: %w", err)
			}
		}

		return &disruption, nil
	case v1beta1.DisruptionCronKind:
		var (
			disruptionCron v1beta1.DisruptionCron
			err            error
		)

		annotations := event.GetAnnotations()

		// If the event has the annotation, use it to unmarshal the disruptionCron
		if disruptionCronString, ok := annotations[v1beta1.EventDisruptionCronAnnotation]; ok {
			if disruptionCronString == "" {
				return nil, fmt.Errorf("eventnotifier: empty disruptionCron annotation")
			}

			if err := json.Unmarshal([]byte(disruptionCronString), &disruptionCron); err != nil {
				return nil, fmt.Errorf("eventnotifier: failed to unmarshal disruptionCron from annotation: %w", err)
			}
		} else {
			// Otherwise, fetch the disruptionCron from the API server
			disruptionCron, err = s.getDisruptionCron(event)
			if err != nil {
				// If we can't get the disruptionCron, we can't notify about it
				return nil, fmt.Errorf("eventnotifier: failed to get disruptionCron: %w", err)
			}
		}

		return &disruptionCron, nil
	}

	return nil, fmt.Errorf("eventnotifier: not a disruption/disruptioncron event: kind %s", event.InvolvedObject.Kind)
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

func (s *NotifierSink) getNotificationType(event *corev1.Event) (notifTypes.NotificationType, error) {
	switch event.Type {
	case corev1.EventTypeWarning:
		return notifTypes.NotificationWarning, nil
	case corev1.EventTypeNormal:
		if v1beta1.IsNotifiableEvent(*event) {
			switch {
			case v1beta1.IsRecoveryEvent(*event):
				return notifTypes.NotificationSuccess, nil
			case v1beta1.IsDisruptionCompletionEvent(*event):
				return notifTypes.NotificationCompletion, nil
			default:
				return notifTypes.NotificationInfo, nil
			}
		}
	}

	return "", fmt.Errorf("notifier: not a notifiable event")
}

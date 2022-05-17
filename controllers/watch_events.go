package controllers

import (
	"context"
	"log"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
)

type TargetWatcher struct {
	watcher watch.Interface
	quit    chan bool
}

func (r *DisruptionReconciler) watchTargetPodEvents(disruption *chaosv1beta1.Disruption) *TargetWatcher {
	disruptionTargetWatcher := TargetWatcher{
		watcher: nil,
		quit:    make(chan bool),
	}

	go func(disruption *chaosv1beta1.Disruption, targetWatcher TargetWatcher) {
		var err error

		fieldSelector := fields.Set{
			"involvedObject.kind": "Pod",
			"type":                "Warning",
		}

		log.Printf("\n\nSTART WATCH")

		for {
			// We create the watcher channels when it's closed
			if targetWatcher.watcher == nil || targetWatcher.watcher.ResultChan() == nil {
				log.Printf("\n\nADDING WATCHER")
				targetWatcher.watcher, err = r.DirectClient.CoreV1().Events(disruption.Namespace).Watch(
					context.Background(),
					v1.ListOptions{
						FieldSelector: fieldSelector.AsSelector().String(),
					})
				if err != nil {
					log.Printf("\n\nERROR: %s", err.Error())
					return
				}
			}

			select {
			case <-targetWatcher.quit:
				return
			case watchEvent, ok := <-targetWatcher.watcher.ResultChan(): // We have changes in the service watched
				if !ok || watchEvent.Type == watch.Error { // channel is closed
					log.Printf("\n\nERROR")
					return
				}

				event, ok := watchEvent.Object.(*corev1.Event)
				if !ok {
					log.Printf("\n\nERROR")
					return
				}

				log.Printf("\n\nThere has been an event here: [%s] %s", event.UID, event.Message)
			}
		}
	}(disruption, disruptionTargetWatcher)

	return &disruptionTargetWatcher
}

func (r *DisruptionReconciler) watchTargetNodeEvents(nodeName string) watch.Interface {
	fieldSelector := fields.Set{
		"involvedObject.kind": "Node",
		"involvedObject.name": nodeName,
		"type":                "Warning",
	}

	watcher, err := r.DirectClient.CoreV1().Events("").Watch(
		context.Background(),
		v1.ListOptions{
			FieldSelector: fieldSelector.AsSelector().String(),
		})
	if err != nil {
		return nil
	}

	return watcher
}

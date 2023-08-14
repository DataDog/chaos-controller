// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package watchers

import (
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatefulSetHandler struct {
	Client client.Client

	log *zap.SugaredLogger
}

func NewStatefulSetHandler(client client.Client, logger *zap.SugaredLogger) StatefulSetHandler {
	return StatefulSetHandler{
		Client: client,
		log:    logger,
	}
}

// OnAdd is a handler function for the add of a statefulset
func (h StatefulSetHandler) OnAdd(_ interface{}) {

}

// OnUpdate is a handler function for the update of a statefulset
func (h StatefulSetHandler) OnUpdate(oldObj, newObj interface{}) {
}

// OnDelete is a handler function for the delete of a statefulset
func (h StatefulSetHandler) OnDelete(_ interface{}) {
	// Do nothing on delete event
}

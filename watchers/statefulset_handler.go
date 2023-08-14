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

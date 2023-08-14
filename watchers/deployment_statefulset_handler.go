package watchers

import (
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentStatefulSetHandler struct {
	Client client.Client

	log *zap.SugaredLogger
}

func NewDeploymentStatefulSetHandler(client client.Client, logger *zap.SugaredLogger) DeploymentStatefulSetHandler {
	return DeploymentStatefulSetHandler{
		Client: client,
		log:    logger,
	}
}

// OnAdd is a handler function for the add of a deployment or a statefulset
func (h DeploymentStatefulSetHandler) OnAdd(_ interface{}) {

}

// OnUpdate is a handler function for the update of a deployment or a statefulset
func (h DeploymentStatefulSetHandler) OnUpdate(oldObj, newObj interface{}) {
}

package watchers

import (
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentHandler struct {
	Client client.Client

	log *zap.SugaredLogger
}

func NewDeploymentHandler(client client.Client, logger *zap.SugaredLogger) DeploymentHandler {
	return DeploymentHandler{
		Client: client,
		log:    logger,
	}
}

// OnAdd is a handler function for the add of a deployment
func (h DeploymentHandler) OnAdd(_ interface{}) {

}

// OnUpdate is a handler function for the update of a deployment
func (h DeploymentHandler) OnUpdate(oldObj, newObj interface{}) {
}

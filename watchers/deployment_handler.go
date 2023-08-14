package watchers

import (
	context "context"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
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
	// Convert oldObj and newObj to Deployment objects
	_, okOldDeployment := oldObj.(*appsv1.Deployment)
	newDeployment, okNewDeployment := newObj.(*appsv1.Deployment)

	// If both old and new are not deployments, do nothing
	if !okOldDeployment || !okNewDeployment {
		return
	}

	// If deployment doesn't have associated disruption rollout, do nothing
	if !h.hasAssociatedDisruptionRollout(newDeployment) {
		return
	}
}

func (h DeploymentHandler) hasAssociatedDisruptionRollout(deployment *appsv1.Deployment) bool {
	indexedValue := "Deployment" + "-" + deployment.Namespace + "-" + deployment.Name

	disruptionRollouts := &chaosv1beta1.DisruptionRolloutList{}

	err := h.Client.List(context.TODO(), disruptionRollouts, client.MatchingFields{"targetResource": indexedValue})
	if err != nil {
		h.log.Errorw("unable to fetch DisruptionRollouts using index", "error", err, "indexedValue", indexedValue)
		return false
	}

	return len(disruptionRollouts.Items) > 0
}

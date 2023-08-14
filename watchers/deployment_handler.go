package watchers

import (
	context "context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	oldDeployment, okOldDeployment := oldObj.(*appsv1.Deployment)
	newDeployment, okNewDeployment := newObj.(*appsv1.Deployment)

	// If both old and new are not deployments, do nothing
	if !okOldDeployment || !okNewDeployment {
		return
	}

	// If deployment doesn't have associated disruption rollout, do nothing
	if !h.hasAssociatedDisruptionRollout(newDeployment) {
		return
	}

	// If pod spec of deployment hasn't changed, do nothing
	oldHash := hashPodSpec(&oldDeployment.Spec.Template.Spec)
	newHash := hashPodSpec(&newDeployment.Spec.Template.Spec)

	if oldHash != newHash {
		return
	}

	h.updateDisruptionRolloutStatus(newDeployment)
}

func (h DeploymentHandler) fetchAssociatedDisruptionRollouts(deployment *appsv1.Deployment) (*chaosv1beta1.DisruptionRolloutList, error) {
	indexedValue := "Deployment" + "-" + deployment.Namespace + "-" + deployment.Name

	disruptionRollouts := &chaosv1beta1.DisruptionRolloutList{}
	err := h.Client.List(context.TODO(), disruptionRollouts, client.MatchingFields{"targetResource": indexedValue})

	if err != nil {
		h.log.Errorw("unable to fetch DisruptionRollouts using index", "error", err, "indexedValue", indexedValue)
		return nil, err
	}

	return disruptionRollouts, nil
}

func (h DeploymentHandler) hasAssociatedDisruptionRollout(deployment *appsv1.Deployment) bool {
	disruptionRollouts, err := h.fetchAssociatedDisruptionRollouts(deployment)
	if err != nil {
		return false
	}

	return len(disruptionRollouts.Items) > 0
}

func (h DeploymentHandler) updateDisruptionRolloutStatus(deployment *appsv1.Deployment) {
	disruptionRollouts, err := h.fetchAssociatedDisruptionRollouts(deployment)
	if err != nil {
		return
	}

	for _, dr := range disruptionRollouts.Items {
		dr.Status.TargetResourcePodSpecHash = hashPodSpec(&deployment.Spec.Template.Spec)
		dr.Status.PodSpecChangeTimestamp = metav1.Now()

		err = h.Client.Status().Update(context.TODO(), &dr)
		if err != nil {
			h.log.Errorw("unable to update DisruptionRollout status", "DisruptionRollout", dr.Name, "error", err)
		}
	}
}

func hashPodSpec(spec *corev1.PodSpec) string {
	data, _ := json.Marshal(spec)
	return hex.EncodeToString(md5.New().Sum(data))
}

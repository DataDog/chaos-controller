// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:webhookVersions={v1beta1},path=/mutate-chaos-datadoghq-com-v1beta1-disruption-user-info,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions;disruptions/status,verbs=create,versions=v1beta1,,name=mdisruption.kb.io,admissionReviewVersions={v1,v1beta1}
type UserInfoMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	decoder *admission.Decoder
}

func (m *UserInfoMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d

	return nil
}

func (m *UserInfoMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	dis := &v1beta1.Disruption{}

	// ensure decoder is set
	if m.decoder == nil {
		m.Log.Errorw("webhook decoder seems to be nil while it should not, aborting")

		return admission.Errored(http.StatusInternalServerError, nil)
	}

	// decode object
	if err := m.decoder.Decode(req, dis); err != nil {
		m.Log.Errorw("error decoding disruption object", "error", err, "name", req.Name, "namespace", req.Namespace)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// retrieve user info
	m.Log.Infow("storing user info in disruption", "name", dis.Name, "namespace", dis.Namespace, "req", req.UserInfo)

	dis.Status.UserInfo = &req.UserInfo

	marshaled, err := json.Marshal(dis)
	if err != nil {
		m.Log.Errorw("error encoding modified disruption object", "error", err, "name", dis.Name, "namespace", dis.Namespace)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

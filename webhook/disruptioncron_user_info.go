// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

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

// +kubebuilder:webhook:webhookVersions={v1},path=/mutate-chaos-datadoghq-com-v1beta1-disruptioncron-user-info,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptioncrons;disruptioncrons/status,verbs=create,versions=v1beta1,,name=mdisruptioncron.kb.io,admissionReviewVersions={v1,v1beta1}

type DisruptionCronUserInfoMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	Decoder *admission.Decoder
}

func (m *DisruptionCronUserInfoMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	dis := &v1beta1.DisruptionCron{}

	// ensure Decoder is set
	if m.Decoder == nil {
		m.Log.Errorw("webhook Decoder seems to be nil while it should not, aborting")

		return admission.Errored(http.StatusInternalServerError, nil)
	}

	// decode object
	if err := m.Decoder.Decode(req, dis); err != nil {
		m.Log.Errorw("error decoding disruption cron object", "error", err, "disruptionName", req.Name, "disruptionNamespace", req.Namespace)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// retrieve user info
	m.Log.Infow("storing user info in annotations", "disruptionCronName", dis.Name, "disruptionCronNamespace", dis.Namespace, "req", req.UserInfo)

	annotations := make(map[string]string)

	for k, v := range dis.Annotations {
		annotations[k] = v
	}

	dis.Annotations = annotations

	err := dis.SetUserInfo(req.UserInfo)
	if err != nil {
		m.Log.Errorw("error defining UserInfo", "error", err, "disruptionCronName", dis.Name, "disruptionCronNamespace", dis.Namespace)
	}

	marshaled, err := json.Marshal(dis)
	if err != nil {
		m.Log.Errorw("error encoding modified annotations", "error", err, "disruptionCronName", dis.Name, "disruptionCronNamespace", dis.Namespace)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

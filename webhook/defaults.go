// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DefaultMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	decoder *admission.Decoder
}

func (m *DefaultMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d

	return nil
}

func (m *DefaultMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
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

	if dis.Spec.Duration == 0 {
		defaultDuration := time.Hour
		m.Log.Infow(fmt.Sprintf("setting default duration of %s in disruption", defaultDuration), "name", dis.Name, "namespace", dis.Namespace)
		dis.Spec.Duration = defaultDuration
	}

	marshaled, err := json.Marshal(dis)
	if err != nil {
		m.Log.Errorw("error encoding modified disruption object", "error", err, "name", dis.Name, "namespace", dis.Namespace)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

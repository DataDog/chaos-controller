// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	cLog "github.com/DataDog/chaos-controller/log"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:webhookVersions={v1},path=/mutate-chaos-datadoghq-com-v1beta1-user-info,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions;disruptions/status;disruptioncrons;disruptions/status,verbs=create,versions=v1beta1,,name=mchaos.kb.io,admissionReviewVersions={v1,v1beta1}

type UserInfoMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	Decoder *admission.Decoder
}

func (m UserInfoMutator) Handle(ctx context.Context, request admission.Request) admission.Response {
	log, err := m.getLogger(request)
	if err != nil {
		m.Log.Errorw("error getting logger", "error", err)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// ensure Decoder is set
	if m.Decoder == nil {
		err = fmt.Errorf("webhook Decoder seems to be nil while it should not, aborting")
		log.Error(err)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	object, err := m.getObject(request)
	if err != nil {
		log.Errorw("error getting object", "error", err)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// retrieve user info
	log.Infow("storing user info in annotations", "request", request.UserInfo)

	if err := m.setUserInfo(object, request); err != nil {
		m.Log.Errorw("error defining UserInfo", "error", err)
	}

	marshaled, err := json.Marshal(object)
	if err != nil {
		m.Log.Errorw("error encoding modified annotations", "error", err)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(request.Object.Raw, marshaled)
}

func (m UserInfoMutator) getObject(request admission.Request) (client.Object, error) {
	var object client.Object

	switch request.Kind.Kind {
	case v1beta1.DisruptionKind:
		object = &v1beta1.Disruption{}
	case v1beta1.DisruptionCronKind:
		object = &v1beta1.DisruptionCron{}
	default:
		return nil, fmt.Errorf("not a valid kind: %s", request.Kind.Kind)
	}

	// decode object
	if err := m.Decoder.Decode(request, object); err != nil {
		return nil, fmt.Errorf("error decoding object: %w", err)
	}

	return object, nil
}

func (m UserInfoMutator) getLogger(request admission.Request) (*zap.SugaredLogger, error) {
	switch request.Kind.Kind {
	case v1beta1.DisruptionKind:
		return m.Log.With(cLog.DisruptionNameKey, request.Name, cLog.DisruptionNamespaceKey, request.Namespace), nil
	case v1beta1.DisruptionCronKind:
		return m.Log.With(cLog.DisruptionCronNameKey, request.Name, cLog.DisruptionCronNamespaceKey, request.Namespace), nil
	}

	return nil, fmt.Errorf("not a valid kind: %s", request.Kind.Kind)
}

func (m UserInfoMutator) setUserInfo(object client.Object, request admission.Request) error {
	switch d := object.(type) {
	case *v1beta1.Disruption:
		return d.SetUserInfo(request.UserInfo)
	case *v1beta1.DisruptionCron:
		return d.SetUserInfo(request.UserInfo)
	}

	return nil
}

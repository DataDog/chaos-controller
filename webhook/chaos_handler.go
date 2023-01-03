// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ChaosHandlerMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	Image   string
	Timeout time.Duration
	decoder *admission.Decoder
}

func (m *ChaosHandlerMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d

	return nil
}

func (m *ChaosHandlerMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	// ensure decoder is set
	if m.decoder == nil {
		m.Log.Errorw("webhook decoder seems to be nil while it should not, aborting")

		return admission.Errored(http.StatusInternalServerError, nil)
	}

	// decode pod object
	err := m.decoder.Decode(req, pod)
	if err != nil {
		m.Log.Errorw("error decoding pod object", "error", err, "pod", pod.Name, "namespace", pod.Namespace)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// define pod name for logs
	// if the pod is created from a replicaset (deployment, statefulset)
	// the name won't be populated yet because it relies on the name generation
	// the logged pod name will then be the pod prefix instead of the full name
	podName := pod.Name
	if podName == "" {
		podName = pod.ObjectMeta.GenerateName
	}

	m.Log.Infow("injecting chaos handler init container into targeted pod", "pod", podName, "namespace", req.Namespace)

	// build chaos handler init container
	init := corev1.Container{
		Name:            "chaos-handler",
		Image:           m.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			"--timeout",
			m.Timeout.String(),
		},
	}

	// prepend chaos handler init container to already existing init containers
	pod.Spec.InitContainers = append([]corev1.Container{init}, pod.Spec.InitContainers...)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		m.Log.Errorw("error encoding modified pod object", "error", err, "pod", pod.Name, "namespace", pod.Namespace)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

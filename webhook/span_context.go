// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:webhookVersions={v1},path=/mutate-chaos-datadoghq-com-v1beta1-disruption-span-context,mutating=true,failurePolicy=fail,sideEffects=None,groups=chaos.datadoghq.com,resources=disruptions;disruptions/status,verbs=create,versions=v1beta1,,name=mdisruption.kb.io,admissionReviewVersions={v1,v1beta1}
type SpanContextMutator struct {
	Client  client.Client
	Log     *zap.SugaredLogger
	decoder *admission.Decoder
}

func (m *SpanContextMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d

	return nil
}

func (m *SpanContextMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	dis := &v1beta1.Disruption{}

	// ensure decoder is set
	if m.decoder == nil {
		m.Log.Errorw("webhook decoder seems to be nil while it should not, aborting")

		return admission.Errored(http.StatusInternalServerError, nil)
	}

	// decode object
	if err := m.decoder.Decode(req, dis); err != nil {
		m.Log.Errorw("error decoding disruption object", "error", err, "disruptionName", req.Name, "disruptionNamespace", req.Namespace)

		return admission.Errored(http.StatusBadRequest, err)
	}

	// retrieve span context
	m.Log.Infow("storing span context in annotations", "disruptionName", dis.Name, "disruptionNamespace", dis.Namespace)

	annotations := make(map[string]string)

	for k, v := range dis.Annotations {
		annotations[k] = v
	}

	dis.Annotations = annotations

	ctx, disruptionSpan := otel.Tracer("").Start(ctx, "disruption", trace.WithNewRoot())
	m.Log.Debugw("debug parent disruption",
		"step", "reconcile span start",
		"disruptionSpan", disruptionSpan,
		"disruptionSpanContext", disruptionSpan.SpanContext(),
	)

	defer disruptionSpan.End()

	m.Log.Debugw("debug parent disruption",
		"step", "reconcile span start",
		"disruptionSpan", disruptionSpan,
		"disruptionSpanContext", disruptionSpan.SpanContext(),
	)

	// writes the traceID and spanID in the annotations of the disruption
	err := dis.SetSpanContext(ctx)
	if err != nil {
		m.Log.Errorw("error defining SpanContext", "error", err, "disruptionName", dis.Name, "disruptionNamespace", dis.Namespace)
	}

	marshaled, err := json.Marshal(dis)
	if err != nil {
		m.Log.Errorw("error encoding modified annotations", "error", err, "disruptionName", dis.Name, "disruptionNamespace", dis.Namespace)

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

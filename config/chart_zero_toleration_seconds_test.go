// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	chaos "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/config"
	"github.com/DataDog/chaos-controller/services"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

// Requires Helm on PATH; no Kubernetes cluster is needed.
func TestChartPreservesZeroTolerationSecondsInInjectorPods(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm is required to render the chart")
	}

	for _, tc := range []struct {
		name    string
		field   string
		wantSet bool
		want    int64
	}{
		{name: "omitted"},
		{name: "null", field: "      tolerationSeconds: null\n"},
		{name: "zero", field: "      tolerationSeconds: 0\n", wantSet: true},
		{name: "positive", field: "      tolerationSeconds: 30\n", wantSet: true, want: 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := "injector:\n  tolerations:\n    - key: node.kubernetes.io/not-ready\n      operator: Exists\n      effect: NoExecute\n" + tc.field
			valuesPath := filepath.Join(t.TempDir(), "values.yaml")
			if err := os.WriteFile(valuesPath, []byte(values), 0600); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command("helm", "template", "regression", filepath.Join("..", "chart"),
				"--show-only", "templates/configmap.yaml", "--values", valuesPath).CombinedOutput()
			if err != nil {
				t.Fatalf("Helm rendering failed (Helm must be installed): %v\n%s", err, output)
			}
			var cm corev1.ConfigMap
			if err := yaml.Unmarshal(output, &cm); err != nil {
				t.Fatalf("decode rendered ConfigMap: %v", err)
			}
			// Decode the actual rendered tolerations using their production config
			// type, independently of unrelated global Viper bindings.
			var rendered struct {
				Injector struct {
					Tolerations []config.Toleration `json:"tolerations"`
				} `json:"injector"`
			}
			if err := yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &rendered); err != nil {
				t.Fatalf("decode controller configuration: %v", err)
			}
			svc, err := services.NewChaosPodService(services.ChaosPodServiceConfig{
				Client:   fake.NewClientBuilder().Build(),
				Injector: services.ChaosPodServiceInjectorConfig{Tolerations: rendered.Injector.Tolerations},
			})
			if err != nil {
				t.Fatal(err)
			}
			disruption := &chaos.Disruption{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "workload", CreationTimestamp: metav1.Now()},
				Spec:       chaos.DisruptionSpec{Duration: "1m"},
			}
			pod := svc.GenerateChaosPodOfDisruption(disruption, "target", "node", nil, chaostypes.DisruptionKindNetworkDisruption)
			if len(pod.Spec.Tolerations) != 1 {
				t.Fatalf("want one rendered toleration, got %v", pod.Spec.Tolerations)
			}
			got := pod.Spec.Tolerations[0].TolerationSeconds
			if !tc.wantSet {
				if got != nil {
					t.Fatalf("omitted/null duration must stay nil, got %d", *got)
				}
			} else if got == nil || *got != tc.want {
				t.Fatalf("tolerationSeconds must retain explicit %d, got %v", tc.want, got)
			}
		})
	}
}

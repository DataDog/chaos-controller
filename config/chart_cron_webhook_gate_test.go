// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// Requires Helm on PATH; it renders the real chart without contacting a cluster.
func TestChartGatesBothCronWebhookRoutes(t *testing.T) {
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm is required to render the chart")
	}

	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			output, err := exec.Command("helm", "template", "regression", filepath.Join("..", "chart"),
				"--show-only", "templates/webhook.yaml",
				"--set", "controller.webhook.generateCert=true",
				"--set", "controller.disruptionCronEnabled="+strconv.FormatBool(enabled)).CombinedOutput()
			if err != nil {
				t.Fatalf("Helm rendering failed (Helm must be installed): %v\n%s", err, output)
			}

			paths := map[string]int{}
			decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(output), 4096)
			for {
				var manifest struct {
					Kind     string `json:"kind"`
					Webhooks []struct {
						ClientConfig admissionv1.WebhookClientConfig `json:"clientConfig"`
					} `json:"webhooks"`
				}
				if err := decoder.Decode(&manifest); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("decode rendered webhook YAML: %v", err)
				}
				if manifest.Kind != "MutatingWebhookConfiguration" && manifest.Kind != "ValidatingWebhookConfiguration" {
					continue
				}
				for _, hook := range manifest.Webhooks {
					if service := hook.ClientConfig.Service; service != nil && service.Path != nil {
						paths[*service.Path]++
					}
				}
			}

			wantCronRoutes := 0
			if enabled {
				wantCronRoutes = 1
			}
			for path, want := range map[string]int{
				"/mutate-chaos-datadoghq-com-v1beta1-disruptioncron":   wantCronRoutes,
				"/validate-chaos-datadoghq-com-v1beta1-disruptioncron": wantCronRoutes,
				"/mutate-chaos-datadoghq-com-v1beta1-disruption":       1,
				"/validate-chaos-datadoghq-com-v1beta1-disruption":     1,
			} {
				if got := paths[path]; got != want {
					t.Errorf("disruptionCronEnabled=%t: %s has %d routes, want %d", enabled, path, got, want)
				}
			}
		})
	}
}

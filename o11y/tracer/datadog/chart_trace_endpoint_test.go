// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package datadog

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestChartTraceEndpointReachesConfiguredAgent(t *testing.T) {
	deployment, err := os.ReadFile("../../../chart/templates/deployment.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Read the literal environment entry actually shipped by the chart. Its only
	// variable is Kubernetes' HOST_IP expansion, so Helm rendering is unnecessary.
	entry := regexp.MustCompile(`(?m)^\s*- name: ((?:DD_)?TRACE_AGENT_URL)\s*\n\s*value:\s*([^\r\n]+)`).FindSubmatch(deployment)
	if len(entry) != 3 {
		t.Fatal("chart trace-agent environment entry not found")
	}

	receivedTrace := make(chan struct{}, 1)
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/info" {
			fmt.Fprint(w, `{"endpoints":["/v0.4/traces"]}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/traces") {
			select {
			case receivedTrace <- struct{}{}:
			default:
			}
		}
		fmt.Fprint(w, `{"rate_by_service":{}}`)
	}))
	defer agent.Close()

	// Keep the baseline's unrecognized variable from contacting a real agent.
	// The chart-provided URL must override this deliberately unreachable fallback.
	t.Setenv("DD_AGENT_HOST", "127.0.0.1")
	t.Setenv("DD_TRACE_AGENT_PORT", "1")
	t.Setenv("DD_DOGSTATSD_PORT", "1")
	t.Setenv("DD_TRACE_AGENT_URL", "")
	t.Setenv("TRACE_AGENT_URL", "")
	t.Setenv("DD_TRACE_ENABLED", "true")
	t.Setenv("DD_TRACE_SAMPLE_RATE", "1")
	t.Setenv("DD_INSTRUMENTATION_TELEMETRY_ENABLED", "false")
	t.Setenv("DD_REMOTE_CONFIGURATION_ENABLED", "false")
	t.Setenv("DD_APPSEC_ENABLED", "false")
	value := strings.Trim(strings.TrimSpace(string(entry[2])), `"'`)
	value = strings.Replace(value, "$(HOST_IP):8126", strings.TrimPrefix(agent.URL, "http://"), 1)
	t.Setenv(string(entry[1]), value)

	sink := New()
	t.Cleanup(func() { _ = sink.Stop() })
	_, span := sink.GetProvider().Tracer("chart-endpoint-regression").Start(context.Background(), "chart-agent-endpoint")
	span.End()
	if err := sink.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-receivedTrace:
	case <-time.After(2 * time.Second):
		t.Fatalf("no trace reached the chart-configured agent using %s=%s", entry[1], value)
	}
}

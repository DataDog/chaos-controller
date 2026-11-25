// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package chart_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// These two templates use only Go-template field lookups, so they can be
// rendered directly without Helm or a Kubernetes cluster.
func TestInjectorRoleBindingMatchesCreatedServiceAccount(t *testing.T) {
	for _, account := range []string{"chaos-injector", "custom-chaos-injector"} {
		t.Run(account, func(t *testing.T) {
			const namespace = "custom-chaos-system"
			values := map[string]interface{}{
				"Values": map[string]interface{}{
					"chaosNamespace": namespace,
					"injector":       map[string]interface{}{"serviceAccount": account},
				},
			}
			type document struct {
				Kind     string            `json:"kind"`
				Metadata metav1.ObjectMeta `json:"metadata"`
				Subjects []rbacv1.Subject  `json:"subjects"`
			}
			accounts := make(map[string]bool)
			var injectorBinding *document
			for _, filename := range []string{"service_account.yaml", "role_binding.yaml"} {
				contents, err := os.ReadFile(filepath.Join("templates", filename))
				if err != nil {
					t.Fatal(err)
				}
				tmpl, err := template.New(filename).Parse(string(contents))
				if err != nil {
					t.Fatal(err)
				}
				var rendered bytes.Buffer
				if err := tmpl.Execute(&rendered, values); err != nil {
					t.Fatal(err)
				}
				decoder := utilyaml.NewYAMLOrJSONDecoder(&rendered, 4096)
				for {
					var doc document
					if err := decoder.Decode(&doc); err != nil {
						if err == io.EOF {
							break
						}
						t.Fatal(err)
					}
					if doc.Kind == "ServiceAccount" {
						accounts[doc.Metadata.Namespace+"/"+doc.Metadata.Name] = true
					}
					if doc.Kind == "ClusterRoleBinding" && doc.Metadata.Name == "chaos-injector" {
						injectorBinding = &doc
					}
				}
			}
			if !accounts[namespace+"/"+account] {
				t.Fatal("chart did not create the configured injector service account")
			}
			if injectorBinding == nil {
				t.Fatal("injector ClusterRoleBinding missing")
			}
			for _, subject := range injectorBinding.Subjects {
				if subject.Kind == "ServiceAccount" && subject.Name == account && subject.Namespace == namespace {
					return
				}
			}
			t.Fatalf("injector ClusterRoleBinding does not authorize the created account %s/%s: %#v", namespace, account, injectorBinding.Subjects)
		})
	}
}

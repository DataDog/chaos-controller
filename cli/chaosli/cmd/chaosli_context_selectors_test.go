// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestContextAdvancedSelectors(t *testing.T) {
	for _, resource := range []string{"pods", "nodes"} {
		for _, withSimple := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/simple=%t", resource, withSimple), func(t *testing.T) {
				queries := make(chan string, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					queries <- r.URL.Query().Get("labelSelector")
					w.Header().Set("Content-Type", "application/json")
					kind := "PodList"
					if resource == "nodes" {
						kind = "NodeList"
					}
					_, _ = fmt.Fprintf(w, `{"apiVersion":"v1","kind":%q,"items":[]}`, kind)
				}))
				defer server.Close()
				oldClient := clientset
				defer func() { clientset = oldClient }()
				var err error
				clientset, err = kubernetes.NewForConfig(&rest.Config{Host: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				disruption := v1beta1.Disruption{
					ObjectMeta: metav1.ObjectMeta{Namespace: "app"},
					Spec: v1beta1.DisruptionSpec{AdvancedSelector: []metav1.LabelSelectorRequirement{{
						Key: "env", Operator: metav1.LabelSelectorOpIn, Values: []string{"production"},
					}}},
				}
				if withSimple {
					disruption.Spec.Selector = labels.Set{"app": "api"}
				}
				if resource == "pods" {
					_, err = getPods(disruption)
				} else {
					_, err = getNodes(disruption)
				}
				if err != nil {
					t.Fatal(err)
				}
				query := <-queries
				selector, err := labels.Parse(query)
				if err != nil {
					t.Fatal(err)
				}
				if selector.Matches(labels.Set{"app": "api", "env": "staging"}) {
					t.Fatalf("request selector %q includes targets excluded by advancedSelector", query)
				}
				if !selector.Matches(labels.Set{"app": "api", "env": "production"}) {
					t.Fatalf("request selector %q excludes the intended target", query)
				}
				if withSimple && selector.Matches(labels.Set{"app": "other", "env": "production"}) {
					t.Fatalf("request selector %q ignores the simple selector", query)
				}
			})
		}
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"errors"
	"strings"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserInfoImmutable(t *testing.T) {
	tests := []struct {
		name, oldInfo, newInfo, wantError string
	}{
		{"identical identity", `{"username":"example","uid":"id","groups":["a","b"],"extra":{"scope":["read","write"]}}`, `{"username":"example","uid":"id","groups":["a","b"],"extra":{"scope":["read","write"]}}`, ""},
		{"groups merged", `{"groups":["a","b"]}`, `{"groups":["a b"]}`, "immutable"},
		{"groups split", `{"groups":["a b"]}`, `{"groups":["a","b"]}`, "immutable"},
		{"empty group added", `{"username":"example","groups":[]}`, `{"username":"example","groups":[""]}`, "immutable"},
		{"empty group cannot initialize identity", `{"groups":[""]}`, `{"username":"example"}`, "immutable"},
		{"groups reordered", `{"groups":["a","b"]}`, `{"groups":["b","a"]}`, "immutable"},
		{"extra values merged", `{"extra":{"scope":["read","write"]}}`, `{"extra":{"scope":["read write"]}}`, "immutable"},
		{"extra values split", `{"extra":{"scope":["read write"]}}`, `{"extra":{"scope":["read","write"]}}`, "immutable"},
		{"extra map formatting collision", `{"extra":{"a":["b] c:[d"]}}`, `{"extra":{"a":["b"],"c":["d"]}}`, "immutable"},
		{"extra whitespace changed", `{"extra":{"scope":["read write"]}}`, `{"extra":{"scope":["read  write"]}}`, "immutable"},
		{"extra key changed", `{"extra":{"scope":["read"]}}`, `{"extra":{"other":["read"]}}`, "immutable"},
		{"extra empty entry added", `{"username":"example","extra":{}}`, `{"username":"example","extra":{"scope":[]}}`, "immutable"},
		{"UID case changed", `{"uid":"id"}`, `{"uid":"ID"}`, "immutable"},
		{"username changed", `{"username":"example"}`, `{"username":"other"}`, "immutable"},
		{"JSON formatting ignored", `{"uid":"id","username":"example"}`, `{ "username": "example", "uid": "id" }`, ""},
		{"empty to empty", `{}`, `{}`, ""},
		{"empty identity initialized", `{}`, `{"username":"example"}`, ""},
		{"empty collections initialized", `{"groups":[],"extra":{}}`, `{"username":"example"}`, ""},
		{"nil to empty groups", `{"username":"example"}`, `{"username":"example","groups":[]}`, ""},
		{"empty to nil groups", `{"username":"example","groups":[]}`, `{"username":"example","groups":null}`, ""},
		{"nil to empty extra", `{"username":"example"}`, `{"username":"example","extra":{}}`, ""},
		{"empty to nil extra", `{"username":"example","extra":{}}`, `{"username":"example"}`, ""},
		{"nil to empty extra values", `{"extra":{"scope":null}}`, `{"extra":{"scope":[]}}`, ""},
		{"empty to nil extra values", `{"extra":{"scope":[]}}`, `{"extra":{"scope":null}}`, ""},
		{"identity cleared", `{"username":"example"}`, `{}`, "immutable"},
		{"new annotation malformed", `{"username":"example"}`, `{`, "unable to unmarshal"},
		{"legacy malformed annotation", `{`, `{"username":"example"}`, ""},
	}
	factories := map[string]func() client.Object{
		"Disruption":     func() client.Object { return &Disruption{} },
		"DisruptionCron": func() client.Object { return &DisruptionCron{} },
	}
	for kind, factory := range factories {
		t.Run(kind, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					oldObject, newObject := factory(), factory()
					oldObject.SetAnnotations(map[string]string{annotationUserInfoKey: tt.oldInfo})
					newObject.SetAnnotations(map[string]string{annotationUserInfoKey: tt.newInfo})
					err := validateUserInfoImmutable(oldObject, newObject)
					if tt.wantError == "" {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
						t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
					}
				})
			}
			t.Run("legacy missing annotation", func(t *testing.T) {
				oldObject, newObject := factory(), factory()
				newObject.SetAnnotations(map[string]string{annotationUserInfoKey: `{"username":"example"}`})
				if err := validateUserInfoImmutable(oldObject, newObject); err != nil {
					t.Fatal(err)
				}
			})
			t.Run("annotation removed", func(t *testing.T) {
				oldObject, newObject := factory(), factory()
				oldObject.SetAnnotations(map[string]string{annotationUserInfoKey: `{"username":"example"}`})
				if err := validateUserInfoImmutable(oldObject, newObject); !errors.Is(err, ErrNoUserInfo) {
					t.Fatalf("expected ErrNoUserInfo, got %v", err)
				}
			})
		})
	}
}

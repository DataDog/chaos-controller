// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"strings"
	"testing"

	"github.com/DataDog/chaos-controller/config"
	"go.uber.org/zap"
)

// Install in config/permitted_groups_userinfo_hook_regression_test.go.
func TestPermittedGroupsRequireAuthenticatedUserInfo(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "reject annotation-only authorization",
			args:    []string{"--user-info-webhook=false", "--permitted-user-groups=chaos-admins"},
			wantErr: true,
		},
		{
			name: "allow authenticated authorization",
			args: []string{"--user-info-webhook=true", "--permitted-user-groups=chaos-admins"},
		},
		{
			name: "allow hook disabled without group restrictions",
			args: []string{"--user-info-webhook=false"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := config.New(nil, zap.NewNop().Sugar(), tc.args)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "permitted user groups") {
					t.Fatalf("expected unsafe authorization configuration to be rejected, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("valid authorization configuration rejected: %v", err)
			}
		})
	}
}

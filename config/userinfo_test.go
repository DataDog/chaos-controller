// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/DataDog/chaos-controller/config"
	"github.com/spf13/viper"
	"go.uber.org/zap/zaptest"
)

func TestPermittedGroupsRequireUserInfoHook(t *testing.T) {
	const guardError = "cannot configure permitted user groups without enabling the user info webhook"

	for _, tt := range []struct {
		name    string
		file    string
		args    []string
		wantErr bool
	}{
		{name: "defaults"},
		{name: "hook off without groups", args: []string{"--user-info-webhook=false"}},
		{name: "groups with default hook", args: []string{"--permitted-user-groups=operators"}},
		{name: "groups with hook on", args: []string{"--permitted-user-groups=operators", "--user-info-webhook=true"}},
		{name: "groups with hook off", args: []string{"--permitted-user-groups=operators", "--user-info-webhook=false"}, wantErr: true},
		{name: "multiple groups with hook off", args: []string{"--user-info-webhook=false", "--permitted-user-groups=operators,reviewers"}, wantErr: true},
		{name: "safemode off still requires hook", args: []string{"--user-info-webhook=false", "--permitted-user-groups=operators", "--safemode-enable=false"}, wantErr: true},
		{name: "file rejects unsafe combination", file: "controller:\n  userInfoHook: false\n  safeMode:\n    permittedUserGroups: [operators]\n", wantErr: true},
		{name: "file accepts hook on", file: "controller:\n  userInfoHook: true\n  safeMode:\n    permittedUserGroups: [operators]\n"},
		{name: "file accepts hook off without groups", file: "controller:\n  userInfoHook: false\n"},
		{name: "flag disables hook from file", file: "controller:\n  userInfoHook: true\n  safeMode:\n    permittedUserGroups: [operators]\n", args: []string{"--user-info-webhook=false"}, wantErr: true},
		{name: "flag enables hook from file", file: "controller:\n  userInfoHook: false\n  safeMode:\n    permittedUserGroups: [operators]\n", args: []string{"--user-info-webhook=true"}},
		{name: "flag adds groups to file", file: "controller:\n  userInfoHook: false\n", args: []string{"--permitted-user-groups=operators"}, wantErr: true},
		{name: "flag clears groups from file", file: "controller:\n  userInfoHook: false\n  safeMode:\n    permittedUserGroups: [operators]\n", args: []string{"--permitted-user-groups="}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// New uses Viper's global instance; isolate configuration precedence cases.
			viper.Reset()
			t.Cleanup(viper.Reset)
			args := tt.args
			if tt.file != "" {
				path := filepath.Join(t.TempDir(), "config.yaml")
				// Match the existing file configuration's scalar service account field.
				contents := tt.file + "injector:\n  serviceAccount: chaos-injector\n"
				if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
					t.Fatal(err)
				}
				args = append([]string{"--config", path}, args...)
			}

			_, err := config.New(nil, zaptest.NewLogger(t).Sugar(), args)
			if tt.wantErr {
				if err == nil || err.Error() != guardError {
					t.Fatalf("expected %q, got %v", guardError, err)
				}
			} else if err != nil {
				t.Fatalf("expected valid configuration, got %v", err)
			}
		})
	}
}

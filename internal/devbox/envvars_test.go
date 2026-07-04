// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"maps"
	"testing"
)

func TestOnlyModifiedEnvs(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]string
		baseEnv map[string]string
		want    map[string]string
	}{
		{
			name:    "empty",
			envs:    map[string]string{},
			baseEnv: map[string]string{},
			want:    map[string]string{},
		},
		{
			name: "drops unchanged variables",
			envs: map[string]string{
				"HOSTNAME":    "myhost",
				"LANG":        "en_US.UTF-8",
				"PROFILEREAD": "true",
			},
			baseEnv: map[string]string{
				"HOSTNAME":    "myhost",
				"LANG":        "en_US.UTF-8",
				"PROFILEREAD": "true",
			},
			want: map[string]string{},
		},
		{
			name: "keeps changed and new variables",
			envs: map[string]string{
				"PATH":                "/nix/store/bin:/usr/bin", // changed
				"HOSTNAME":            "myhost",                  // unchanged
				"DEVBOX_PROJECT_ROOT": "/home/user/project",      // new
			},
			baseEnv: map[string]string{
				"PATH":     "/usr/bin",
				"HOSTNAME": "myhost",
			},
			want: map[string]string{
				"PATH":                "/nix/store/bin:/usr/bin",
				"DEVBOX_PROJECT_ROOT": "/home/user/project",
			},
		},
		{
			name: "keeps variable that became empty",
			envs: map[string]string{
				"FOO": "",
			},
			baseEnv: map[string]string{
				"FOO": "bar",
			},
			want: map[string]string{
				"FOO": "",
			},
		},
		{
			name: "empty base env keeps everything",
			envs: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
			baseEnv: map[string]string{},
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := onlyModifiedEnvs(test.envs, test.baseEnv)
			if !maps.Equal(got, test.want) {
				t.Errorf("onlyModifiedEnvs() = %v, want %v", got, test.want)
			}
		})
	}
}

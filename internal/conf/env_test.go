// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package conf

import (
	"maps"
	"testing"
)

func TestSplitDefault(t *testing.T) {
	tests := []struct {
		ref, name, def string
		ok             bool
	}{
		{"FOO", "FOO", "", false},
		{"FOO:-bar", "FOO", "bar", true},
		{"FOO:-", "FOO", "", true},
		{"FOO:-a:-b", "FOO", "a:-b", true},
		{"FOO:-$BAR", "FOO", "$BAR", true},
	}
	for _, tt := range tests {
		name, def, ok := SplitDefault(tt.ref)
		if name != tt.name || def != tt.def || ok != tt.ok {
			t.Errorf("SplitDefault(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.ref, name, def, ok, tt.name, tt.def, tt.ok)
		}
	}
}

func TestOSExpandEnvMap(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		existingEnv map[string]string
		want        map[string]string
	}{
		{
			name:        "references existing env vars",
			env:         map[string]string{"A": "$FOO", "B": "${FOO}/x", "C": "$MISSING"},
			existingEnv: map[string]string{"FOO": "foo"},
			want:        map[string]string{"A": "foo", "B": "foo/x", "C": ""},
		},
		{
			name:        "default used when unset",
			env:         map[string]string{"A": "${MISSING:-fallback}"},
			existingEnv: map[string]string{},
			want:        map[string]string{"A": "fallback"},
		},
		{
			name:        "default used when empty",
			env:         map[string]string{"A": "${EMPTY:-fallback}"},
			existingEnv: map[string]string{"EMPTY": ""},
			want:        map[string]string{"A": "fallback"},
		},
		{
			name:        "default ignored when set",
			env:         map[string]string{"A": "${FOO:-fallback}"},
			existingEnv: map[string]string{"FOO": "foo"},
			want:        map[string]string{"A": "foo"},
		},
		{
			name:        "empty default",
			env:         map[string]string{"A": "x${MISSING:-}y"},
			existingEnv: map[string]string{},
			want:        map[string]string{"A": "xy"},
		},
		{
			name:        "default references another var",
			env:         map[string]string{"A": "${MISSING:-$FOO}", "B": "${MISSING:-$ALSO_MISSING}"},
			existingEnv: map[string]string{"FOO": "foo"},
			want:        map[string]string{"A": "foo", "B": ""},
		},
		{
			name:        "default within larger string",
			env:         map[string]string{"URL": "http://${HOST:-localhost}:${PORT:-8080}/"},
			existingEnv: map[string]string{"PORT": "9000"},
			want:        map[string]string{"URL": "http://localhost:9000/"},
		},
		{
			name:        "nil existing env uses defaults",
			env:         map[string]string{"A": "${FOO:-fallback}", "B": "$FOO"},
			existingEnv: nil,
			want:        map[string]string{"A": "fallback", "B": ""},
		},
		{
			name:        "PWD is project dir",
			env:         map[string]string{"A": "$PWD/bin", "B": "${PWD:-nope}"},
			existingEnv: map[string]string{"PWD": "/somewhere/else"},
			want:        map[string]string{"A": "/project/bin", "B": "/project"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OSExpandEnvMap(tt.env, tt.existingEnv, "/project")
			if !maps.Equal(got, tt.want) {
				t.Errorf("OSExpandEnvMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

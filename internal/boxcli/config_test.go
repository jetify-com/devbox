// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"testing"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/envir"
)

// TestConfigFlagDefaultsToEnv verifies that the --config flag defaults to the
// DEVBOX_CONFIG environment variable when the flag is not passed, and that an
// explicit flag value still takes precedence.
func TestConfigFlagDefaultsToEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string // empty means unset
		args   []string
		want   string
	}{
		{
			name: "unset env and no flag yields empty path",
			want: "",
		},
		{
			name:   "env var sets the default config path",
			envVal: "/path/from/env",
			want:   "/path/from/env",
		},
		{
			name:   "explicit flag overrides env var",
			envVal: "/path/from/env",
			args:   []string{"--config", "/path/from/flag"},
			want:   "/path/from/flag",
		},
		{
			name: "explicit flag without env var",
			args: []string{"-c", "/path/from/flag"},
			want: "/path/from/flag",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.envVal != "" {
				t.Setenv(envir.DevboxConfig, testCase.envVal)
			} else {
				t.Setenv(envir.DevboxConfig, "")
			}

			flags := &pathFlag{}
			cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
			flags.register(cmd)
			cmd.SetArgs(testCase.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}

			if flags.path != testCase.want {
				t.Errorf("flags.path = %q, want %q", flags.path, testCase.want)
			}
		})
	}
}

// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"testing"

	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/envir"
)

// TestPathFlagEnvVar verifies that the --config flag falls back to the
// DEVBOX_CONFIG environment variable when the flag is not provided, and that
// an explicitly passed --config flag takes precedence over the env var.
func TestPathFlagEnvVar(t *testing.T) {
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
			name:   "env var sets path when flag is absent",
			envVal: "/from/env",
			want:   "/from/env",
		},
		{
			name:   "explicit flag overrides env var",
			envVal: "/from/env",
			args:   []string{"--config", "/from/flag"},
			want:   "/from/flag",
		},
		{
			name: "explicit flag without env var",
			args: []string{"--config", "/from/flag"},
			want: "/from/flag",
		},
	}

	for _, persistent := range []bool{false, true} {
		for _, tt := range tests {
			name := tt.name
			if persistent {
				name += " (persistent)"
			}
			t.Run(name, func(t *testing.T) {
				if tt.envVal != "" {
					t.Setenv(envir.DevboxConfig, tt.envVal)
				} else {
					// Ensure the env var is unset for this subtest.
					t.Setenv(envir.DevboxConfig, "")
				}

				flags := &configFlags{}
				cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
				if persistent {
					flags.registerPersistent(cmd)
				} else {
					flags.register(cmd)
				}

				cmd.SetArgs(tt.args)
				if err := cmd.Execute(); err != nil {
					t.Fatalf("cmd.Execute() error = %v", err)
				}

				if flags.path != tt.want {
					t.Errorf("flags.path = %q, want %q", flags.path, tt.want)
				}
			})
		}
	}
}

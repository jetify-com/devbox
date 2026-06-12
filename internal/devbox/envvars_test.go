// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import "testing"

func TestExportify(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want string
	}{
		{
			name: "simple value",
			vars: map[string]string{"FOO": "bar"},
			want: `export FOO="bar";`,
		},
		{
			name: "escapes shell-special characters",
			vars: map[string]string{"FOO": "a$b`c\"d\\e"},
			want: `export FOO="a\$b\` + "`" + `c\"d\\e";`,
		},
		{
			// Regression test for #2814: a multi-line value (e.g. a
			// PROMPT_COMMAND set by bash-preexec) must keep its newlines
			// literal. Escaping a newline with a backslash produces a line
			// continuation that the shell removes, concatenating the lines and
			// corrupting the value.
			name: "preserves embedded newlines without escaping",
			vars: map[string]string{"PROMPT_COMMAND": "__bp_precmd_invoke_cmd\ndbus-send >/dev/null 2>&1\n__bp_interactive_mode"},
			want: "export PROMPT_COMMAND=\"__bp_precmd_invoke_cmd\ndbus-send >/dev/null 2>&1\n__bp_interactive_mode\";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exportify(tt.vars)
			if got != tt.want {
				t.Errorf("exportify() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

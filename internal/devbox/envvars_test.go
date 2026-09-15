// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"io"
	"strings"
	"testing"
)

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
			got := exportify(io.Discard, tt.vars)
			if got != tt.want {
				t.Errorf("exportify() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestIsValidEnvName(t *testing.T) {
	valid := []string{"FOO", "_foo", "foo_BAR_123", "a", "_"}
	for _, name := range valid {
		if !isValidEnvName(name) {
			t.Errorf("isValidEnvName(%q) = false, want true", name)
		}
	}

	invalid := []string{"//", "//ccache", "bad.name", "1leading", "with space", "with-dash", ""}
	for _, name := range invalid {
		if isValidEnvName(name) {
			t.Errorf("isValidEnvName(%q) = true, want false", name)
		}
	}
}

// TestExportifySkipsInvalidNames ensures that env vars whose names aren't valid
// shell identifiers (e.g. a "//" comment key in devbox.json) are dropped instead
// of producing invalid shell that breaks the whole shell.
func TestExportifySkipsInvalidNames(t *testing.T) {
	got := exportify(io.Discard, map[string]string{
		"GOOD":     "value",
		"//":       "comment-as-json-hack",
		"//ccache": "another comment",
		"bad.name": "dotted",
		"1leading": "starts with digit",
	})

	if !strings.Contains(got, `export GOOD="value";`) {
		t.Errorf("expected valid var to be exported, got:\n%s", got)
	}
	for _, bad := range []string{"//", "//ccache", "bad.name", "1leading"} {
		if strings.Contains(got, bad) {
			t.Errorf("expected invalid name %q to be skipped, got:\n%s", bad, got)
		}
	}
}

func TestExportifyNushellSkipsInvalidNames(t *testing.T) {
	got := exportifyNushell(io.Discard, map[string]string{
		"GOOD": "value",
		"//":   "comment",
	})

	if !strings.Contains(got, `$env.GOOD = "value"`) {
		t.Errorf("expected valid var to be exported, got:\n%s", got)
	}
	if strings.Contains(got, "//") {
		t.Errorf("expected invalid name to be skipped, got:\n%s", got)
	}
}

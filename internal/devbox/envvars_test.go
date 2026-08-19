// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"io"
	"strings"
	"testing"
)

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

// TestExportifyPreservesNewlines ensures that multi-line values (e.g. a
// PROMPT_COMMAND made of several commands separated by newlines) are emitted so
// that the shell preserves the newlines. A backslash before a newline would be a
// line continuation that the shell deletes, joining adjacent lines and corrupting
// the value (see jetify-com/devbox#2814).
func TestExportifyPreservesNewlines(t *testing.T) {
	value := "__bp_precmd_invoke_cmd\ndbus-send ... >/dev/null 2>&1\n__bp_interactive_mode"
	got := exportify(io.Discard, map[string]string{"PROMPT_COMMAND": value})

	// The emitted value must contain a bare newline, not an escaped one. If the
	// newline were escaped, "2>&1\" and "__bp_interactive_mode" would collapse
	// into "2>&1__bp_interactive_mode" when the shell sourced the export.
	if strings.Contains(got, "\\\n") {
		t.Errorf("newline should not be backslash-escaped, got:\n%q", got)
	}
	want := "export PROMPT_COMMAND=\"" + value + "\";"
	if !strings.Contains(got, want) {
		t.Errorf("expected exported value to preserve newlines.\ngot:\n%q\nwant to contain:\n%q", got, want)
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

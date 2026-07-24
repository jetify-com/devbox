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

// TestExportifyPreservesNewlines ensures multi-line env values (e.g. a
// PROMPT_COMMAND that spans several lines) survive round-tripping through the
// shell. A newline must be emitted literally, not as a backslash-newline: inside
// double quotes the latter is a line continuation that the shell removes, which
// silently joins adjacent lines and corrupts the value. See issue #2814, where a
// multi-line PROMPT_COMMAND collapsed into `... 2>&1__bp_interactive_mode` and
// produced a "bash: ...: ambiguous redirect" error at every prompt.
func TestExportifyPreservesNewlines(t *testing.T) {
	value := "line1 >/dev/null 2>&1\n__bp_interactive_mode"
	got := exportify(io.Discard, map[string]string{"PROMPT_COMMAND": value})

	// The value must be emitted with a literal newline, never a
	// backslash-newline (which bash would strip as a line continuation).
	if strings.Contains(got, "2>&1\\\n") {
		t.Errorf("newline was escaped as a line continuation, which corrupts the value:\n%s", got)
	}
	want := "export PROMPT_COMMAND=\"line1 >/dev/null 2>&1\n__bp_interactive_mode\";"
	if got != want {
		t.Errorf("exportify(...) = %q, want %q", got, want)
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

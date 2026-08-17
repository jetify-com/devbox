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

// TestExportifyPreservesNewlines ensures that a value containing newlines is
// emitted with literal (unescaped) newlines. Escaping a newline as `\<newline>`
// makes the shell treat it as a line continuation and strip it, collapsing a
// multi-line value onto a single line. That corrupted, for example, a
// PROMPT_COMMAND whose `... 2>&1` and `__bp_interactive_mode` lines got joined
// into `... 2>&1__bp_interactive_mode`, yielding a "1__bp_...: ambiguous
// redirect" error at every prompt. See issue #2814.
func TestExportifyPreservesNewlines(t *testing.T) {
	value := "cmd_a >/dev/null 2>&1\n__bp_interactive_mode"
	got := exportify(io.Discard, map[string]string{"PROMPT_COMMAND": value})

	want := "export PROMPT_COMMAND=\"cmd_a >/dev/null 2>&1\n__bp_interactive_mode\";"
	if got != want {
		t.Errorf("exportify newline handling:\n got: %q\nwant: %q", got, want)
	}

	// A backslash-newline (line continuation) must not appear: that is the
	// exact corruption this test guards against.
	if strings.Contains(got, "\\\n") {
		t.Errorf("exportify escaped a newline as a line continuation, corrupting the value:\n%s", got)
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

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

// TestExportifyPreservesNewlines ensures that env var values containing newlines
// (e.g. a PROMPT_COMMAND whose commands are newline-separated, as set up by
// bash-preexec) are emitted with literal newlines rather than backslash-newline
// line continuations. A backslash-newline is deleted by the shell, which would
// glue adjacent lines together and corrupt the value (see issue #2814, where
// `... 2>&1\n__bp_interactive_mode` became `... 2>&1__bp_interactive_mode` and
// produced a "1__bp_interactive_mode: ambiguous redirect" error).
func TestExportifyPreservesNewlines(t *testing.T) {
	got := exportify(io.Discard, map[string]string{
		"PROMPT_COMMAND": "__bp_precmd_invoke_cmd\ncmd >/dev/null 2>&1\n__bp_interactive_mode",
	})

	want := "export PROMPT_COMMAND=\"__bp_precmd_invoke_cmd\ncmd >/dev/null 2>&1\n__bp_interactive_mode\";"
	if got != want {
		t.Errorf("exportify did not preserve newlines\ngot:\n%q\nwant:\n%q", got, want)
	}
	// A backslash immediately before a newline is a shell line continuation and
	// must not appear, since it would delete the newline and join the lines.
	if strings.Contains(got, "\\\n") {
		t.Errorf("exportify emitted a backslash-newline line continuation, which corrupts multi-line values:\n%q", got)
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

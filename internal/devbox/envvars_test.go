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

// TestOnlyModifiedEnvVars ensures that variables identical to the ambient
// environment are dropped while new or changed variables are kept. This is what
// keeps `devbox shellenv` from re-exporting unrelated, possibly read-only
// variables (see issue #2826).
func TestOnlyModifiedEnvVars(t *testing.T) {
	ambient := map[string]string{
		"HOSTNAME":    "myhost",
		"LANG":        "en_US.UTF-8",
		"PROFILEREAD": "true",
		"PATH":        "/usr/bin:/bin",
	}
	env := map[string]string{
		"HOSTNAME":            "myhost",               // unchanged -> dropped
		"LANG":                "en_US.UTF-8",          // unchanged -> dropped
		"PROFILEREAD":         "true",                 // unchanged (read-only) -> dropped
		"PATH":                "/devbox/bin:/usr/bin", // changed -> kept
		"DEVBOX_PROJECT_ROOT": "/home/user/proj",      // new -> kept
	}

	got := onlyModifiedEnvVars(env, ambient)

	want := map[string]string{
		"PATH":                "/devbox/bin:/usr/bin",
		"DEVBOX_PROJECT_ROOT": "/home/user/proj",
	}
	if len(got) != len(want) {
		t.Fatalf("onlyModifiedEnvVars returned %d vars, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("onlyModifiedEnvVars[%q] = %q, want %q", k, got[k], v)
		}
	}
	for _, dropped := range []string{"HOSTNAME", "LANG", "PROFILEREAD"} {
		if _, ok := got[dropped]; ok {
			t.Errorf("expected unchanged var %q to be dropped, got:\n%v", dropped, got)
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

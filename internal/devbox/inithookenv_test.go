// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNulEnv(t *testing.T) {
	in := []byte("FOO=bar\x00MULTI=line1\nline2\x00EMPTY=\x00")
	got := parseNulEnv(in)
	want := map[string]string{
		"FOO":   "bar",
		"MULTI": "line1\nline2",
		"EMPTY": "",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %#v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q: got %q, want %q", k, got[k], v)
		}
	}
}

func TestCaptureEnvWithInitHook(t *testing.T) {
	dir := t.TempDir()
	hooksPath := filepath.Join(dir, "hooks.sh")
	// The init hook sets a new var, modifies an existing one, and prints to
	// stdout (which must not leak into the captured env).
	hookBody := "echo 'hello from init hook'\n" +
		"export TEST=true\n" +
		"export BASE=modified\n"
	if err := os.WriteFile(hooksPath, []byte(hookBody), 0o755); err != nil {
		t.Fatal(err)
	}

	baseEnv := map[string]string{
		"BASE": "original",
		"PATH": os.Getenv("PATH"),
	}

	var hookStdout bytes.Buffer
	got, err := captureEnvWithInitHook(context.Background(), hooksPath, baseEnv, &hookStdout)
	if err != nil {
		t.Fatalf("captureEnvWithInitHook returned error: %v", err)
	}

	if got["TEST"] != "true" {
		t.Errorf("TEST: got %q, want %q", got["TEST"], "true")
	}
	if got["BASE"] != "modified" {
		t.Errorf("BASE: got %q, want %q (init hook should override base env)", got["BASE"], "modified")
	}

	// The init hook's stdout must not corrupt the captured environment.
	if _, ok := got["hello from init hook"]; ok {
		t.Errorf("init hook stdout leaked into captured env: %#v", got)
	}
}

func TestCaptureEnvWithInitHook_NoHooksFile(t *testing.T) {
	baseEnv := map[string]string{"FOO": "bar"}
	got, err := captureEnvWithInitHook(
		context.Background(),
		filepath.Join(t.TempDir(), "does-not-exist.sh"),
		baseEnv,
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["FOO"] != "bar" {
		t.Errorf("expected base env to be returned unchanged, got %#v", got)
	}
}

// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/cmdutil"
)

// TestRunScriptUsesBashForBashisms verifies that RunScript executes scripts
// with bash so that init hooks and scripts relying on bash features work with
// `devbox run`, matching the behavior of `devbox shell`. `source` is a bash
// builtin that POSIX sh implementations like dash don't provide, so it makes a
// good proxy for "are we running under bash?".
//
// Regression test for https://github.com/jetify-com/devbox/issues/2607
func TestRunScriptUsesBashForBashisms(t *testing.T) {
	if !cmdutil.Exists("bash") {
		t.Skip("bash is not installed; RunScript falls back to sh")
	}

	// `source` fails under dash (`source: not found`) but succeeds under bash.
	dir := t.TempDir()
	sourced := filepath.Join(dir, "sourced.sh")
	if err := os.WriteFile(sourced, []byte("SOURCED=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := RunScript(dir, "source "+sourced, map[string]string{})
	if err != nil {
		t.Fatalf("RunScript should run bashisms under bash, got error: %v", err)
	}
}

func TestScriptRunnerPathPrefersBash(t *testing.T) {
	if !cmdutil.Exists("bash") {
		t.Skip("bash is not installed")
	}
	if got := scriptRunnerPath(); !strings.HasSuffix(got, "bash") {
		t.Errorf("scriptRunnerPath() = %q, want a path ending in \"bash\"", got)
	}
}

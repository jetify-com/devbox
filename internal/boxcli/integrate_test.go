// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEditorBinary(t *testing.T) {
	// Create a temp directory with a bin/code binary to simulate
	// a VS Code installation (e.g., WSL's VSCODE_CWD).
	installDir := t.TempDir()
	binDir := filepath.Join(installDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	codePath := filepath.Join(binDir, "code")
	if err := os.WriteFile(codePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// A directory that does NOT contain bin/code, simulating
	// macOS where VSCODE_CWD is the workspace directory.
	workspaceDir := t.TempDir()

	tests := []struct {
		name      string
		vscodeCWD string // empty means unset
		ideName   string
		want      string
	}{
		{
			name:    "VSCODE_CWD unset falls back to bare name",
			ideName: "code",
			want:    "code",
		},
		{
			name:      "VSCODE_CWD with valid binary uses full path",
			vscodeCWD: installDir,
			ideName:   "code",
			want:      filepath.Join(installDir, "bin", "code"),
		},
		{
			name:      "VSCODE_CWD without binary falls back to bare name",
			vscodeCWD: workspaceDir,
			ideName:   "code",
			want:      "code",
		},
		{
			name:      "non-default IDE name resolves correctly",
			vscodeCWD: workspaceDir,
			ideName:   "cursor",
			want:      "cursor",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.vscodeCWD != "" {
				t.Setenv("VSCODE_CWD", testCase.vscodeCWD)
			} else {
				os.Unsetenv("VSCODE_CWD")
			}

			got := resolveEditorBinary(testCase.ideName)
			if got != testCase.want {
				t.Errorf(
					"resolveEditorBinary(%q) = %q, want %q",
					testCase.ideName, got, testCase.want,
				)
			}
		})
	}
}

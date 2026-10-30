// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/nix"
)

func TestGenerateSubcommandsAcceptConfigFlags(t *testing.T) {
	for _, sub := range generateCmd().Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if err := sub.ParseFlags([]string{"-c", "./proj", "--environment", "prod"}); err != nil {
				t.Errorf("generate %s: %v", sub.Name(), err)
			}
		})
	}
}

// TestGenerateDirenvPrintEnvrcSkipsNixCheck ensures that `generate direnv
// --print-envrc`, which direnv runs on every reload, doesn't check for nix.
func TestGenerateDirenvPrintEnvrcSkipsNixCheck(t *testing.T) {
	if nix.Ensured() {
		t.Skip("an earlier test already ensured nix is installed")
	}

	root := RootCmd()
	stdout := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetArgs([]string{"generate", "direnv", "--print-envrc"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "devbox shellenv") {
		t.Errorf("got output %q, want an .envrc that calls devbox shellenv", stdout)
	}
	if nix.Ensured() {
		t.Error("generate direnv --print-envrc checked that nix is installed")
	}
}

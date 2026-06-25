// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"io"
)

// These make it easier to stub out nix for testing
type NixInstance struct{}

type Nixer interface {
	PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*PrintDevEnvOut, error)
	RunScriptWithStreams(projectDir, cmdWithArgs string, env map[string]string, stdin io.Reader, stdout, stderr io.Writer, capture bool) (*RunScriptOutput, error)
}

func (n *NixInstance) RunScriptWithStreams(projectDir, cmdWithArgs string, env map[string]string, stdin io.Reader, stdout, stderr io.Writer, capture bool) (*RunScriptOutput, error) {
	return RunScriptWithStreams(projectDir, cmdWithArgs, env, stdin, stdout, stderr, capture)
}

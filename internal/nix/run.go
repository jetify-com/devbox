// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/debug"
)

func RunScript(projectDir, cmdWithArgs string, env map[string]string) error {
	if cmdWithArgs == "" {
		return errors.New("attempted to run an empty command or script")
	}

	envPairs := []string{}
	for k, v := range env {
		envPairs = append(envPairs, fmt.Sprintf("%s=%s", k, v))
	}

	// Try to find sh in the PATH, if not, default to a well known absolute path.
	shPath := cmdutil.GetPathOrDefault("sh", "/bin/sh")
	cmd := exec.Command(shPath, "-c", cmdWithArgs)
	cmd.Env = envPairs
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	debug.Log("Executing: %v", cmd.Args)
	// Report error as exec error when executing scripts.
	return usererr.NewExecError(cmd.Run())
}

// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/cmdutil"
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

	slog.Debug("executing script", "cmd", cmd.Args)
	// Report error as exec error when executing scripts.
	return usererr.NewExecError(cmd.Run())
}

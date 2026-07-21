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

	cmd := exec.Command(scriptRunnerPath(), "-c", cmdWithArgs)
	cmd.Env = envPairs
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	slog.Debug("executing script", "cmd", cmd.Args)
	// Report error as exec error when executing scripts.
	return usererr.NewExecError(cmd.Run())
}

// scriptRunnerPath returns the shell used to execute Devbox scripts and init
// hooks. It prefers bash so that scripts run with `devbox run` behave the same
// as they do inside `devbox shell`, which sources the init hooks from the
// user's interactive shell (usually bash). Init hooks commonly rely on bash
// features such as the `source` builtin, which POSIX sh implementations like
// dash don't provide. We fall back to `sh` (and finally a well-known absolute
// path) when bash isn't available.
// See https://github.com/jetify-com/devbox/issues/2607
func scriptRunnerPath() string {
	// Prefer bash on PATH.
	if bashPath := cmdutil.GetPathOrDefault("bash", ""); bashPath != "" {
		return bashPath
	}
	// bash may be installed at a well-known location even when it isn't on
	// PATH (e.g. a restricted environment where PATH doesn't include /bin or
	// /usr/bin). Check those before giving up on bash.
	for _, p := range []string{"/bin/bash", "/usr/bin/bash", "/usr/local/bin/bash"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	// Fall back to POSIX sh.
	return cmdutil.GetPathOrDefault("sh", "/bin/sh")
}

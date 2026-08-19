// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"runtime/trace"
	"strings"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/fileutil"
	"go.jetify.com/devbox/internal/shellgen"
)

// EnvVarsWithInitHook returns the environment variables for the Devbox
// environment, including any variables set (or modified) by the project's
// init hook.
//
// Unlike EnvVars, which deliberately excludes the init hook, this sources the
// init hook in a subshell and captures the resulting environment. It is meant
// for integrations that launch a program directly, without going through a
// devbox shell (or `devbox run`) that would otherwise source the init hook —
// for example the VSCode "Reopen in Devbox" action. See issue #2703.
//
// If the init hook fails, the environment without the hook's modifications is
// returned rather than erroring, so the integration keeps working.
func (d *Devbox) EnvVarsWithInitHook(ctx context.Context) ([]string, error) {
	ctx, task := trace.NewTask(ctx, "devboxEnvVarsWithInitHook")
	defer task.End()

	env, err := d.ensureStateIsUpToDateAndComputeEnv(ctx, devopt.EnvOptions{})
	if err != nil {
		return nil, err
	}

	// Persist the init hook (and scripts) to disk so we can source the hooks
	// file below.
	if err := shellgen.WriteScriptsToFiles(d); err != nil {
		return nil, err
	}

	hooksPath := shellgen.ScriptPath(d.ProjectDir(), shellgen.HooksFilename)
	withHooks, err := captureEnvWithInitHook(ctx, hooksPath, env, d.stderr)
	if err != nil {
		// Don't fail the whole integration if the init hook errors. Fall back
		// to the environment without the hook's modifications.
		slog.Debug("failed to run init hook while computing env", "err", err)
		return envir.MapToPairs(env), nil
	}
	return envir.MapToPairs(withHooks), nil
}

// captureEnvWithInitHook sources the init hook script at hooksPath using the
// given base environment and returns the resulting environment. The init
// hook's own stdout is redirected to hookStderr so it cannot corrupt the
// captured environment dump.
func captureEnvWithInitHook(
	ctx context.Context,
	hooksPath string,
	baseEnv map[string]string,
	hookStderr io.Writer,
) (map[string]string, error) {
	if !fileutil.Exists(hooksPath) {
		// No hooks file; nothing to source.
		return baseEnv, nil
	}

	// Source the init hook, then print the resulting environment NUL-separated
	// so values containing newlines stay intact.
	//
	//   - The hooks path is passed as a positional parameter ($1) rather than
	//     interpolated into the script, so special characters in the path (e.g.
	//     spaces, quotes, $(), backticks) can't change shell parsing.
	//   - The hook's own stdout is redirected to stderr (1>&2) so only the env
	//     dump reaches stdout and can't corrupt it.
	//   - awk's POSIX ENVIRON is used instead of `env -0`: the latter isn't
	//     supported by macOS' default /usr/bin/env, whereas awk is portable.
	const script = `. "$1" 1>&2
exec awk 'BEGIN { for (k in ENVIRON) printf "%s=%s%c", k, ENVIRON[k], 0 }'`
	cmd := exec.CommandContext(ctx, "sh", "-c", script, "sh", hooksPath)
	cmd.Env = envir.MapToPairs(baseEnv)
	cmd.Stderr = hookStderr
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return parseNulEnv(out), nil
}

// parseNulEnv parses NUL-separated KEY=VALUE pairs into a map.
func parseNulEnv(b []byte) map[string]string {
	result := map[string]string{}
	for _, pair := range strings.Split(string(b), "\x00") {
		if pair == "" {
			continue
		}
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		result[key] = value
	}
	return result
}

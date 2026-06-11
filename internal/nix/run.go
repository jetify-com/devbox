// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/cmdutil"
)

// RunScriptOutput contains the output from running a script
type RunScriptOutput struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func RunScript(projectDir, cmdWithArgs string, env map[string]string) error {
	_, err := RunScriptWithOutput(projectDir, cmdWithArgs, env, false)
	return err
}

// RunScriptWithOutput runs a script and optionally captures stdout/stderr
// When capture is true, it returns the output; otherwise it behaves like RunScript
func RunScriptWithOutput(projectDir, cmdWithArgs string, env map[string]string, capture bool) (*RunScriptOutput, error) {
	return RunScriptWithStreams(projectDir, cmdWithArgs, env, os.Stdin, os.Stdout, os.Stderr, capture)
}

// RunScriptWithStreams runs a script with custom stdin/stdout/stderr streams
// When capture is true, it also captures output and returns it; otherwise it only uses the provided streams
func RunScriptWithStreams(projectDir, cmdWithArgs string, env map[string]string, stdin io.Reader, stdout, stderr io.Writer, capture bool) (*RunScriptOutput, error) {
	if cmdWithArgs == "" {
		return nil, errors.New("attempted to run an empty command or script")
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
	cmd.Stdin = stdin

	var stdoutBuf, stderrBuf bytes.Buffer
	if capture {
		// Use a multi-writer to both capture and stream output
		cmd.Stdout = io.MultiWriter(&stdoutBuf, stdout)
		cmd.Stderr = io.MultiWriter(&stderrBuf, stderr)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}

	slog.Debug("executing script", "cmd", cmd.Args)
	
	err := cmd.Run()
	
	output := &RunScriptOutput{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: 0,
	}
	
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
		}
		// Report error as exec error when executing scripts.
		return output, usererr.NewExecError(err)
	}
	
	return output, nil
}

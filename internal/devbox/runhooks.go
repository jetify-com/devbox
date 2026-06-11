// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/devconfig/configfile"
)

// HookContext provides context to hooks about the command being run
type HookContext struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Dir     string   `json:"dir"`
}

// HookResult is the result of a hook execution
type HookResult struct {
	Success      bool   `json:"success"`
	ExitCode     int    `json:"exit_code,omitempty"`
	ModifiedArgs []string `json:"modified_args,omitempty"`
	ModifiedEnv  map[string]string `json:"modified_env,omitempty"`
	Block        bool   `json:"block,omitempty"`
	BlockReason  string `json:"block_reason,omitempty"`
	ModifiedExit *int   `json:"modified_exit,omitempty"`
	ModifiedStdout string `json:"modified_stdout,omitempty"`
	ModifiedStderr string `json:"modified_stderr,omitempty"`
}

// executePreRunHook executes a pre_run hook with the given context
func (d *Devbox) executePreRunHook(ctx context.Context, hook *configfile.RunHook, hookCtx *HookContext) (*HookResult, error) {
	slog.Debug("Executing pre_run hook", "command", hook.Command)

	result := &HookResult{
		Success: true,
	}

	// Set hook context environment variables
	env := make(map[string]string)
	for k, v := range hookCtx.Env {
		env[k] = v
	}
	
	// Convert hook context to JSON for environment variables
	argsJSON, _ := json.Marshal(hookCtx.Args)
	envJSON, _ := json.Marshal(hookCtx.Env)
	
	env["DEVBOX_HOOK_COMMAND"] = hookCtx.Command
	env["DEVBOX_HOOK_ARGS"] = string(argsJSON)
	env["DEVBOX_HOOK_ENV"] = string(envJSON)
	env["DEVBOX_HOOK_DIR"] = hookCtx.Dir

	// Execute the hook command
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = d.projectDir
	cmd.Env = d.envSlice(env)

	// Capture stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// If hook can modify stdin, we could pipe stdin here
	// For now, we'll keep it simple

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Success = false
		} else {
			return nil, errors.Wrap(err, "hook execution failed")
		}
	}

	// Parse hook output if it returned JSON
	if result.Success && stdoutBuf.Len() > 0 {
		var hookOutput HookResult
		if err := json.Unmarshal(stdoutBuf.Bytes(), &hookOutput); err == nil {
			// Hook returned valid JSON, use it
			result = &hookOutput
		} else {
			// Hook returned non-JSON output, log it but don't fail
			slog.Debug("Hook returned non-JSON output", "output", stdoutBuf.String())
		}
	}

	// Filter hook results based on capability gates
	filteredResult := &HookResult{
		Success:  result.Success,
		ExitCode: result.ExitCode,
	}

	// Only allow blocking if capability is granted
	if hook.CanBlock {
		filteredResult.Block = result.Block
		filteredResult.BlockReason = result.BlockReason
	}

	// Only allow arg modifications if capability is granted
	if hook.CanModifyArgs {
		filteredResult.ModifiedArgs = result.ModifiedArgs
	}

	// Only allow env modifications if capability is granted
	if hook.CanModifyEnv {
		filteredResult.ModifiedEnv = result.ModifiedEnv
	}

	// Only allow stdin modifications if capability is granted
	if hook.CanModifyStdin {
		// Note: stdin modification not yet implemented
	}

	// Check if hook blocked execution
	if filteredResult.Block {
		return filteredResult, nil
	}

	// Apply modifications if capabilities allow
	if hook.CanModifyArgs && filteredResult.ModifiedArgs != nil {
		hookCtx.Args = filteredResult.ModifiedArgs
	}
	if hook.CanModifyEnv && filteredResult.ModifiedEnv != nil {
		for k, v := range filteredResult.ModifiedEnv {
			hookCtx.Env[k] = v
		}
	}

	return filteredResult, nil
}

// executePostRunHook executes a post_run hook with the given context
func (d *Devbox) executePostRunHook(ctx context.Context, hook *configfile.RunHook, hookCtx *HookContext, exitCode int, stdout, stderr string) (*HookResult, error) {
	slog.Debug("Executing post_run hook", "command", hook.Command)

	// We'll pass exit code and output via environment variables for simplicity
	env := make(map[string]string)
	for k, v := range hookCtx.Env {
		env[k] = v
	}
	env["DEVBOX_HOOK_EXIT_CODE"] = fmt.Sprintf("%d", exitCode)
	env["DEVBOX_HOOK_STDOUT"] = stdout
	env["DEVBOX_HOOK_STDERR"] = stderr

	result := &HookResult{
		Success: true,
	}

	// Execute the hook command
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = d.projectDir
	cmd.Env = d.envSlice(env)

	// Capture stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Success = false
		} else {
			return nil, errors.Wrap(err, "hook execution failed")
		}
	}

	// Parse hook output if it returned JSON
	if result.Success && stdoutBuf.Len() > 0 {
		var hookOutput HookResult
		if err := json.Unmarshal(stdoutBuf.Bytes(), &hookOutput); err == nil {
			// Hook returned valid JSON, use it
			result = &hookOutput
		} else {
			// Hook returned non-JSON output, log it but don't fail
			slog.Debug("Hook returned non-JSON output", "output", stdoutBuf.String())
		}
	}

	// Filter hook results based on capability gates
	filteredResult := &HookResult{
		Success:  result.Success,
		ExitCode: result.ExitCode,
	}

	// Only allow exit code modifications if capability is granted
	if hook.CanModifyExit {
		filteredResult.ModifiedExit = result.ModifiedExit
	}

	// Only allow stdout modifications if capability is granted
	if hook.CanModifyStdout {
		filteredResult.ModifiedStdout = result.ModifiedStdout
	}

	// Only allow stderr modifications if capability is granted
	if hook.CanModifyStderr {
		filteredResult.ModifiedStderr = result.ModifiedStderr
	}

	return filteredResult, nil
}

// envSlice converts a map to environment variable slice
func (d *Devbox) envSlice(envMap map[string]string) []string {
	env := os.Environ()
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// applyCommandWrapper applies the command wrapper to the command
func applyCommandWrapper(cmdWithArgs []string, wrapper string) []string {
	if wrapper == "" {
		return cmdWithArgs
	}

	// Split wrapper into parts
	wrapperParts := strings.Fields(wrapper)
	if len(wrapperParts) == 0 {
		return cmdWithArgs
	}

	// Prepend wrapper to command
	return append(wrapperParts, cmdWithArgs...)
}

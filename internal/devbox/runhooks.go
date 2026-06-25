// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/devconfig/configfile"
)

// closedReader is an io.Reader that always returns EOF
type closedReader struct{}

func (cr *closedReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

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
	return d.executePreRunHookWithStreams(ctx, hook, hookCtx, os.Stdin, os.Stdout, os.Stderr)
}

// executePreRunHookWithStreams executes a pre_run hook with custom streams
func (d *Devbox) executePreRunHookWithStreams(ctx context.Context, hook *configfile.RunHook, hookCtx *HookContext, stdin io.Reader, stdout, stderr io.Writer) (*HookResult, error) {
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

	// Set up streams with read capability checks
	// If hook doesn't have can_read_stdin, provide a closed reader to prevent access
	if hook.CanReadStdin {
		cmd.Stdin = stdin
	} else {
		cmd.Stdin = &closedReader{}
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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

	// For non-streaming hooks, parse JSON output from stdout
	// We can detect streaming by checking if stdout is os.Stdout
	if buf, ok := stdout.(*bytes.Buffer); ok {
		// Non-streaming case with buffer - parse JSON output
		// Limit JSON output size to prevent OOM (1MB limit)
		const maxJSONSize = 1 * 1024 * 1024 // 1MB
		output := buf.String()
		if len(output) > maxJSONSize {
			slog.Warn("Hook JSON output too large, skipping parsing", "size", len(output), "max_size", maxJSONSize)
		} else if output != "" {
			if err := json.Unmarshal([]byte(output), result); err != nil {
				// If JSON parsing fails, just use the default result
				slog.Debug("Failed to parse hook JSON output", "error", err)
			}
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
	return d.executePostRunHookWithStreams(ctx, hook, hookCtx, exitCode, os.Stdin, os.Stdout, os.Stderr)
}

// executePostRunHookWithStreams executes a post_run hook with custom streams
func (d *Devbox) executePostRunHookWithStreams(ctx context.Context, hook *configfile.RunHook, hookCtx *HookContext, exitCode int, stdin io.Reader, stdout, stderr io.Writer) (*HookResult, error) {
	slog.Debug("Executing post_run hook", "command", hook.Command)

	// We'll pass exit code via environment variable for simplicity
	env := make(map[string]string)
	for k, v := range hookCtx.Env {
		env[k] = v
	}
	env["DEVBOX_HOOK_EXIT_CODE"] = fmt.Sprintf("%d", exitCode)

	result := &HookResult{
		Success: true,
	}

	// Execute the hook command
	cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
	cmd.Dir = d.projectDir
	cmd.Env = d.envSlice(env)

	// Set up streams with read capability checks
	// If hook doesn't have can_read_stdin, provide a closed reader to prevent access
	if hook.CanReadStdin {
		cmd.Stdin = stdin
	} else {
		cmd.Stdin = &closedReader{}
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Success = false
		} else {
			return nil, errors.Wrap(err, "hook execution failed")
		}
	}

	// For non-streaming hooks, parse JSON output from stdout
	// We can detect streaming by checking if stdout is os.Stdout
	if buf, ok := stdout.(*bytes.Buffer); ok {
		// Non-streaming case with buffer - parse JSON output
		// Limit JSON output size to prevent OOM (1MB limit)
		const maxJSONSize = 1 * 1024 * 1024 // 1MB
		output := buf.String()
		if len(output) > maxJSONSize {
			slog.Warn("Hook JSON output too large, skipping parsing", "size", len(output), "max_size", maxJSONSize)
		} else if output != "" {
			if err := json.Unmarshal([]byte(output), result); err != nil {
				// If JSON parsing fails, just use the default result
				slog.Debug("Failed to parse hook JSON output", "error", err)
			}
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

// executeWithStreamingPipeline executes the command with a streaming hook pipeline
// Pipeline: stdin -> [pre_run hooks] -> [command_wrapper] -> [post_run hooks] -> stdout
func (d *Devbox) executeWithStreamingPipeline(ctx context.Context, hookCtx *HookContext, cmdWithArgs []string, wrapper string) error {
	// Get hooks from config
	cfg := d.cfg
	
	preRunHooks := cfg.Root.PreRunHooks()
	postRunHooks := cfg.Root.PostRunHooks()
	
	// If no hooks and no wrapper, run directly
	if len(preRunHooks) == 0 && len(postRunHooks) == 0 && wrapper == "" {
		return d.executeCommandDirectly(ctx, hookCtx, cmdWithArgs)
	}
	
	// Build the pipeline
	var stdin io.Reader = os.Stdin
	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr
	
	// Stage 1: Pre-run hooks
	for i := range preRunHooks {
		// Create pipe for this hook's output
		pr, pw := io.Pipe()
		
		// Execute hook with current stdin, writing to pipe
		result, err := d.executePreRunHookWithStreams(ctx, preRunHooks[i], hookCtx, stdin, pw, stderr)
		pw.Close()
		
		if err != nil {
			pr.Close()
			return errors.Wrap(err, "pre_run hook failed")
		}
		
		// Check if hook blocked execution
		if result.Block {
			pr.Close()
			if result.BlockReason != "" {
				return errors.New(result.BlockReason)
			}
			return errors.New("command blocked by pre_run hook")
		}
		
		// Next stage reads from this pipe
		stdin = pr
	}
	
	// Stage 2: Command wrapper + actual command
	// Apply wrapper if present
	finalCmd := cmdWithArgs
	if wrapper != "" {
		finalCmd = applyCommandWrapper(cmdWithArgs, wrapper)
	}
	
	// Create pipe for command output
	cmdPr, cmdPw := io.Pipe()
	
	// Execute the command with streaming
	cmdString := strings.Join(finalCmd, " ")
	
	// We need to run the command in a goroutine to stream output
	// but also capture the exit code for post-run hooks
	type commandResult struct {
		exitCode int
		err      error
	}
	cmdResultChan := make(chan commandResult, 1)
	
	go func() {
		defer cmdPw.Close()
		// Run the command with stdin from pre-run hooks, stdout to pipe
		output, err := d.nix.RunScriptWithStreams(d.projectDir, cmdString, hookCtx.Env, stdin, cmdPw, stderr, false)
		if err != nil {
			// Still return output even on error for exit code
			cmdResultChan <- commandResult{exitCode: output.ExitCode, err: err}
			return
		}
		cmdResultChan <- commandResult{exitCode: output.ExitCode, err: nil}
	}()
	
	// Stage 3: Post-run hooks (process streaming stdin from command)
	// Process command output through post-run hooks
	currentReader := cmdPr
	var exitCode int
	
	for i := range postRunHooks {
		// Create pipe for this hook's output
		hookPr, hookPw := io.Pipe()
		
		// Execute hook with stdin from previous stage
		result, err := d.executePostRunHookWithStreams(ctx, postRunHooks[i], hookCtx, exitCode, currentReader, hookPw, stderr)
		hookPw.Close()
		
		if err != nil {
			hookPr.Close()
			currentReader.Close()
			return errors.Wrap(err, "post_run hook failed")
		}
		
		// Apply exit code modification if allowed
		if postRunHooks[i].CanModifyExit && result.ModifiedExit != nil {
			exitCode = *result.ModifiedExit
		}
		
		// Close previous stage's reader
		currentReader.Close()
		
		// Next stage reads from this pipe
		currentReader = hookPr
	}
	
	// Final output goes to stdout
	go func() {
		io.Copy(stdout, currentReader)
		currentReader.Close()
	}()
	
	// Wait for command to complete
	result := <-cmdResultChan
	exitCode = result.exitCode
	
	// Return the command error if any
	if result.err != nil {
		return result.err
	}
	
	// If exit code was modified and is non-zero, return an error
	if exitCode != 0 {
		return fmt.Errorf("command exited with code %d", exitCode)
	}
	
	return nil
}

// executeCommandDirectly executes a command without any hooks
func (d *Devbox) executeCommandDirectly(ctx context.Context, hookCtx *HookContext, cmdWithArgs []string) error {
	cmdString := strings.Join(cmdWithArgs, " ")
	_, err := d.nix.RunScriptWithStreams(d.projectDir, cmdString, hookCtx.Env, os.Stdin, os.Stdout, os.Stderr, false)
	return err
}

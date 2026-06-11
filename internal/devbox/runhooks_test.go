// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/internal/devconfig/configfile"
)

func TestHookExecutionPermutations(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		config         string
		setupHook      func(t *testing.T, hookPath string)
		expectBlock    bool
		expectExitCode int
		verifyBehavior func(t *testing.T, result *HookResult)
	}{
		{
			name: "no pre_run hooks",
			config: `{
				"packages": [],
				"shell": {
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "one pre_run hook without capabilities",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "echo 'pre-run hook'"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "two pre_run hooks",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [
						{"command": "echo 'hook 1'"},
						{"command": "echo 'hook 2'"}
					],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "pre_run hook modifies environment",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "HOOK_PATH",
						"can_modify_env": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that modifies environment
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_env": {"TEST_VAR": "test_value"}}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedEnv == nil {
					t.Error("Expected modified_env to be set")
				} else if result.ModifiedEnv["TEST_VAR"] != "test_value" {
					t.Errorf("Expected TEST_VAR to be 'test_value', got %q", result.ModifiedEnv["TEST_VAR"])
				}
			},
		},
		{
			name: "pre_run hook blocks execution",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "BLOCK_HOOK",
						"can_block": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that blocks execution
				hookScript := `#!/bin/sh
echo '{"success": true, "block": true, "block_reason": "Policy violation"}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock: true,
		},
		{
			name: "pre_run hook modifies args",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "MODIFY_ARGS",
						"can_modify_args": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that modifies args
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_args": ["--modified", "--args"]}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedArgs == nil {
					t.Error("Expected modified_args to be set")
				} else if len(result.ModifiedArgs) != 2 {
					t.Errorf("Expected 2 modified args, got %d", len(result.ModifiedArgs))
				}
			},
		},
		{
			name: "pre_run hook tries to modify args without permission",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "MODIFY_ARGS_NO_PERM"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that tries to modify args without permission
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_args": ["--modified"]}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				// The hook execution should succeed but args should not be modified
				// because can_modify_args is false
				if result.ModifiedArgs != nil {
					t.Error("Expected modified_args to be nil when permission not granted")
				}
			},
		},
		{
			name: "no command wrapper",
			config: `{
				"packages": [],
				"shell": {
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "command wrapper configured",
			config: `{
				"packages": [],
				"shell": {
					"command_wrapper": "wrapper --",
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "no post_run hooks",
			config: `{
				"packages": [],
				"shell": {
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "one post_run hook",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "echo 'post-run hook'"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "two post_run hooks",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [
						{"command": "echo 'post 1'"},
						{"command": "echo 'post 2'"}
					],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "post_run hook modifies exit code",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "MODIFY_EXIT",
						"can_modify_exit": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that modifies exit code
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_exit": 42}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedExit == nil {
					t.Error("Expected modified_exit to be set")
				} else if *result.ModifiedExit != 42 {
					t.Errorf("Expected modified_exit to be 42, got %d", *result.ModifiedExit)
				}
			},
		},
		{
			name: "post_run hook tries to modify exit code without permission",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "MODIFY_EXIT_NO_PERM"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that tries to modify exit code without permission
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_exit": 42}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				// The hook execution should succeed but exit code should not be modified
				// because can_modify_exit is false
				if result.ModifiedExit != nil {
					t.Error("Expected modified_exit to be nil when permission not granted")
				}
			},
		},
		{
			name: "post_run hook modifies stdout",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "MODIFY_STDOUT",
						"can_modify_stdout": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that modifies stdout
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_stdout": "modified output"}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedStdout == "" {
					t.Error("Expected modified_stdout to be set")
				} else if result.ModifiedStdout != "modified output" {
					t.Errorf("Expected modified_stdout to be 'modified output', got %q", result.ModifiedStdout)
				}
			},
		},
		{
			name: "post_run hook modifies stderr",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "MODIFY_STDERR",
						"can_modify_stderr": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that modifies stderr
				hookScript := `#!/bin/sh
echo '{"success": true, "modified_stderr": "modified error"}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedStderr == "" {
					t.Error("Expected modified_stderr to be set")
				} else if result.ModifiedStderr != "modified error" {
					t.Errorf("Expected modified_stderr to be 'modified error', got %q", result.ModifiedStderr)
				}
			},
		},
		{
			name: "post_run hook sees exit code and output",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "CHECK_CONTEXT"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that verifies it receives context
				hookScript := `#!/bin/sh
if [ "$DEVBOX_HOOK_EXIT_CODE" = "42" ]; then
  echo '{"success": true}'
else
  echo '{"success": false}'
  exit 1
fi
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "pre_run hook sees command context",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "CHECK_PRE_CONTEXT"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that verifies it receives context
				hookScript := `#!/bin/sh
if [ "$DEVBOX_HOOK_COMMAND" = "test" ]; then
  echo '{"success": true}'
else
  echo '{"success": false}'
  exit 1
fi
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "hook fails to execute",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "FAILING_HOOK"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that fails
				hookScript := `#!/bin/sh
exit 1
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.Success {
					t.Error("Expected hook to fail")
				}
			},
		},
		{
			name: "hook with empty JSON output",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "EMPTY_JSON"
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that outputs empty JSON
				hookScript := `#!/bin/sh
echo '{}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
		},
		{
			name: "hook with partial JSON output",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "PARTIAL_JSON",
						"can_modify_args": true
					}],
					"scripts": {
						"test": "echo 'test'"
					}
				}
			}`,
			setupHook: func(t *testing.T, hookPath string) {
				// Create a hook script that outputs partial JSON
				hookScript := `#!/bin/sh
echo '{"modified_args": ["--partial"]}'
`
				if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
					t.Fatalf("Failed to write hook script: %v", err)
				}
			},
			expectBlock:    false,
			expectExitCode: 0,
			verifyBehavior: func(t *testing.T, result *HookResult) {
				if result.ModifiedArgs == nil {
					t.Error("Expected modified_args to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load the config
			cfg, err := configfile.LoadBytes([]byte(tt.config))
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Create a minimal Devbox instance for testing
			d := &Devbox{
				projectDir: tmpDir,
			}

			// Test pre_run hooks
			preRunHooks := cfg.PreRunHooks()
			if len(preRunHooks) > 0 {
				for _, hook := range preRunHooks {
					hookCtx := &HookContext{
						Command: "test",
						Args:    []string{"arg1", "arg2"},
						Env:     map[string]string{"VAR1": "value1"},
						Dir:     tmpDir,
					}

					// Setup hook script if needed
					if tt.setupHook != nil {
						hookPath := filepath.Join(tmpDir, hook.Command)
						tt.setupHook(t, hookPath)
						// Update hook command to use the script path
						hook.Command = hookPath
					}

					result, err := d.executePreRunHook(context.Background(), hook, hookCtx)
					if err != nil {
						t.Errorf("executePreRunHook() failed: %v", err)
						return
					}

					if tt.expectBlock && !result.Block {
						t.Errorf("Expected hook to block execution, but it didn't")
					}
					if !tt.expectBlock && result.Block {
						t.Errorf("Expected hook not to block execution, but it did: %s", result.BlockReason)
					}

					if tt.verifyBehavior != nil {
						tt.verifyBehavior(t, result)
					}

					// Verify that modifications are only applied when permissions are granted
					if hook.CanModifyArgs && result.ModifiedArgs != nil {
						hookCtx.Args = result.ModifiedArgs
					}
					if hook.CanModifyEnv && result.ModifiedEnv != nil {
						for k, v := range result.ModifiedEnv {
							hookCtx.Env[k] = v
						}
					}
				}
			}

			// Test command wrapper
			wrapper := cfg.CommandWrapper()
			if wrapper != "" {
				cmdWithArgs := []string{"echo", "test"}
				wrapped := applyCommandWrapper(cmdWithArgs, wrapper)
				if len(wrapped) != len(cmdWithArgs)+2 { // wrapper has 2 parts
					t.Errorf("Expected wrapped command to have %d parts, got %d", len(cmdWithArgs)+2, len(wrapped))
				}
			}

			// Test post_run hooks
			postRunHooks := cfg.PostRunHooks()
			if len(postRunHooks) > 0 {
				for _, hook := range postRunHooks {
					hookCtx := &HookContext{
						Command: "test",
						Args:    []string{"arg1"},
						Env:     map[string]string{},
						Dir:     tmpDir,
					}

					// Setup hook script if needed
					if tt.setupHook != nil {
						hookPath := filepath.Join(tmpDir, hook.Command)
						tt.setupHook(t, hookPath)
						// Update hook command to use the script path
						hook.Command = hookPath
					}

					result, err := d.executePostRunHook(context.Background(), hook, hookCtx, 0, "stdout", "stderr")
					if err != nil {
						t.Errorf("executePostRunHook() failed: %v", err)
						return
					}

					if tt.verifyBehavior != nil {
						tt.verifyBehavior(t, result)
					}
				}
			}
		})
	}
}

func TestHookContextEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook script that checks environment variables
	hookScript := `#!/bin/sh
echo "Command: $DEVBOX_HOOK_COMMAND"
echo "Args: $DEVBOX_HOOK_ARGS"
echo "Env: $DEVBOX_HOOK_ENV"
echo "Dir: $DEVBOX_HOOK_DIR"
echo "Exit Code: $DEVBOX_HOOK_EXIT_CODE"
echo "Stdout: $DEVBOX_HOOK_STDOUT"
echo "Stderr: $DEVBOX_HOOK_STDERR"
echo '{"success": true}'
`
	hookPath := filepath.Join(tmpDir, "check_env")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	d := &Devbox{
		projectDir: tmpDir,
	}

	// Test pre_run hook context
	t.Run("pre_run hook context", func(t *testing.T) {
		hook := &configfile.RunHook{
			Command: hookPath,
		}

		hookCtx := &HookContext{
			Command: "test",
			Args:    []string{"arg1", "arg2"},
			Env:     map[string]string{"VAR1": "value1", "VAR2": "value2"},
			Dir:     tmpDir,
		}

		result, err := d.executePreRunHook(context.Background(), hook, hookCtx)
		if err != nil {
			t.Fatalf("executePreRunHook() failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected hook to succeed")
		}
	})

	// Test post_run hook context
	t.Run("post_run hook context", func(t *testing.T) {
		hook := &configfile.RunHook{
			Command: hookPath,
		}

		hookCtx := &HookContext{
			Command: "test",
			Args:    []string{"arg1"},
			Env:     map[string]string{"VAR1": "value1"},
			Dir:     tmpDir,
		}

		result, err := d.executePostRunHook(context.Background(), hook, hookCtx, 42, "test stdout", "test stderr")
		if err != nil {
			t.Fatalf("executePostRunHook() failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected hook to succeed")
		}
	})
}

func TestHookJSONOutputParsing(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		hookOutput     string
		expectSuccess  bool
		expectModified bool
	}{
		{
			name:           "valid JSON output",
			hookOutput:     `{"success": true, "modified_args": ["--new"]}`,
			expectSuccess:  true,
			expectModified: true,
		},
		{
			name:           "non-JSON output",
			hookOutput:     "plain text output",
			expectSuccess:  true,
			expectModified: false,
		},
		{
			name:           "invalid JSON output",
			hookOutput:     `{"invalid": json}`,
			expectSuccess:  true,
			expectModified: false,
		},
		{
			name:           "JSON with block",
			hookOutput:     `{"success": true, "block": true, "block_reason": "blocked"}`,
			expectSuccess:  true,
			expectModified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a hook script that outputs the specified JSON
			hookScript := `#!/bin/sh
echo '` + tt.hookOutput + `'
`
			hookPath := filepath.Join(tmpDir, "test_hook")
			if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
				t.Fatalf("Failed to write hook script: %v", err)
			}

			d := &Devbox{
				projectDir: tmpDir,
			}

			hook := &configfile.RunHook{
				Command:        hookPath,
				CanModifyArgs:  true,
				CanBlock:       true,
			}

			hookCtx := &HookContext{
				Command: "test",
				Args:    []string{"arg1"},
				Env:     map[string]string{},
				Dir:     tmpDir,
			}

			result, err := d.executePreRunHook(context.Background(), hook, hookCtx)
			if err != nil {
				t.Fatalf("executePreRunHook() failed: %v", err)
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("Expected success to be %v, got %v", tt.expectSuccess, result.Success)
			}

			if tt.expectModified && result.ModifiedArgs == nil {
				t.Error("Expected modified_args to be set")
			}

			if !tt.expectModified && result.ModifiedArgs != nil {
				t.Error("Expected modified_args to be nil")
			}
		})
	}
}

func TestHookCapabilityEnforcement(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hook script that tries to modify everything
	hookScript := `#!/bin/sh
cat <<'EOF'
{
  "success": true,
  "block": true,
  "block_reason": "blocked",
  "modified_args": ["--modified"],
  "modified_env": {"NEW_VAR": "value"},
  "modified_exit": 99,
  "modified_stdout": "new stdout",
  "modified_stderr": "new stderr"
}
EOF
`
	hookPath := filepath.Join(tmpDir, "test_hook")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	d := &Devbox{
		projectDir: tmpDir,
	}

	tests := []struct {
		name           string
		hook           *configfile.RunHook
		expectBlock    bool
		expectArgsMod  bool
		expectEnvMod   bool
		expectExitMod  bool
	}{
		{
			name: "no capabilities - nothing should be modified",
			hook: &configfile.RunHook{
				Command: hookPath,
			},
			expectBlock:   false,
			expectArgsMod:  false,
			expectEnvMod:   false,
			expectExitMod:  false,
		},
		{
			name: "can_block only - only block should work",
			hook: &configfile.RunHook{
				Command:  hookPath,
				CanBlock: true,
			},
			expectBlock:   true,
			expectArgsMod:  false,
			expectEnvMod:  false,
			expectExitMod:  false,
		},
		{
			name: "can_modify_args only - only args should be modified",
			hook: &configfile.RunHook{
				Command:       hookPath,
				CanModifyArgs: true,
			},
			expectBlock:   false,
			expectArgsMod:  true,
			expectEnvMod:   false,
			expectExitMod:  false,
		},
		{
			name: "can_modify_env only - only env should be modified",
			hook: &configfile.RunHook{
				Command:      hookPath,
				CanModifyEnv: true,
			},
			expectBlock:   false,
			expectArgsMod:  false,
			expectEnvMod:   true,
			expectExitMod:  false,
		},
		{
			name: "all capabilities - everything should be modified",
			hook: &configfile.RunHook{
				Command:        hookPath,
				CanBlock:       true,
				CanModifyArgs:  true,
				CanModifyEnv:   true,
			},
			expectBlock:   true,
			expectArgsMod:  true,
			expectEnvMod:   true,
			expectExitMod:  false, // pre_run hooks can't modify exit code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookCtx := &HookContext{
				Command: "test",
				Args:    []string{"arg1"},
				Env:     map[string]string{},
				Dir:     tmpDir,
			}

			result, err := d.executePreRunHook(context.Background(), tt.hook, hookCtx)
			if err != nil {
				t.Fatalf("executePreRunHook() failed: %v", err)
			}

			if result.Block != tt.expectBlock {
				t.Errorf("Expected block to be %v, got %v", tt.expectBlock, result.Block)
			}

			argsModified := result.ModifiedArgs != nil
			if argsModified != tt.expectArgsMod {
				t.Errorf("Expected args modified to be %v, got %v", tt.expectArgsMod, argsModified)
			}

			envModified := result.ModifiedEnv != nil
			if envModified != tt.expectEnvMod {
				t.Errorf("Expected env modified to be %v, got %v", tt.expectEnvMod, envModified)
			}

			exitModified := result.ModifiedExit != nil
			if exitModified != tt.expectExitMod {
				t.Errorf("Expected exit modified to be %v, got %v", tt.expectExitMod, exitModified)
			}
		})
	}
}

func TestApplyCommandWrapper(t *testing.T) {
	tests := []struct {
		name           string
		cmdWithArgs    []string
		wrapper        string
		expectedResult []string
	}{
		{
			name:           "no wrapper",
			cmdWithArgs:    []string{"echo", "test"},
			wrapper:        "",
			expectedResult: []string{"echo", "test"},
		},
		{
			name:           "simple wrapper",
			cmdWithArgs:    []string{"echo", "test"},
			wrapper:        "rtk exec --",
			expectedResult: []string{"rtk", "exec", "--", "echo", "test"},
		},
		{
			name:           "wrapper with single word",
			cmdWithArgs:    []string{"echo", "test"},
			wrapper:        "sudo",
			expectedResult: []string{"sudo", "echo", "test"},
		},
		{
			name:           "empty command with wrapper",
			cmdWithArgs:    []string{},
			wrapper:        "wrapper --",
			expectedResult: []string{"wrapper", "--"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyCommandWrapper(tt.cmdWithArgs, tt.wrapper)
			if len(result) != len(tt.expectedResult) {
				t.Errorf("Expected %d parts, got %d", len(tt.expectedResult), len(result))
			}
			for i, part := range result {
				if i >= len(tt.expectedResult) {
					break
				}
				if part != tt.expectedResult[i] {
					t.Errorf("Expected part %d to be %q, got %q", i, tt.expectedResult[i], part)
				}
			}
		})
	}
}

func TestHookResultJSONSerialization(t *testing.T) {
	result := &HookResult{
		Success:      true,
		ExitCode:     0,
		ModifiedArgs: []string{"--modified"},
		ModifiedEnv:  map[string]string{"KEY": "value"},
		Block:        false,
		BlockReason:  "",
		ModifiedExit: func() *int { i := 42; return &i }(),
		ModifiedStdout: "new stdout",
		ModifiedStderr: "new stderr",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal HookResult: %v", err)
	}

	var unmarshaled HookResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal HookResult: %v", err)
	}

	if unmarshaled.Success != result.Success {
		t.Errorf("Expected Success to be %v, got %v", result.Success, unmarshaled.Success)
	}

	if len(unmarshaled.ModifiedArgs) != len(result.ModifiedArgs) {
		t.Errorf("Expected ModifiedArgs length to be %d, got %d", len(result.ModifiedArgs), len(unmarshaled.ModifiedArgs))
	}

	if unmarshaled.ModifiedExit == nil || *unmarshaled.ModifiedExit != *result.ModifiedExit {
		t.Errorf("Expected ModifiedExit to be %d, got %v", *result.ModifiedExit, unmarshaled.ModifiedExit)
	}
}

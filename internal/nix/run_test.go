// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"testing"
)

func TestRunScriptWithOutput(t *testing.T) {
	tests := []struct {
		name          string
		cmdWithArgs   string
		env           map[string]string
		capture       bool
		expectError   bool
		expectOutput  bool
		expectExitCode int
	}{
		{
			name:        "simple command without capture",
			cmdWithArgs: "echo 'hello'",
			env:         map[string]string{},
			capture:     false,
			expectError: false,
		},
		{
			name:          "simple command with capture",
			cmdWithArgs:   "echo 'hello'",
			env:           map[string]string{},
			capture:       true,
			expectError:   false,
			expectOutput:  true,
			expectExitCode: 0,
		},
		{
			name:          "command with env var",
			cmdWithArgs:   "echo $TEST_VAR",
			env:           map[string]string{"TEST_VAR": "test_value"},
			capture:       true,
			expectError:   false,
			expectOutput:  true,
			expectExitCode: 0,
		},
		{
			name:          "failing command with capture",
			cmdWithArgs:   "exit 42",
			env:           map[string]string{},
			capture:       true,
			expectError:   true,
			expectOutput:  true,
			expectExitCode: 42,
		},
		{
			name:          "command with stderr",
			cmdWithArgs:   "echo 'error' >&2",
			env:           map[string]string{},
			capture:       true,
			expectError:   false,
			expectOutput:  true,
			expectExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RunScriptWithOutput("", tt.cmdWithArgs, tt.env, tt.capture)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.expectOutput {
				if output == nil {
					t.Error("Expected output but got nil")
					return
				}
				if output.ExitCode != tt.expectExitCode {
					t.Errorf("Expected exit code %d, got %d", tt.expectExitCode, output.ExitCode)
				}
			} else if output != nil && tt.capture {
				t.Error("Expected no output when capture is false")
			}
		})
	}
}

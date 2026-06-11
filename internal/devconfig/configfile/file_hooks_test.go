// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package configfile

import (
	"testing"
)

func TestValidateRunHooks(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pre_run hook",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "echo 'pre-run hook'"
					}]
				}
			}`,
			wantErr: false,
		},
		{
			name: "valid post_run hook",
			config: `{
				"packages": [],
				"shell": {
					"post_run": [{
						"command": "echo 'post-run hook'",
						"can_modify_exit": true
					}]
				}
			}`,
			wantErr: false,
		},
		{
			name: "empty hook command",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": ""
					}]
				}
			}`,
			wantErr: true,
			errMsg:  "hook command cannot be empty",
		},
		{
			name: "post-run capabilities in pre_run hook",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "echo 'hook'",
						"can_modify_exit": true
					}]
				}
			}`,
			wantErr: true,
			errMsg:  "post-run capabilities",
		},
		{
			name: "multiple hooks",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [
						{"command": "echo 'hook 1'"},
						{"command": "echo 'hook 2'"}
					],
					"post_run": [
						{"command": "echo 'post 1'"}
					]
				}
			}`,
			wantErr: false,
		},
		{
			name: "command wrapper",
			config: `{
				"packages": [],
				"shell": {
					"command_wrapper": "rtk exec --"
				}
			}`,
			wantErr: false,
		},
		{
			name: "hook with all capabilities",
			config: `{
				"packages": [],
				"shell": {
					"pre_run": [{
						"command": "echo 'hook'",
						"can_block": true,
						"can_modify_args": true,
						"can_modify_env": true,
						"can_modify_stdin": true
					}]
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadBytes([]byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Errorf("LoadBytes() returned nil config for valid input")
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestHookAccessors(t *testing.T) {
	config := `{
		"packages": [],
		"shell": {
			"pre_run": [
				{"command": "echo 'pre1'"},
				{"command": "echo 'pre2'"}
			],
			"command_wrapper": "rtk exec --",
			"post_run": [
				{"command": "echo 'post1'"}
			]
		}
	}`

	cfg, err := LoadBytes([]byte(config))
	if err != nil {
		t.Fatalf("LoadBytes() failed: %v", err)
	}

	// Test PreRunHooks
	preRunHooks := cfg.PreRunHooks()
	if len(preRunHooks) != 2 {
		t.Errorf("Expected 2 pre_run hooks, got %d", len(preRunHooks))
	}
	if preRunHooks[0].Command != "echo 'pre1'" {
		t.Errorf("Expected first hook command to be 'echo 'pre1'', got %q", preRunHooks[0].Command)
	}

	// Test CommandWrapper
	wrapper := cfg.CommandWrapper()
	if wrapper != "rtk exec --" {
		t.Errorf("Expected command wrapper to be 'rtk exec --', got %q", wrapper)
	}

	// Test PostRunHooks
	postRunHooks := cfg.PostRunHooks()
	if len(postRunHooks) != 1 {
		t.Errorf("Expected 1 post_run hook, got %d", len(postRunHooks))
	}
	if postRunHooks[0].Command != "echo 'post1'" {
		t.Errorf("Expected first post hook command to be 'echo 'post1'', got %q", postRunHooks[0].Command)
	}
}

func TestHookAccessorsNilShell(t *testing.T) {
	config := `{
		"packages": []
	}`

	cfg, err := LoadBytes([]byte(config))
	if err != nil {
		t.Fatalf("LoadBytes() failed: %v", err)
	}

	// Test that accessors return empty/nil values when shell is not configured
	preRunHooks := cfg.PreRunHooks()
	if len(preRunHooks) != 0 {
		t.Errorf("Expected 0 pre_run hooks when shell is nil, got %d", len(preRunHooks))
	}

	wrapper := cfg.CommandWrapper()
	if wrapper != "" {
		t.Errorf("Expected empty command wrapper when shell is nil, got %q", wrapper)
	}

	postRunHooks := cfg.PostRunHooks()
	if len(postRunHooks) != 0 {
		t.Errorf("Expected 0 post_run hooks when shell is nil, got %d", len(postRunHooks))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
